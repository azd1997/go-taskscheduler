package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// 编程建议：变量集中声明在函数头部，便于阅读，另一方面也有利于GOTO的使用
	var (
		cmd *exec.Cmd
		output []byte
		err error
	)
	// 生成cmd  bash解释器参数中任务间以分号间隔
	cmd = exec.Command("C:\\Windows\\System32\\bash.exe", "-c", "sleep 5;ls -l;echo hello")

	// 执行了命令后，捕获子进程的输出（pipe()）
	if output, err = cmd.CombinedOutput(); err != nil {
		fmt.Println(err)
		return
	}

	// 打印子进程输出
	fmt.Println(string(output))
}
