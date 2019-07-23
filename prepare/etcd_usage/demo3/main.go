package main

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"time"
)

func main() {

	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		getResp *clientv3.GetResponse
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

	// 获取KV对象，用于操作键值对
	kv = clientv3.NewKV(client)
	// 第四个参数指附加选项，以With开头   // 没有特别需求，参数一填TODO()，这个context主要用于并发编程中协程间通信
	if getResp, err = kv.Get(context.TODO(), "/cron/jobs/job2"); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(getResp.Kvs)
	}
}


