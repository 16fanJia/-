package router

import (
	"cronTab/master/handler"
	"github.com/gin-gonic/gin"
)

//RegisterMasterApi 注册master 的API
func RegisterMasterApi(r *gin.Engine) {
	cronGroup := r.Group("/cron")
	{
		cronGroup.POST("/save_job", handler.SaveJobToEtcd)
		cronGroup.DELETE("/delete_job/:jobName", handler.DeleteJobFromEtcd)
		cronGroup.GET("/list_job", handler.ListJobs)
		cronGroup.POST("/kill_job/:jobName", handler.KillJob)
		cronGroup.GET("/job_log/:jobName", handler.GetJobLog)
		cronGroup.GET("/worker_list", handler.GetWorkerList)
	}

}
