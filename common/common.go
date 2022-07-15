package common

//定义全局可用的etcd key值
const (
	EtcdJobKey    = "/cron/jobs/"
	EtcdKillKey   = "/cron/kill/"
	EtcdWorkerKey = "/cron/workers/"
)

//Job 任务结构体
type Job struct {
	JobName  string `bson:"job_name" json:"job_name"`   //任务名
	Command  string `bson:"command" json:"command"`     //shell 命令
	CronExpr string `bson:"cron_expr" json:"cron_expr"` //cron表达式
}
