package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type result struct {
	err error
	output []byte
}

func main() {
	// 执行一个cmd，让它在一个协程里执行2秒：sleep 2；echo hello
	// 1秒时杀死cmd

	var (
		ctx context.Context
		cancelFunc context.CancelFunc
		cmd *exec.Cmd
		resultChan chan *result
		res *result
	)

	resultChan = make(chan *result, 1000)

	// context: chan byte
	// cancelFunc: close(chan byte)
	ctx, cancelFunc = context.WithCancel(context.TODO())

	// 开辟协程执行任务
	go func() {
		var (
			output []byte
			err error
		)
		cmd = exec.CommandContext(ctx, "C:\\Windows\\System32\\bash.exe", "-c", "sleep 2;echo hello;")
		// 执行任务捕获输出
		output, err = cmd.CombinedOutput()

		// 把任务结果通过通道传递给主协程
		resultChan <- &result{
			err:err,
			output:output,
		}

	}()

	// 与此同时继续往下走
	time.Sleep(1*time.Second)
	// 取消上下文
	cancelFunc()

	// 主协程中等待一秒就取消任务，然而子协程中要睡2秒才执行echo hello，显然hello是输出不了的

	// 主协程等待子协程退出，并打印任务执行结果
	res = <- resultChan
	fmt.Println(res.err)		// exit status 1
	fmt.Println(string(res.output))		// 空
}