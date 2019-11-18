/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/7/2 22:28
* @Description: 分布式锁
***********************************************************************/

package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"golang-taskschedule/crontab/common"
)


//分布式锁（TXN事务 ）
type JobLock struct {
	//etcd客户端的kv和lease
	kv clientv3.KV
	lease clientv3.Lease
	jobName string  //任务名
	cancelFunc context.CancelFunc  //用于终止自动续租
	leaseId clientv3.LeaseID  //有了leaseId和cancelFunc就可以实现释放锁
	isLocked bool //上锁成功与否
}

//初始化一把锁，锁对象
func initJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease ) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:kv,
		lease:lease,
		jobName:jobName,
	}
	return
}

//尝试上锁，抢得到就可以执行任务
func (jobLock *JobLock) TryLock() (err error) {

	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx context.Context
		cancelFunc context.CancelFunc
		leaseId clientv3.LeaseID
		keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
		txn clientv3.Txn
		lockKey string
		txnResp *clientv3.TxnResponse
	)

	//创建租约（5秒）
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	//创建用于取消的自动续租的context
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	//租约ID
	leaseId = leaseGrantResp.ID

	//自动续租
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FAIL //自动续约失败，相当于上锁失败，就直接把租约释放掉，没必要浪费资源存储
	}

	//处理续租应答的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <- keepRespChan: //自动续租应答
				if keepResp == nil { //说明自动续租被取消了
					goto END
				}
			}
		}
		END:  //退出协程
	}()

	//创建事务txn
	txn = jobLock.kv.Txn(context.TODO())
	//锁路径
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName

	//事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).  //说明lockkey不存在
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).  //存入lockkey，也就是抢到锁
		Else(clientv3.OpGet(lockKey))   //锁已经被别人抢了，取下信息（没什么实际意义）

	//提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL  //提交失败有两种情况：没抢到锁，或者抢到锁提交应答时网络出问题。但你没法检查具体，所以依然取消续约
	}

	//成功返回，失败则释放租约
	if !txnResp.Succeeded {
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}
	//上锁成功，则把leaseId返回一下，用以后边UNlOCK()使用
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return


FAIL:  //FAIL确保在任何异常下都可以回滚
	cancelFunc()  //取消自动续租
	jobLock.lease.Revoke(context.TODO(), leaseId) //释放租约

	return


}

//释放锁
func (jobLock *JobLock) UnLock() {
	if jobLock.isLocked {
		jobLock.cancelFunc()  //取消程序自动续租的协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId)
	}
}