/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2019/6/24 12:32
* @Description: 读取配置文件
***********************************************************************/

package master

import (
	"encoding/json"
	"io/ioutil"
)

var (
	// 单例
	G_config *Config
)

type Config struct {
	ApiServerPort         int      `json:"apiServerPort"`
	ApiServerReadTimeout  int      `json:"apiServerReadTimeout"`
	ApiServerWriteTimeout int      `json:"apiServerWriteTimeout"`
	EtcdEndpoints         []string `json:"etcdEndpoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	WebRoot string `json:"webroot"`
}

// 读取配置
func InitConfig(configFile string) (err error) {

	var (
		content []byte
		config Config
	)

	// 读取配置文件，得到[]byte内容
	if content, err = ioutil.ReadFile(configFile); err != nil {
		return
	}

	// 反序列化
	if err = json.Unmarshal(content, &config); err != nil {
		return
	}

	// 赋值单例
	G_config = &config

	//log.Print(G_config)

	return

}