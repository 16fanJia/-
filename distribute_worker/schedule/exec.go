package schedule

import (
	"cronTab/distribute_worker/lock"
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

//Executor 任务执行器
type Executor struct{}

//JobExecuteResult 执行输出返回数据结构体
type JobExecuteResult struct {
	exeInfo   *jobExecutionInfo //执行状态
	outPut    []byte            //脚本输出
	err       error             //脚本错误
	startTime time.Time         //启动时间
	endTime   time.Time         //结束时间
}

var exe *Executor

const (
	Lower  = time.Duration(150) * time.Millisecond //最低
	Higher = time.Duration(300) * time.Millisecond //最高
)

func init() {
	exe = &Executor{}
}

func (ex *Executor) ExecuteJob(jobExecuteInfo *jobExecutionInfo) {
	go func() {
		var (
			outPut []byte
			err    error
		)

		fmt.Println("进入执行=======", jobExecuteInfo.Job.JobName)
		//随机睡眠 保证锁公平竞争
		time.Sleep(randTimeDuration(Lower, Higher))

		dLock := lock.NewDLock(jobExecuteInfo.Job.JobName)
		//尝试获取分布式锁
		if err = dLock.TryDLock(); err != nil {
			dLock.UnLock() //释放锁

			resultErr := &JobExecuteResult{
				exeInfo: jobExecuteInfo,
				err:     err,
			}
			//错误返回给调度者
			SchedulerClient.PushJobExeResult(resultErr)
			return
		}

		result := &JobExecuteResult{
			exeInfo:   jobExecuteInfo,
			startTime: time.Now(), //任务开始时间
		}

		cmd := exec.CommandContext(jobExecuteInfo.Ctx, "bash", "-c", jobExecuteInfo.Job.Command)

		//执行命令 捕获子进程的输出（pipe）
		outPut, err = cmd.CombinedOutput()

		result.err = err
		result.outPut = outPut
		result.endTime = time.Now() //任务结束时间

		//取消锁
		dLock.UnLock()

		fmt.Printf("%s 执行任务 :%s", jobExecuteInfo.Job.JobName, string(outPut))
		//执行结果放入调度者通道
		SchedulerClient.PushJobExeResult(result)
	}()
}

func randTimeDuration(lower, higher time.Duration) time.Duration {
	//随机数种子
	rand.Seed(time.Now().Unix())
	//150-300 ns 之间随机生成一个数
	num := rand.Int63n(higher.Nanoseconds()-lower.Nanoseconds()) + lower.Nanoseconds()

	return time.Duration(num) * time.Nanosecond
}
