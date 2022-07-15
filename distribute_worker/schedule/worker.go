package schedule

import (
	"cronTab/common"
	"cronTab/etcd"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"strings"
)

//MonitorEtcdJob 监听etcd job 的变化
func MonitorEtcdJob() (err error) {
	var (
		getResp *clientv3.GetResponse
		job     common.Job
	)

	//1 get一下/cron/jobs/目录下的所有任务，并且获得当前集群的reversion
	if getResp, err = etcd.EtcdKvClient.Get(context.TODO(), common.EtcdJobKey, clientv3.WithPrefix()); err != nil {
		return
	}

	watchStartVersion := getResp.Header.Revision + 1

	//遍历jobs
	for _, kvPair := range getResp.Kvs {
		fmt.Println("获取任务=======", string(kvPair.Key))
		if err = jsoniter.Unmarshal(kvPair.Value, &job); err != nil {
			return
		}
		//构建一个任务事件
		jobEvent := BuildJobEvent(SAVE, job)

		fmt.Println("获取后的jobEvent-----", jobEvent)
		//同步给scheduler
		SchedulerClient.PushJobEvents(jobEvent)
	}

	//2，启动一个监听协程 从该reversion向后监听变化事件
	go func() {
		if err = watchJobs(watchStartVersion); err != nil {
			return
		}
	}()

	return nil
}

func watchJobs(startVersion int64) error {
	var (
		watchChan clientv3.WatchChan
		job       common.Job
		err       error
		jobName   string
		jobEvent  *JobEvent
	)

	watchChan = etcd.EtcdWatchClient.Watch(context.TODO(), common.EtcdJobKey, clientv3.WithPrefix(), clientv3.WithRev(startVersion))

	//处理监听事件
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			fmt.Println(event.Kv)
			switch event.Type {
			case mvccpb.PUT:
				//反序列化新job 推送给 scheduler 调度
				if err = jsoniter.Unmarshal(event.Kv.Value, &job); err != nil {
					return err
				}
				//构造一个更新Event
				jobEvent = BuildJobEvent(SAVE, job)

			case mvccpb.DELETE:
				//从key中提取任务名
				jobName = strings.TrimPrefix(string(event.Kv.Key), common.EtcdJobKey)
				job = common.Job{
					JobName: jobName,
				}
				//构造一个删除Event
				jobEvent = BuildJobEvent(DELETE, job)
			}
			SchedulerClient.PushJobEvents(jobEvent)
		}
	}

	return nil
}

//MonitorEtcdKiller 监听etcd killer 的变化
func MonitorEtcdKiller() {
	//开启协程去监听/cron/kill/ 下的事件
	go watchKiller()
}

func watchKiller() {
	var (
		watchChan clientv3.WatchChan
		jobName   string
		job       common.Job
	)
	//监听目录
	watchChan = etcd.EtcdWatchClient.Watch(context.TODO(), common.EtcdKillKey, clientv3.WithPrefix())

	//处理监听事件
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {

			switch event.Type {
			case mvccpb.PUT: //杀死某个事件
				//从key中提取任务名
				jobName = strings.TrimPrefix(string(event.Kv.Key), common.EtcdKillKey)
				job = common.Job{
					JobName: jobName,
				}
				//构建强杀事件
				jobEvent := BuildJobEvent(KILL, job)
				//事件推给schedule

				SchedulerClient.PushJobEvents(jobEvent)
			}
		}
	}

}
