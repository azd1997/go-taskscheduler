/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/7/1 17:12
* @Description: 任务调度
***********************************************************************/

package worker

import (
	"fmt"
	"golang-taskschedule/crontab/common"
	"time"
)

type Scheduler struct {
	jobEventChan chan *common.JobEvent	// etcd任务事件队列，用来和watch协程通信
	jobPlanTable map[string]*common.JobSchedulePlan  //任务调度计划表
	jobExecuteTable map[string]*common.JobExecuteInfo  //任务执行状态表
	jobResultChan chan *common.JobExecuteResult	// 任务执行结果传递通道
}

var (
	G_scheduler *Scheduler
)

//初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:make(chan *common.JobEvent, 1000),
		jobPlanTable:make(map[string]*common.JobSchedulePlan),
		jobExecuteTable:make(map[string]*common.JobExecuteInfo),
		jobResultChan:make(chan *common.JobExecuteResult, 1000),
	}

	//启动调度协程
	go G_scheduler.schedulerLoop()

	return
}

//调度协程
func (scheduler *Scheduler) schedulerLoop() {

	var (
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult *common.JobExecuteResult
	)

	//初始化一次，得到一个下次调度的时间，不调度的时候就睡觉
	scheduleAfter = scheduler.TrySchedule()

	//调度的延时定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//定时任务
	for {
		select {
		case jobEvent = <- scheduler.jobEventChan: //监听任务变化事件
			//对变化的任务事件作增删改查
			scheduler.handleJobEvent(jobEvent)
		case <- scheduleTimer.C:  //最近的任务到期了，我们需要重新调度
		case jobResult = <- scheduler.jobResultChan:  //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重置定时器(调度间隔)
		scheduleTimer.Reset(scheduleAfter)  //重置的原因是因为上边可能是定时器到时间了，也有可能是有新的任务变化，这时需要重置重新定时

	}
}


func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {

	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted bool
		err error
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
	)

	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:  //保存任务事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return  //err!=nil时不要这个任务了
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan   //存入计划表
	case common.JOB_EVENT_DELETE: //删除任务事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobSchedulePlan.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		//取消掉Command执行
		//首先判断任务是否在执行中
		//fmt.Println("强杀事件￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥")
		if jobExecuteInfo, jobExecuting = scheduler.jobExecuteTable[jobEvent.Job.Name]; jobExecuting {
			//fmt.Println("强杀事件成功11111￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥￥")
			jobExecuteInfo.CancelFunc()  //触发Command杀死shell子进程，任务得到退出
		}
	}
}

//推送任务事件到通道中
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {

	var (
		jobPlan *common.JobSchedulePlan
		now time.Time
		nearestTime *time.Time
	)

	//如果任务表为空，随便睡眠多久
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1*time.Second
		return
	}

	//如果有任务，就往下执行，找到最近的任务，计算出睡眠时间（可以不睡眠，一直扫描，但是很浪费）

	now = time.Now()

	//1.遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {	//任务过期(现在已经到了任务计划时间或者过了计划时间)
			//TODO：尝试执行任务
			//fmt.Println("执行任务", jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now)  //重新计算并更新下次任务执行时间
		}

		//统计最近一个将要过期的任务时间
		if nearestTime == nil || jobPlan.NextTime.Before(*nearestTime) {	//寻找现在所有任务当中（都是计划任务）最近的那一个任务时间
			nearestTime = &jobPlan.NextTime
		}

	}

	scheduleAfter = (*nearestTime).Sub(now)	//也就是在scheduleAfter后有第一个计划任务应当执行

	return
}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobSchedulePlan *common.JobSchedulePlan) {
	// 调度和执行是两回事。
	// 执行的任务可能运行很久，比如说需要运行1分钟，1分钟会调度60次，但只能执行一次，防止并发

	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
	)

	fmt.Println("**************************************************")

	//如果任务正在执行（那么任务执行信息会在执行状态表）,则跳过
	if jobExecuteInfo, jobExecuting = scheduler.jobExecuteTable[jobSchedulePlan.Job.Name]; jobExecuting {
		fmt.Println("任务已经在执行，跳过执行", jobSchedulePlan.Job.Name)
		return
	}

	//构建执行状态信息并保存
	jobExecuteInfo = common.BuildJobExecuteInfo(jobSchedulePlan)
	scheduler.jobExecuteTable[jobExecuteInfo.Job.Name] = jobExecuteInfo

	//执行任务，也就是执行shell命令
	fmt.Println("执行shell任务:", jobExecuteInfo.Job.Name)
	fmt.Println("计划开始时间:", jobExecuteInfo.PlanTime)
	fmt.Println("实际开始时间:", jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)

	//任务执行完之后还需要从执行状态表删除对应信息
}

//回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}


func (scheduler *Scheduler) handleJobResult(jobResult *common.JobExecuteResult) {

	var (
		jobLog *common.JobLog
	)

	//删除执行状态
	delete(scheduler.jobExecuteTable, jobResult.ExecuteInfo.Job.Name)
	fmt.Println("**************************************************")
	fmt.Println("任务执行完毕:", jobResult.ExecuteInfo.Job.Name)
	fmt.Println("任务输出：", string(jobResult.Output))
	fmt.Println("任务错误信息：", jobResult.Err)
	fmt.Println("任务开始时间：", jobResult.StartTime)
	fmt.Println("任务结束时间：", jobResult.EndTime)

	//生成执行日志
	if jobResult.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		//ERR_LOCK...是自定义的错误，额外处理，而且在集群部署中这个ERR是正常的

		jobLog = &common.JobLog{
			JobName:jobResult.ExecuteInfo.Job.Name,
			Command:jobResult.ExecuteInfo.Job.Command,
			Output:string(jobResult.Output),
			PlanTime:jobResult.ExecuteInfo.PlanTime.UnixNano()/1000/1000,
			ScheduleTime:jobResult.ExecuteInfo.RealTime.UnixNano()/1000/1000,
			StartTime:jobResult.StartTime.UnixNano()/1000/1000,
			EndTime:jobResult.EndTime.UnixNano()/1000/1000,

		}
		if jobResult.Err != nil {
			jobLog.Err = jobResult.Err.Error()
		} else {
			jobLog.Err = ""
		}

		//TODO：将日志存入MONGODB,但是不能在这里进行这个操作
		//原因是：这是在scheduleLoop中调用的，会影响任务调度的精准度，甚至终止了调度
		//所以我们需要在另外一个协程去处理日志存储
		G_logSink.Append(jobLog)

	}

	//todo:将日志cunruMONGODB

}




