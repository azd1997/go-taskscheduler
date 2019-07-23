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
		putResp *clientv3.PutResponse
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
	// 第四个参数指附加选项，以With开头
	if putResp, err = kv.Put(context.TODO(), "/cron/jobs/job2", "hello,newjob", clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("PutResponse: Header: Revision : ", putResp.Header.Revision)
		if putResp.PrevKv != nil {
			fmt.Println("PrevValue: ", string(putResp.PrevKv.Value))	//打印 hello,job2
		}
	}
}


