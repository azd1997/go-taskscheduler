package main

// linux crontab
// cron 表达式基本格式
// *	*	*	*	*	Command
// 分	时	日	月	星期	Shell命令
//0-59 0-23 1-31 1-12 0-7

// cronexpr 支持7个粒度，增加 秒、年

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"time"
)

func main()  {
	var (
		expr     *cronexpr.Expression
		err      error
		now      time.Time
		nextTime time.Time
	)
	//// 每分钟执行一次
	//if expr, err = cronexpr.Parse("* * * * *"); err != nil {
	//	fmt.Println(err)
	//	return
	//}

	// 每5秒执行一次
	if expr, err = cronexpr.Parse("*/5 * * * * * *"); err != nil {
		fmt.Println(err)
		return
	}

	// 计算下次任务调度时间
	now = time.Now()
	nextTime = expr.Next(now)
	//fmt.Println(now, nextTime)

	// 等待定时器超时
	time.AfterFunc(nextTime.Sub(now), func(){
		fmt.Println("被调度了：", nextTime)
	})

	time.Sleep(5*time.Second)

}
