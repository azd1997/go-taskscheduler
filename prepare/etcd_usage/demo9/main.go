/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/20 20:59
* @Description: 使用Op实现分布式下的乐观锁，也就是分布锁
***********************************************************************/

package main

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"time"
)

func main() {

	// Start your code here

	// op操作
	// txn事务： if else then
	// 组合这两种操作即可实现乐观锁
	// lease实现锁自动过期

	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		lease clientv3.Lease
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		leaseKeepRespChan <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeepResp *clientv3.LeaseKeepAliveResponse
		ctx context.Context
		cancelFunc context.CancelFunc
		kv clientv3.KV
		txn clientv3.Txn
		txnResp *clientv3.TxnResponse
	)

	// 新建配置，建立连接
	config = clientv3.Config{
		Endpoints:[]string{"127.0.0.1:2379"},
		DialTimeout:5*time.Second,
	}
	if client, err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("即将1")

	// 实现乐观锁的步骤
	// 1. 上锁
	// 2. 处理业务
	// 3. 释放锁

	// 1. 上锁（创建租约，自动续租，拿着租约去抢占一个key）
	lease = clientv3.NewLease(client)
	fmt.Println("即将11")
	if leaseGrantResp, err = lease.Grant(context.TODO(), 5); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("即将2")
	leaseId = leaseGrantResp.ID
	ctx, cancelFunc = context.WithCancel(context.TODO())	//准备用于取消自动续租的context
	fmt.Println("即将3")
	defer cancelFunc()	//确保函数退出后自动续租会取消
	defer lease.Revoke(context.TODO(), leaseId)	//废除（释放）lease
	fmt.Println("即将4")
	if leaseKeepRespChan, err = lease.KeepAlive(ctx, leaseId); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("即将进入携程")
	go func() {		//处理自动续租应答
		for {
			select {
			case leaseKeepResp = <-leaseKeepRespChan:
				if leaseKeepRespChan == nil {
					fmt.Println("租约已经失效了")
					goto END
				} else {	// 定时续租，续租就会收到续租回复
					fmt.Println("收到自动续租应答：", leaseKeepResp.ID)
				}
			}
		}
		END:
	}()
	// 租约和自动续约已经做好，现在需要抢key，将key和租约绑定，使之关联，这称为锁
	// 在分布式里边，各个节点的进程里都在抢锁，抢到锁用完之后key被删除，又让其他人去抢锁创建新key
	// if key不存在， then 设置它， else抢锁失败
	fmt.Println("即将创建KV")
	kv = clientv3.NewKV(client)
	// 创建事务
	txn = kv.Txn(context.TODO())
	// 定义事务: 比较key的创建revision是否等于0（这说明key不存在）
	ifCondition := clientv3.Compare(clientv3.CreateRevision("/cron/lock/job9"), "=", 0)		//检查key是否存在
	thenOperation := clientv3.OpPut("/cron/lock/job9", "xxxx", clientv3.WithLease(leaseId))		//创建key，抢到锁
	elseOperation := clientv3.OpGet("/cron/lock/job9")	//否则抢锁失败
	txn.If(ifCondition).Then(thenOperation).Else(elseOperation)
	// 提交事务
	if txnResp, err = txn.Commit(); err != nil {
		fmt.Println(err)
		return	// 没有问题
	}
	// 判断执行的事务是否抢到了锁
	if !txnResp.Succeeded {
		fmt.Println("锁被占用：", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		return
	}				// 抢锁失败的话，就会执行前面的Else，然后在这里获取到别人设置的value值
	// 成功的话就自己设置value为“xxxx”，但是不会打印出来。	事实上，因为是分布式的调度程序，所以抢锁失败打印出的是别人写入的"xxxx"



	// 2. 处理业务
	// 此时已经在锁内，很安全
	fmt.Println("处理任务")
	time.Sleep(5*time.Second)


	// 3. 释放锁（取消自动续租，释放租约）
	// defer cancelFunc()	//确保函数退出后自动续租会取消
	//defer lease.Revoke(context.TODO(), leaseId)	//废除（释放）lease
	// defer把租约释放掉，关联的KV就被删除了

	return
}