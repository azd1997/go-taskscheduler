/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/23 21:53
* @Description: 主文件，入口
***********************************************************************/

package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"golang-taskschedule/crontab/common"
	"golang-taskschedule/crontab/master"
	"runtime"
	"time"
)

var (
	configFile string  // 配置文件路径
)

// 解析命令行参数，将命令行参数传递给configFile变量
func initArgs() {
	// master -config ./master.json

	flag.StringVar(&configFile, "config", common.MASTER_CONFIG_DEFAULT_DIR, "指定master.json加载路径")

	flag.Parse()
}

func initEnv() {
	// 配置golang使用的线程数量，使之与CPU数量相等
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	var (
		err error
	)

	// 1.初始化命令行参数
	initArgs()

	// 2.初始化线程
		// 因为golang是个多线程的执行，语言本身支持多线程。而开发时用的是协程，协程会
		// 被调度到线程上，线程是一个操作系统的概念。 要发挥golang多线程优势，就要让其
		// 线程设置为计算机CPU核心数量。
	initEnv()

	// 3.加载配置文件
		// InitConfig将配置文件中配置读出，存入G_config，只需要访问G_config就可以使用配置
	if err = master.InitConfig(configFile); err != nil {
		goto ERR
	}

	// 4.启动任务管理器（和ETCD连接）
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	// 2.启动api http 服务
		// master要通过http向WEB管理应用提供任务的增删改查
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	//为了避免APIServer协程工作时主协程退出，在这里写睡眠循环
	for {
		time.Sleep(1*time.Second)
	}

	//正常退出
	return

ERR:
	log.Error(err)
}