package schedule

import (
	"cronTab/common"
	"cronTab/distribute_worker/cronJobLog"
	"cronTab/distribute_worker/lock"
	"fmt"
	"github.com/gorhill/cronexpr"
	"golang.org/x/net/context"
	"time"
)

var SchedulerClient *Scheduler

//InitScheduler 初始化任务调度器
func InitScheduler() (err error) {
	SchedulerClient = &Scheduler{
		jobEvents:     make(chan *JobEvent, 1000),
		jobPlanTable:  make(map[string]*jobSchedulePlan, 1000),
		jobExecution:  make(map[string]*jobExecutionInfo, 1000),
		jobResultChan: make(chan *JobExecuteResult, 1000),
	}
	//开启协程去执行无限读取事件通道
	go SchedulerClient.scheduleLoop()
	//启动job监听
	if err = MonitorEtcdJob(); err != nil {
		return
	}
	//启动kill 监听
	MonitorEtcdKiller()

	return nil
}

//scheduleLoop 循环
func (s *Scheduler) scheduleLoop() {
	var (
		jobEvent      *JobEvent
		err           error
		sleepDuration time.Duration
		result        *JobExecuteResult
	)
	//初始 状态 无任务 睡眠
	sleepDuration = 1 * time.Second
	scheduleTimer := time.NewTimer(sleepDuration)

	for {
		select {
		case jobEvent = <-s.jobEvents: //监听任务
			fmt.Println("监听任务列表===", jobEvent.Job.JobName, jobEvent.Job.Command)
			//将事件任务放入调度器中
			if err = s.handleJobEvent(jobEvent); err != nil {
				//记录错误 处理下一个任务
				fmt.Println("处理任务出错", err)
				continue
			}
		case result = <-s.jobResultChan: //监听任务运行结果
			s.handleExeResult(result)

		case <-scheduleTimer.C: //最新任务到期
		}
		//重新调度任务
		sleepDuration = s.TrySchedule()
		//重置计时
		scheduleTimer.Reset(sleepDuration)
	}
}

//TrySchedule 尝试调度
func (s *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		job      *jobSchedulePlan
		nearTime *time.Time
	)
	//无任务
	if len(s.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	//遍历所有任务
	for _, job = range s.jobPlanTable {
		//执行事件等于当前时间  或者是当前时间已经过了执行时间
		if time.Now().Equal(job.NextTime) || time.Now().After(job.NextTime) {
			//尝试执行任务
			s.TryStartJob(job)

			//更新下次调度时间
			job.NextTime = job.Expr.Next(time.Now())
		}
		//统计一个最近要过期的时间
		if nearTime == nil || job.NextTime.Before(*nearTime) {
			nearTime = &job.NextTime
		}

		//到下次执行最近任务调度 的间隔时间
		scheduleAfter = (*nearTime).Sub(time.Now())
	}
	return
}

func (s *Scheduler) TryStartJob(job *jobSchedulePlan) {
	var (
		jobExecuteInfo *jobExecutionInfo
		jobIsExecuting bool
	)
	//如果任务正在执行
	if jobExecuteInfo, jobIsExecuting = s.jobExecution[job.Job.JobName]; jobIsExecuting {
		return
	}

	//构建执行信息
	jobExecuteInfo = s.buildJobExecution(job)

	//保存执行信息
	s.jobExecution[job.Job.JobName] = jobExecuteInfo

	fmt.Println("任务调度列表=============", s.jobPlanTable)
	fmt.Println("正在执行的任务列表========", s.jobExecution)

	//执行任务
	exe.ExecuteJob(jobExecuteInfo)
}

//PushJobEvents 把job事件推送到 调度器的通道之中
func (s *Scheduler) PushJobEvents(jobEvent *JobEvent) {
	s.jobEvents <- jobEvent
}

//处理job 函数
func (s *Scheduler) handleJobEvent(jobEvent *JobEvent) error {
	var (
		plan  *jobSchedulePlan
		exist bool
		err   error
		jobE  *jobExecutionInfo
	)

	fmt.Println("处理job任务========", jobEvent.Job.JobName, "====", jobEvent.EventType)
	//根据不同类型的事件去处理
	switch jobEvent.EventType {
	case SAVE:
		//构建执行计划
		if plan, err = s.buildJobPlan(jobEvent.Job); err != nil {
			return err
		}
		//跟新事件的调度计划表
		s.jobPlanTable[jobEvent.Job.JobName] = plan
		fmt.Println("任务构建的调度列表============", s.jobPlanTable)

	case DELETE:
		//调度器中如果存在
		if plan, exist = s.jobPlanTable[jobEvent.Job.JobName]; exist {
			//删除任务
			delete(s.jobPlanTable, jobEvent.Job.JobName)
		}
	case KILL:
		//取消掉command 执行
		//判断任务是否在执行
		if jobE, exist = s.jobExecution[jobEvent.Job.JobName]; exist {
			jobE.CancelFunc() //处罚中断杀死
		}

	}
	return nil
}

func (s *Scheduler) handleExeResult(result *JobExecuteResult) {
	//将执行的结果保存到mongodb
	//生成日志 非锁占用日志
	if result.err != lock.DLockAlreadyExists {
		log := &cronJobLog.JobLog{
			JobName:      result.exeInfo.Job.JobName,
			Command:      result.exeInfo.Job.Command,
			OutPut:       string(result.outPut),
			PlanTime:     result.exeInfo.PlanScheduleTime.UnixMilli(),
			ScheduleTime: result.exeInfo.RealScheduleTime.UnixMilli(),
			StartTime:    result.startTime.UnixMilli(),
			EndTime:      result.endTime.UnixMilli(),
		}
		//没错误 则err 指针为空 需做判断
		if result.err != nil {
			log.Err = ""
		}

		//将日子推送到mongo 定义到buffer中
		go cronJobLog.MLog.PushLog(log)
	}

	//删除执行状态
	delete(s.jobExecution, result.exeInfo.Job.JobName)
}

func (s *Scheduler) buildJobPlan(job common.Job) (*jobSchedulePlan, error) {
	//expr 表达式的解析
	var (
		expr *cronexpr.Expression
		err  error
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}

	//构建任务执行计划
	jobPlan := &jobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return jobPlan, nil
}

//buildJobExecution 构建job 执行信息
func (s *Scheduler) buildJobExecution(plan *jobSchedulePlan) *jobExecutionInfo {
	ctx, cancel := context.WithCancel(context.Background())
	return &jobExecutionInfo{
		Job:              plan.Job,
		PlanScheduleTime: plan.NextTime,
		RealScheduleTime: time.Now(),
		Ctx:              ctx,
		CancelFunc:       cancel,
	}
}

func (s *Scheduler) PushJobExeResult(result *JobExecuteResult) {
	s.jobResultChan <- result
}
