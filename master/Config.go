package master

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	ApiPort int `json:"apiPort"`
	ApiReadTimeout int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
	EtcdEndpoints []string `json:"etcdEndpoints"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	WebRoot string `json:"webRoot"`
	ElasticHost string `json:"elasticHost"`
	ElasticPort string `json:"elasticPort"`
	ElasticUsername string `json:"elasticUsername"`
	ElasticPassword string `json:"elasticPassword"`
	IndexName string `json:"indexName"`
	ProjectNames []string `json:"projectNames"`
	JWTSecretKey  string `json:"jwtSecretKey"`
	TokenExpiration int `json:"expiration"`
	Username string `json:"username"`
	Password string `json:"password"`
	MongodbUri string `json:"mongodbUri"`
	MongodbConnectTimeout int `json:"mongodbConnectTimeout"`
}

var (
	// 单例
	G_config *Config
)

func InitConfig(filename string) (err error){
	var (
		content []byte
		conf Config
	)
	// 1.把配置文件读进来
	if content ,err = ioutil.ReadFile(filename);err != nil {
		return
	}

	// 2. JSON反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3. 赋值单例
	G_config = &conf
	//fmt.Println(fmt.Sprintf("配置项: %v",conf))
	return
}


