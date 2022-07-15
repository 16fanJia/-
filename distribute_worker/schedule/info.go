package schedule

import (
	"context"
	"cronTab/common"
	"github.com/gorhill/cronexpr"
	"time"
)

//Scheduler 任务调度
type Scheduler struct {
	jobEvents     chan *JobEvent               //放入事件
	jobPlanTable  map[string]*jobSchedulePlan  //任务调度计划表
	jobExecution  map[string]*jobExecutionInfo //job任务执行信息
	jobResultChan chan *JobExecuteResult       //任务结果通道
}

//jobSchedulePlan 任务调度计划
type jobSchedulePlan struct {
	Job      common.Job           //要调度的任务信息
	Expr     *cronexpr.Expression //解析好的cronexpr 表达式
	NextTime time.Time            //job下次执行事件
}

//jobExecutionInfo 任务执行信息
type jobExecutionInfo struct {
	Job              common.Job
	PlanScheduleTime time.Time          //理论调度时间
	RealScheduleTime time.Time          //实际调度时间
	Ctx              context.Context    //用于的context
	CancelFunc       context.CancelFunc //取消任务函数
}
