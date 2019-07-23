/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/24 13:53
* @Description: The file is for
***********************************************************************/

package master

import (
	"context"
	"encoding/json"
	"go.etcd.io/etcd/clientv3"
	"golang-taskschedule/crontab/common"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
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

	//赋值单例
	G_jobMgr = &JobMgr{
		client:client,
		kv:kv,
		lease:lease,
	}

	return
}

// 保存任务，返回旧任务
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	//把任务保存到cron/jobs/任务名->json

	var (
		jobKey string
		jobValue []byte
		putResp *clientv3.PutResponse
		////这里需要设置oldJobObj作为临时变量
		//oldJobObj common.Job
	)

	//etcd的保存Key
	jobKey = common.JOB_SAVE_DIR + job.Name

	//将job序列化成json，作为value
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	//将任务的key-value保存到ETCD
	if putResp, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV());err != nil {
		return
	}

	//如果PUT是更新，那么返回旧值
	if putResp.PrevKv != nil {
		//对旧值反序列化并返回给网页客户端
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJob); err != nil {
			err = nil 	//因为返回oldJob出错不影响job保存（此时新job已经存好了）
			return
		}
	}

	return
}

func (jobMgr *JobMgr) DeleteJob(jobName string) (deletedJob *common.Job, err error) {

	var (
		jobKey string
		delResp *clientv3.DeleteResponse
	)

	jobKey = common.JOB_SAVE_DIR + jobName
	if delResp, err = jobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	//返回被删除任务的信息 被删除的job若原先存在，在返回的PrevKVs长度一定为1，否则为0
	if len(delResp.PrevKvs) != 0 {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &deletedJob); err != nil {
			err = nil
			return
		}
	}

	return
}

//列举所有crontab任务  无翻页
func (jobMgr *JobMgr) ListJobs() (allJobs []*common.Job, err error) {
	var (
		dirKey string
		getResp *clientv3.GetResponse
		/*kvPair *mvccpb.KeyValue
		job *common.Job*/
		lenKvs int
	)
	//任务保存的目录
	dirKey = common.JOB_SAVE_DIR
	//获取目录下所有任务信息
	if getResp, err = jobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		return
	}

/*	注释掉的这种写法，好处是直接根据len(allJobs)判断成功或失败。
	坏处是allJobs需要不断重新分配更大的空间，如果任务很多，那么这样做效率很低
	//初始化allJobs数组空间，使得其不为空
	allJobs = make([]*common.Job, 0)
	//len(allJobs)==0

	//遍历所有任务，进行反序列化
	for _, kvPair = range getResp.Kvs {
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			err = nil //允许个别kvPair反序列化失败
			continue
		}
		allJobs = append(allJobs, job)
	}*/

	/*第二种做法，预先分配数组长度和容量*/
	lenKvs = len(getResp.Kvs)
	allJobs = make([]*common.Job, lenKvs, lenKvs)
	for i:=0;i<lenKvs;i++ {
		//注意传入的是&allJobs[i]而不是allJobs[i]
		if err = json.Unmarshal(getResp.Kvs[i].Value, &allJobs[i]); err != nil {
			err = nil //允许个别kvPair反序列化失败
			continue
		}
	}

	return
}

//杀死任务
// 更新key=/cron/killer/任务名, worker节点监听其变化，来杀死任务
// 当worker节点监听到killer变化时，看下自己有没有执行该任务，有就杀死
func (jobMgr *JobMgr) KillJob(toBeKilledJobName string) (err error) {

	var (
		killerKey string
		leaseGrantResponse *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
	)

	killerKey = common.JOB_KILLER_DIR + toBeKilledJobName

	//因为只是拿kv变化作为通知，所以不需要让etcd一直保存kv，过一段时间自动过期就好。这里设置1秒过期

	if leaseGrantResponse, err = jobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	leaseId = leaseGrantResponse.ID

	//保存key,value不关心，设空值
	if _, err = jobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return
}
