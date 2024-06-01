package worker

import (
	"encoding/json"
	"fmt"
	"go-cron/common"
	"io/ioutil"
)

// 程序配置
type Config struct {
	EtcdEndpoints []string `json:"etcdEndpoints"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	ElasticHost string `json:"elasticHost"`
	ElasticPort string `json:"elasticPort"`
	ElasticUsername string `json:"elasticUsername"`
	ElasticPassword string `json:"elasticPassword"`
	IndexName string `json:"indexName"`
	MongodbUri string `json:"mongodbUri"`
	MongodbConnectTimeout int `json:"mongodbConnectTimeout"`
	JobLogBatchSize int `json:"jobLogBatchSize"`
	JobLogCommitTimeout int `json:"jobLogCommitTimeout"`
	LocalIP string `json:"ip_addr"`
}

var (
	// 单例
	G_config *Config
)

func InitConfig(filename string) (err error){
	var (
		content []byte
		conf Config
		localIP string
	)
	// 1.把配置文件读进来
	if content ,err = ioutil.ReadFile(filename);err != nil {
		return
	}

	// 2. JSON反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3. 获取本机ip
	if localIP, err = common.GetLocalIP(); err != nil {
		return
	}
	fmt.Println(fmt.Sprintf("local ip: %s", localIP))
	conf.LocalIP = localIP
	// 4. 赋值单例
	G_config = &conf
	//fmt.Println(fmt.Sprintf("配置项: %v",conf))
	return
}


