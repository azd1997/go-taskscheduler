/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/24 22:56
* @Description: The file is for
***********************************************************************/

package common

const (
	JOB_SAVE_DIR   = "/cron/jobs/"   //任务保存目录
	JOB_KILLER_DIR = "/cron/killer/" //任务强杀目录
	JOB_LOCK_DIR   = "/cron/lock/"   //任务锁目录

	MASTER_CONFIG_DEFAULT_DIR = "./master.json"
	WORKER_CONFIG_DEFAULT_DIR = "./worker.json"

	JOB_EVENT_SAVE   = 1 //保存任务事件
	JOB_EVENT_DELETE = 2 //删除任务事件
	JOB_EVENT_KILL   = 3 //强杀任务事件
)