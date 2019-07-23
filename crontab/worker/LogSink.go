/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/7/3 16:56
* @Description: mongoDB存储日志
***********************************************************************/

package worker

import (
	"context"
	"fmt"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"golang-taskschedule/crontab/common"
	"time"
)

type LogSink struct {
	client *mongo.Client
	logCollection *mongo.Collection
	logChan chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

func InitLogSink() (err error) {
	//连接mongoDB

	var (
		client *mongo.Client
	)

	//建立mongoDb连接
	if client, err = mongo.Connect(
		context.TODO(),
		G_config.MongodbUri,
		clientopt.ConnectTimeout(time.Duration(G_config.MongodbConnectTimeout)*time.Millisecond)); err != nil {
		return
	}

	//选择db和collection
	G_logSink = &LogSink{
		client:client,
		logCollection:client.Database("cron").Collection("log"),
		logChan:make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch,1000),
	}

	//启动一个mongoDB处理携程
	go G_logSink.writeLoop()

	return
}

//日志存储协程
func (logSink *LogSink) writeLoop() {

	var (
		jobLog *common.JobLog
		logBatch *common.LogBatch  //一般批次
		commitTimer *time.Timer
		timeoutBatch *common.LogBatch  //超时批次
	)

	for {
		select {
		case jobLog = <-logSink.logChan:
			//把这条log写到mongoDB中
			//logSink.logCollection.InsertOnce
			//但是这么做有个缺点：每次插入都需要等待mongoDB的一次请求往返，耗时可能因为网络慢而花费比较长时间
			//所以定义一个日志批次logBatch，其实就是定义一个数据结构，来对log作缓存 Protocal.go/LogBatch
			if logBatch == nil {  //先对logBatch做初始化
				logBatch = &common.LogBatch{}
				//让这个批次超时自动提交（有的时候可能日志数较少，batch村不满，那么用户久久看不到日志）
				commitTimer = time.AfterFunc(
					time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond,
					/*func() {
						//回调函数是在另外一个协程运行，这样的话writeLoop又在操作batch，这导致了batch的并发访问
						//所以这里不直接操作batch（不直接提交batch），而是向writeLoop发出一个超时通知
						logSink.autoCommitChan <- logBatch  //直接把此时的logbatch传过去，for循环的下一次就会接收到这个信号
						//但是这又有一个问题：logBatch指针会随着程序执行到下边被清空，然后下一次初始化又变了
						//所以改成返回一个回调函数并且立即执行

					}*/
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch),  //这样是把当前时刻的logBatch当成参数传给回调函数，就立即得到使用了
				)
			}
			//把新的日志追加到数组中
			logBatch.Logs = append(logBatch.Logs, jobLog)

			//如果批次满了，立即发送
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				//发送日志
				logSink.saveLogs(logBatch)
				//清空logBatch
				logBatch = nil
				//此时还需要取消定时器，考虑的情形是：在5秒内（超时时间）批次已满（达到100条）,那么需要停掉定时器
				//下一个循环又会重新创建这个定时器
				commitTimer.Stop()
				//这里还要注意一个问题：有可能100条批次满时刚好定时器超时（达到5秒），这种情况下就会导致，
				// 在这里把批次提交了一次，而在自动提交那里也会提交一次，这就会产生冲突
				// 因此，需要在超时提交的case里先做判断
			}
		case timeoutBatch = <- logSink.autoCommitChan:
			//先判断超时批次是否仍是当前批次
			if timeoutBatch != logBatch {
				continue   //timeoutBatch != logBatch只有一种解释就是logBatch，
				// 在前边那个case里已经被清空了，说明已经提交过了。就直接跳过
			}
			//把超时批次写入到mongodb中
			logSink.saveLogs(timeoutBatch)
			//把当前批次清空
			logBatch = nil
		}
	}

}

//批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {

	logSink.logCollection.InsertMany(context.TODO(), batch.Logs)
	fmt.Println("日志批次提交到mongoDB")
}

//发送日志的API,给其他地方调用，然后把日志添加到logChan。 logChan的日志会在writeLoop中提交到mongodb
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:  //将日志写入logChan队列
	default:
		//logChan队列满了(可能是因为mongodb写入比较慢)
		// 那么就把日志丢弃（这个考量是因为日志不是特别重要的）。丢弃就啥都不干就行
	}
}

//MONGODB
//打开电脑D:/mongodb/bin，启动mongod（服务器）， 再启动mongo（客户端），use cron， db.log.find()查看日志