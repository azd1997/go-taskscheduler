/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/7/1 20:11
* @Description: 任务执行器
***********************************************************************/

package worker

import (
	"golang-taskschedule/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

type Executor struct {

}

var (
	G_executor *Executor
)

//初始化执行器
func InitExecutor() (err error) {

	G_executor = &Executor{

	}

	return
}

//执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	//启动协程去处理执行任务的逻辑
	go func() {

		var (
			cmd *exec.Cmd
			output []byte
			err error
			result *common.JobExecuteResult
			jobLock *JobLock
		)

		//任务结果
		result = &common.JobExecuteResult{
			ExecuteInfo:info,
			Output:make([]byte, 0),
		}

		//首先获取分布式锁，抢到了锁，才执行，没抢到不执行任务，并且返回相关信息
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)

		//记录任务开始时间
		result.StartTime = time.Now()

		//抢锁之前随机睡眠0~1s，目的是牺牲一些调度时间准确性，来弥补不同机器时钟的差异（一般在微秒级）
		//这样能使机器之间抢锁比较公平，比较均匀 不会有倾斜性
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		//尝试抢锁
		err = jobLock.TryLock()
		defer jobLock.UnLock()

		if err != nil {
			result.Err = err //上锁失败
			result.EndTime = time.Now()
		} else {

			//执行shell命令
			cmd = exec.CommandContext(info.CancelCtx, "C:\\Windows\\System32\\bash.exe", "-c", info.Job.Command)

			//执行并捕获输出
			output, err = cmd.CombinedOutput()

			//记录结束时间
			result.EndTime = time.Now()
			//任务执行结果返回给调度器，
			result.Output = output
			result.Err = err
		}



		G_scheduler.PushJobResult(result)
		//调度器监听这个通道，如果收到了结果，那么就删除任务执行状态


	}()
}

