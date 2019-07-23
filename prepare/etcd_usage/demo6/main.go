package main

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"time"
)

// 学习租约（lease）使用

func main() {

	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		putResp *clientv3.PutResponse
		getResp *clientv3.GetResponse
		lease clientv3.Lease
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		keepResp *clientv3.LeaseKeepAliveResponse
		keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
	)

	// 客户端配置
	config = clientv3.Config{
		Endpoints:[]string{"127.0.0.1:2379"},
		DialTimeout:5*time.Second,
	}

	// 建立连接
	if client, err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	// 1.申请一个lease对象
	lease = clientv3.NewLease(client)

	// 2.申请一个10秒的租约  Grant
	if leaseGrantResp, err = lease.Grant(context.TODO(), 10); err != nil {
		fmt.Println(err)
		return
	}

	// 3. 拿到租约ID
	leaseId = leaseGrantResp.ID

	// (4).自动续租
	// ctx, _ := context.WithTimeout(context.TODO(), 5*time.Second)		// ctx填入下式中可以在五秒后取消该操作（自动续租）
	// 所以总的租约有10（生命期）+5=15s
	if keepRespChan, err = lease.KeepAlive(context.TODO(), leaseId); err != nil {
		fmt.Println(err)
		return
	}

	// 启动一个协程，“消费”KeepRespChan   处理续约应答的协程
	go func() {
		for {
			select {
			case keepResp = <- keepRespChan:
				if keepRespChan == nil {	//自动续约失效有两种可能： 一是网络异常导致长时间未续约而失效；二是自己设定了context去指定失效
					fmt.Println("租约已经失效了")
					goto END
				} else {	//每秒会续租一次，所以会收到一次应答
					fmt.Println("收到自动续租应答：", keepResp.ID)
				}
			}
		}
		END:
	}()


	// 获取KV对象，用于操作键值对
	kv = clientv3.NewKV(client)
	// Put一个KV对，将其与lease关联起来，实现10秒后自动过期（淘汰掉，清除掉）
	if putResp, err = kv.Put(context.TODO(), "/cron/jobs/job3", "hello, worker",clientv3.WithLease(leaseId)); err != nil {
		fmt.Println(err)
	}

	fmt.Println("写入成功：", putResp.Header.Revision)

	// 定时看下key有没有过期
	for {
		if getResp, err = kv.Get(context.TODO(), "/cron/jobs/job3"); err != nil {
			fmt.Println(err)
			return
		}
		if getResp.Count == 0 {
			fmt.Println("kv过期了")
			break
		}
		fmt.Println("还没过期", getResp.Kvs)
		time.Sleep(2*time.Second)
	}

	// 租约过期主要用于程序宕机之后锁自动释放
	// 但实际我们不希望
}
