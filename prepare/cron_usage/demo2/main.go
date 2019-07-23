package main

// 多个任务的调度

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"time"
)

// 代表一个任务
type CronJob struct {
	expr *cronexpr.Expression
	nextTime time.Time	// expr.Next(now)
}

func main()  {

	// 需要有一个调度协程，定时检查所有的CRon任务，谁过期了执行谁

	var (
		cronJob *CronJob
		expr *cronexpr.Expression
		now time.Time
		scheduleTable map[string]*CronJob
	)

	now = time.Now()
	scheduleTable = make(map[string]*CronJob)

	// 1. 定义两个CRonJob
	expr = cronexpr.MustParse("*/5 * * * * * *")
	cronJob = &CronJob{
		expr:expr,
		nextTime:expr.Next(now),
	}

	// 将任务注册到调度表
	scheduleTable["job1"] = cronJob

	expr = cronexpr.MustParse("*/5 * * * * * *")
	cronJob = &CronJob{
		expr:expr,
		nextTime:expr.Next(now),
	}

	// 将任务注册到调度表
	scheduleTable["job2"] = cronJob

	// 启动一个调度协程
	go func() {
		var (
			jobName string
			cronJob *CronJob
			now time.Time
		)

		// 定时检查任务调度表
		for {
			now = time.Now()

			for jobName, cronJob = range scheduleTable {
				// 判断是否过期
				if cronJob.nextTime.Before(now) || cronJob.nextTime.Equal(now) {
					// 启动一个协程，执行这个任务
					go func(name string) {
						fmt.Println("执行任务：", name)
					}(jobName)		//尽管这里直接传jobName（使用闭包）也行，但建议新建一个变量
					//go func() {
					//	fmt.Println(jobName)
					//}()

					// 计算下一次调度时间
					cronJob.nextTime = cronJob.expr.Next(now)
					fmt.Println("下次执行时间：", cronJob.nextTime)
				}
			}

			// 不建议一直检查，耗费CPU资源
			// 因为任务调度粒度为秒级，所以可以设置睡眠
			//time.Sleep(100*time.Millisecond)
			//通过等待通道信号，也是一种另类实现
			select {
			case <- time.NewTimer(100*time.Millisecond).C:	//将在100ms后可读，返回
			}
		}

	}()

	// 避免主协程退出，让其睡100秒
	time.Sleep(100*time.Second)
}
