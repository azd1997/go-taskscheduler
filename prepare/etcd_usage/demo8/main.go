package main

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"time"
)

// OP 新的PUTGET操作
// 这是实现分布式锁必须用到的知识点


func main() {
	var(
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		putOp clientv3.Op
		opResp clientv3.OpResponse
		getOp clientv3.Op
	)

	// 客户端配置
	config = clientv3.Config{
		Endpoints:[]string{"127.0.0.1:2379"},
		DialTimeout:5*time.Second,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	// 获取KV对象
	kv = clientv3.NewKV(client)

	// Op : operation 创建Op对象
	putOp = clientv3.OpPut("a", "第一名")

	// 执行Op
	if opResp, err = kv.Do(context.TODO(), putOp); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("写入Revision：", opResp.Put().Header.Revision)

	// 同样地，来一遍GetOP
	getOp = clientv3.OpGet("a")
	if opResp, err = kv.Do(context.TODO(), getOp); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("数据Revision:", opResp.Get().Header.Revision)
	fmt.Println("数据value:", string(opResp.Get().Kvs[0].Value))



}