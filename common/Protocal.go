/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/24 13:49
* @Description: The file is for
***********************************************************************/

package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name string `json:"name"`	//任务名
	Command string `json:"command"` //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
}

//任务调度计划
type JobSchedulePlan struct {
	Job *Job  //要调度的任务信息
	Expr *cronexpr.Expression //解析好的cron表达式
	NextTime time.Time //下次调度时间
}

//任务执行状态
type JobExecuteInfo struct {
	Job *Job //任务信息
	PlanTime time.Time //计划执行时间
	RealTime time.Time //实际执行时间
	CancelCtx context.Context  //用于取消任务
	CancelFunc context.CancelFunc  //用于取消任务
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行状态
	Output []byte  //脚本输出
	Err error  //脚本执行错误原因
	StartTime time.Time  //脚本真实启动时间
	EndTime time.Time  //脚本结束时间

}

//任务执行日志,存入MONGODB
type JobLog struct {
	JobName string `bson:"jobName"`
	Command string `bson:"command"`
	Err string `bson:"err"`
	Output string `bson:"output"`
	PlanTime int64 `bson:"planTime"`  //计划开始时间
	ScheduleTime int64 `bson:"scheduleTime"`  //实际调度时间，和计划调度时间应该差在us级别
	StartTime int64 `bson:"startTime"`  //任务执行开始
	EndTime int64 `bson:"endTime"`  //任务执行结束
}

//日志批次（多条日志，日志缓存）
type LogBatch struct {
	Logs []interface{}  //多条日志
}

//HTTP接口应答
type Response struct {
	Errno int `json:"errno"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

//任务更新/删除事件
type JobEvent struct {
	EventType int	//SAVE DELETE
	Job *Job
}

//应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	//1.定义一个response
	var (
		response Response
	)

	response.Errno = errno
	response.Msg = msg
	response.Data = data

	// 2.json序列化
	resp, err = json.Marshal(response)

	return
}

//反序列化Job
func UnpackJob(packedJob []byte) (job *Job, err error) {

	job = &Job{}
	if err = json.Unmarshal(packedJob, job); err != nil {
		return
	}

	return
}

//从etcd的key中提取任务名
//ex. /cron/jobs/job1 -> job1
func ExtractJobName(jobKey string) (string) {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

//从etcd的key中提取任务名
//ex. /cron/killer/job1 -> job1
func ExtractKillerName(killerKey string) (string) {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)

}

//任务变化事件有两种：更新任务事件 删除任务事件
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	jobEvent = &JobEvent{eventType, job}
	return
}

//构造执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {

	var (
		expr *cronexpr.Expression

	)

	//解析cron表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}

	//生成任务调度计划对象
	jobSchedulePlan = &JobSchedulePlan{
		Job:job,
		Expr:expr,
		NextTime:expr.Next(time.Now()),
	}
	return
}

//构造执行状态
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:jobSchedulePlan.Job,
		PlanTime:jobSchedulePlan.NextTime,  //计算的调度时间
		RealTime:time.Now(),  //当任务执行时，构建此信息，那么这里存的就是实际执行时间
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())

	return
}