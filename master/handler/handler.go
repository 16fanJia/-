package handler

import (
	"cronTab/common"
	"cronTab/etcd"
	"cronTab/master/cronlog"
	"cronTab/master/response"
	"cronTab/master/response/code"
	"cronTab/mongodb"
	"fmt"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"net/http"
)

//SaveJobParams 保存job 参数
type SaveJobParams struct {
	JobName  string `json:"job_name" form:"job_name"`
	Command  string `json:"command" form:"command"`
	CronExpr string `json:"cron_expr" form:"cron_expr"`
}

//SaveJobToEtcd 保存job 接口
func SaveJobToEtcd(c *gin.Context) {
	var (
		Params      SaveJobParams
		err         error
		valueString string
		key         string
		putResp     *clientv3.PutResponse
		oldJob      common.Job
	)

	if err = c.ShouldBind(&Params); err != nil {
		//请求成功 但数据错误
		response.Fail(c, http.StatusOK, code.ParamsError, err)
		return
	}
	fmt.Println(Params.CronExpr)

	record := &common.Job{
		JobName:  Params.JobName,
		Command:  Params.Command,
		CronExpr: Params.CronExpr,
	}
	//序列化为字符串
	if valueString, err = jsoniter.MarshalToString(record); err != nil {
		response.Fail(c, http.StatusOK, code.DataParsingFailed, err)
		return
	}

	//存入etcd
	key = common.EtcdJobKey + Params.JobName

	if putResp, err = etcd.EtcdKvClient.Put(context.TODO(), key, valueString, clientv3.WithPrevKV()); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdPutFailed, err)
		return
	}
	//如果是更新 就可以给前端返回更新之前的值
	if putResp.PrevKv != nil {
		if err = jsoniter.Unmarshal(putResp.PrevKv.Value, &oldJob); err != nil {
			response.Fail(c, http.StatusOK, code.DataParsingFailed, err)
			return
		}
	}

	response.Success(c, code.Ok, oldJob)
}

//DeleteJobFromEtcd 删除job 接口
func DeleteJobFromEtcd(c *gin.Context) {
	var (
		jobName string
		key     string
		delResp *clientv3.DeleteResponse
		err     error
		oldJob  common.Job
	)
	jobName = c.Param("jobName")

	key = common.EtcdJobKey + jobName

	//从etcd 删除数据
	if delResp, err = etcd.EtcdKvClient.Delete(context.TODO(), key, clientv3.WithPrevKV()); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdDelFailed, err)
		return
	}
	//返回删除前的数据
	if delResp.Deleted != 0 {
		//旧值解析返回
		if err = jsoniter.Unmarshal(delResp.PrevKvs[0].Value, &oldJob); err != nil {
			response.Fail(c, http.StatusOK, code.DataParsingFailed, err)
			return
		}
	}

	response.Success(c, code.Ok, oldJob)
}

//ListJobs 列举任务列表
func ListJobs(c *gin.Context) {
	var (
		err     error
		getResp *clientv3.GetResponse
		job     common.Job
	)

	if getResp, err = etcd.EtcdKvClient.Get(context.TODO(), common.EtcdJobKey, clientv3.WithPrefix()); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdGetFailed, err)
		return
	}
	//创建返回数据的数组
	jobList := make([]common.Job, 0, getResp.Count)

	//遍历结果
	for _, kvPair := range getResp.Kvs {
		if err = jsoniter.Unmarshal(kvPair.Value, &job); err != nil {
			response.Fail(c, http.StatusOK, code.DataParsingFailed, err)
			return
		}
		jobList = append(jobList, job)
	}

	response.Success(c, code.Ok, jobList)

}

//KillJob 杀死任务
func KillJob(c *gin.Context) {
	var (
		jonName        string
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
	)
	jonName = c.Param("jobName")

	killKey := common.EtcdKillKey + jonName

	//给killKey 设置租约 1
	if leaseGrantResp, err = etcd.EtcdLeaseClient.Grant(context.TODO(), 1); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdCreateLeaseFailed, err)
		return
	}
	//租约ID
	leaseId := leaseGrantResp.ID

	//租约关联到killKey 让worker 监听到这个任务已经被杀死
	if _, err = etcd.EtcdKvClient.Put(context.TODO(), killKey, "", clientv3.WithLease(leaseId)); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdPutFailed, err)
		return
	}
	response.Success(c, code.Ok, nil)
}

type GetJobLogParams struct {
	JobName string `json:"job_name" form:"job_name"`
	Size    int    `json:"size" form:"size,default=1"`  //每一页的大小
	Page    int    `json:"page" form:"page,default=15"` //页数
}

//GetJobLog 获取job 运行日志
func GetJobLog(c *gin.Context) {
	var (
		params GetJobLogParams
		err    error
		cur    *mongo.Cursor
	)
	if err = c.ShouldBind(&params); err != nil {
		response.Fail(c, http.StatusOK, code.ParamsError, err)
		return
	}

	filter := bson.D{{"job_name", params.JobName}}

	op := &options.FindOptions{}
	op.SetSort(bson.D{{"start_time", "-1"}}) //根据开始时间倒排
	op.SetLimit(int64(params.Size))
	op.SetSkip(int64((params.Page - 1) * params.Size))

	cur, err = mongodb.MongoClient.Database("cron").
		Collection("log").
		Find(context.Background(), filter, op)
	if err != nil {
		response.Fail(c, http.StatusOK, code.MongodbSearchFailed, err)
		return
	}
	//查询数据
	result := make([]cronlog.CronJobLog, 0, params.Size)

	if err = cur.All(context.TODO(), &result); err != nil {
		response.Fail(c, http.StatusOK, code.DataParsingFailed, err)
		return
	}

	response.Success(c, code.Ok, result)
}

// GetWorkerList 获取worker列表
func GetWorkerList(c *gin.Context) {
	var (
		err     error
		getResp *clientv3.GetResponse
	)

	if getResp, err = etcd.EtcdKvClient.Get(context.TODO(), common.EtcdWorkerKey, clientv3.WithPrefix()); err != nil {
		response.Fail(c, http.StatusOK, code.EtcdGetFailed, err)
		return
	}
	var workerList []string
	for _, workerPair := range getResp.Kvs {
		workerList = append(workerList, string(workerPair.Value))
	}

	response.Success(c, code.Ok, workerList)

}
