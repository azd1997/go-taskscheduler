package main


import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

func main() {

	var (
		config clientv3.Config
		client *clientv3.Client
		err error
		kv clientv3.KV
		getResp *clientv3.GetResponse
		watchStartRevision int64
		watcher clientv3.Watcher
		watchRespChan <-chan clientv3.WatchResponse
		watchResp clientv3.WatchResponse
		event *clientv3.Event
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


	// 模逆etcd中KV的不停变化
	go func() {
		for {
			kv.Put(context.TODO(), "hello", "eiger")
			kv.Delete(context.TODO(), "hello")
			time.Sleep(1*time.Second)
		}
	}()

	// 任务：监听"hello"
	// 1. 先GET到当前值，再监听后续变化
	if getResp, err = kv.Get(context.TODO(), "hello"); err != nil {
		fmt.Println(err)
		return
	}
	// 现在"hello"是存在的
	if len(getResp.Kvs) != 0 {
		fmt.Println("当前值：", getResp.Kvs)
	}

	// 当前etcd集群事务ID，单调递增。  从这个事务开始，监听后续变化
	watchStartRevision = getResp.Header.Revision + 1

	// 创建一个watcher
	watcher = clientv3.NewWatcher(client)

	// 启动监听
	fmt.Println("从该版本开始监视：", watchStartRevision)

	// 设置主动取消
	ctx, cancelFunc := context.WithCancel(context.TODO())
	// 5秒后cancelFunc.取消掉cancelFunc对应的ctx填入的函数，导致下边的watchRespChan遍历直接退出
	time.AfterFunc(5*time.Second, func() {
		cancelFunc()
	})

	watchRespChan = watcher.Watch(ctx, "hello", clientv3.WithRev(watchStartRevision))

	// 接下来需要不停地取channel，可以遍历channel 或者使用for循环一个一个取
	for watchResp = range watchRespChan {
		for _, event = range watchResp.Events {
			switch event.Type {
			case mvccpb.PUT:
				fmt.Println("修改为：", string(event.Kv.Value), "Revision：", event.Kv.ModRevision)
			case mvccpb.DELETE:
				fmt.Println("删除了：", "Revision：", event.Kv.ModRevision)
			}
		}
	}


}