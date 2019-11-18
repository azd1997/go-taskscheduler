/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/24 13:53
* @Description: The file is for
***********************************************************************/

package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"golang-taskschedule/crontab/common"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
	watcher clientv3.Watcher
}

//定义一个单例
var (
	G_jobMgr *JobMgr
)

// 初始化任务管理器，主要是做好和ETCD的连接
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
		watcher clientv3.Watcher
	)

	//初始化ETCD Client配置
	config = clientv3.Config{
		Endpoints: G_config.EtcdEndpoints,	//集群节点地址
		DialTimeout:time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,	//连接超时时间
	}

	//和ETCD服务器建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 得到KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	watcher = clientv3.NewWatcher(client)

	//赋值单例
	G_jobMgr = &JobMgr{
		client:client,
		kv:kv,
		lease:lease,
		watcher:watcher,
	}

	//启动任务监听
	G_jobMgr.watchJobs()

	//启动任务强杀目录监听
	G_jobMgr.watchKiller()

	return
}

//监听任务变化
func (jobMgr *JobMgr) watchJobs() (err error) {

	var (
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		job *common.Job
		watchStartRevision int64
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName string
		jobEvent *common.JobEvent
	)

	//1.get /cron/jobs/目录下所有任务，并获知当前集群Revision
	if getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		goto ERR
	}

	for _, kvPair = range getResp.Kvs {
		if job, err = common.UnpackJob(kvPair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			//fmt.Println(*jobEvent)
			G_scheduler.PushJobEvent(jobEvent)
		}
	}

	//2.从该Revision往后监听变化
	go func() {	//监听协程
		//从GET时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		//监听/cron/jobs/的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())

		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:	//任务保存（新建或修改）
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						continue
					}
					//构造一个更新event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:	//任务删除
					//DELETE /cron/jobs/job1
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					//构造一个删除event事件
					job = &common.Job{Name:jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				}
				G_scheduler.PushJobEvent(jobEvent)
				// fmt.Println(*jobEvent)
			}
		}

	}()

	return

ERR:
	return err
}

//创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	//返回一把锁
	jobLock = initJobLock(jobName, jobMgr.kv, jobMgr.lease)
	return
}

//监听强杀目录
func (jobMgr *JobMgr) watchKiller() {
	var (
		job *common.Job
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobName string
		jobEvent *common.JobEvent
	)

	go func() {	//监听协程
		//从现在开始监听/cron/jobs/的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:	//任务保存（新建或修改）
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					job = &common.Job{Name:jobName}  //杀死任务只需要任务名就可以了
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					G_scheduler.PushJobEvent(jobEvent)

				case mvccpb.DELETE:	//租约过期强杀key自动删除，不需要做什么

				}

			}
		}

	}()

	return
}

