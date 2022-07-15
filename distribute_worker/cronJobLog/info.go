package cronJobLog

//JobLog 日志结构
type JobLog struct {
	JobName      string `json:"job_name" bson:"job_name"`           //任务名
	Command      string `json:"command" bson:"command"`             //命令
	Err          string `json:"err" bson:"err"`                     //错误
	OutPut       string `json:"out_put" bson:"out_put"`             //输出
	PlanTime     int64  `json:"plan_time" bson:"plan_time"`         //计划调度事件
	ScheduleTime int64  `json:"schedule_time" bson:"schedule_time"` //实际调度时间
	StartTime    int64  `json:"start_time" bson:"start_time"`       //任务开始时间
	EndTime      int64  `json:"end_time" bson:"end_time"`           //任务结束时间
}
