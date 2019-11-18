/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/23 22:09
* @Description: The file is for
***********************************************************************/

package master

import (
	"encoding/json"
	"golang-taskschedule/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	// 单例对象，提供外部调用，这里InitApiServer需要在main/main.go去调用
	G_apiServer *ApiServer	// 初始为空
)

// 初始化服务
func InitApiServer() (err error) {

	var (
		mux *http.ServeMux
		listener net.Listener
		httpServer *http.Server
		staticDir http.Dir
		staticHandler http.Handler
	)
	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/hello", handleHello)
	mux.HandleFunc("/test/postform", handlePostForm)
	//handlefunc是用来处理动态接口的

	// 静态文件目录
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	staticHandler = http.StripPrefix("/", staticHandler)	//去掉url /index.html 的前缀"/"
	mux.Handle("/", staticHandler)	//分析url，看哪个最匹配（重合的长度最长）  / + index.html

	// 启动TCP监听
	if listener, err = net.Listen("tcp", "127.0.0.1:" + strconv.Itoa(G_config.ApiServerPort)); err != nil {
		return
	}

	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:time.Duration(G_config.ApiServerReadTimeout) * time.Millisecond,
		WriteTimeout:time.Duration(G_config.ApiServerWriteTimeout) * time.Millisecond,
		Handler:mux,
	}

	// 赋值单例   创建单例对象，以便外部调用
	G_apiServer = &ApiServer{
		httpServer:httpServer,
	}

	// 启动服务端，在协程中监听
	go httpServer.Serve(listener)

	return
}

// 保存任务接口
// 网页客户端 POST job={"name"="job1", "command"="echo hello", "cronExpr"="* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	// 将任务保存到ETCD中。 这里需要先定义“任务”和“ETCD”，因为任务是通过网页传入的，而Worker也需要
	// 所以将这部分写在common包里；而ETCD是由Master来增删改查，所以写在Master包里 Master/JobMgr.go

	var (
		err error
		postJob string
		job common.Job
		oldJob *common.Job
		respBytes []byte
	)

	// 1.解析POST表单
		//golang的http服务默认不会解析，因为会耗费CPU资源，所以需要自己去调用方法
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 2.取表单的job字段
	postJob = req.PostForm.Get("job")

	// 3.对job作json反序列化
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	// 4.将job对象保存起来。 （传给JobMgr，再由JobMgr保存到ETCD）
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	// 5.向网页客户端返回正常应答 {"error":0, "msg":"", "data":{...}} 为避免麻烦，组装一个应答结构体 common/Protocal.go/Response
	if respBytes, err = common.BuildResponse(0, "success！", oldJob); err == nil {
		//goto ERR	//todo:如果在ERR中定义返回ERR应答，那么这里可能会无限ERR应答
		resp.Write(respBytes)
	}
	return	//注意这里要return,不然会继续执行一次ERR:

ERR:
	// 返回异常应答
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respBytes)
	}
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello, eiger"))
}

func handlePostForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {

	}

	// 2.取表单的job字段
	postJob := r.PostForm.Get("job")

	// 3.对job作json反序列化
	var job common.Job
	if err := json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	// 5.向网页客户端返回正常应答 {"error":0, "msg":"", "data":{...}} 为避免麻烦，组装一个应答结构体 common/Protocal.go/Response
	if respBytes, err := common.BuildResponse(0, "success！", job); err == nil {
		w.Write(respBytes)
	}

ERR:
	// 返回异常应答
	if respBytes, err := common.BuildResponse(-1, "Error!", nil); err == nil {
		w.Write(respBytes)
	}
}

// 删除任务
// 网页客户端 POST /job/delete name=job1
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		toBeDeletedJobname string
		deletedJob *common.Job
		respBytes []byte
	)

	// 1.解析POST表单
	//golang的http服务默认不会解析，因为会耗费CPU资源，所以需要自己去调用方法
	// POST form形如 : a=1&b=2&c=3
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	// 2.取表单的job字段
	toBeDeletedJobname = r.PostForm.Get("name")

	// 3.删除job。 （传给JobMgr，再由JobMgr去ETCD删除）
	if deletedJob, err = G_jobMgr.DeleteJob(toBeDeletedJobname); err != nil {
		goto ERR
	}

	// 4.向网页客户端返回正常应答 {"error":0, "msg":"", "data":{...}} 为避免麻烦，组装一个应答结构体 common/Protocal.go/Response
	if respBytes, err = common.BuildResponse(0, "success！", deletedJob); err == nil {
		w.Write(respBytes)
	}
	return	//注意这里要return,不然会继续执行一次ERR:

ERR:
	// 返回异常应答
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(respBytes)
	}
}

//列举所有crontab任务
func handleJobList(w http.ResponseWriter, r *http.Request) {

	var (
		allJobs []*common.Job
		err error
		respBytes []byte
	)

	if allJobs, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}

	if respBytes, err = common.BuildResponse(0, "success!", allJobs); err == nil {
		w.Write(respBytes)
		//data, _ := json.Marshal(allJobs)
		//w.Write(data)	//调试
	}

	return

ERR:
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(respBytes)
	}
}

//强杀任务
// POST /job/kill  name=job1
func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		toBeKilledJobName string
		respBytes []byte
	)

	//解析表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	//获取强杀任务的任务名
	toBeKilledJobName = r.PostForm.Get("name")

	if err = G_jobMgr.KillJob(toBeKilledJobName); err != nil {
		goto ERR
	}

	if respBytes, err = common.BuildResponse(0, "success!", toBeKilledJobName); err == nil {
			w.Write(respBytes)
	}

	return

ERR:
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(respBytes)
	}

}

//列举所有crontab任务
func handleJobList(w http.ResponseWriter, r *http.Request) {

	var (
		err error
		jobName string  //任务名
		respBytes []byte
	)

	if allJobs, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}

	if respBytes, err = common.BuildResponse(0, "success!", allJobs); err == nil {
		w.Write(respBytes)
		//data, _ := json.Marshal(allJobs)
		//w.Write(data)	//调试
	}

	return

ERR:
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(respBytes)
	}
}