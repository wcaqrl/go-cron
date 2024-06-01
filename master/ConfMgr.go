package master

import (
	"context"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/etcd-io/etcd/clientv3"
	"go-cron/common"
	"strings"
	"time"
)

// 配置管理器
type ConfMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}

var (
	// 单例
	G_confMgr *ConfMgr
)

// 初始化
func InitConfMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
	)

	// 初始化配置
	config = clientv3.Config{
		Endpoints: G_config.EtcdEndpoints, // 集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	// 建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 得到KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	// 赋值单例
	G_confMgr = &ConfMgr{
		client: client,
		kv: kv,
		lease: lease,
	}

	return
}

// 保存任务
func (confMgr *ConfMgr) SaveConfig(confKey, confStr string)(oldConf string, err error){
	// 把任务保存到 /cron/conf/配置名 -> text
	var putResp *clientv3.PutResponse
	confKey = strings.TrimSpace(confKey)
	confStr = strings.TrimSpace(confStr)
	if confKey == "" || confStr == "" {
		err = common.ERR_EMPTY_PARAMS
		return
	}
	// etcd的保存key
	confKey = common.API_CONF_DIR + confKey
	// 保存到etcd
	if putResp, err = confMgr.kv.Put(context.TODO(), confKey, confStr, clientv3.WithPrevKV()); err != nil{
		return
	}

	// 如果是更新, 那么返回旧值
	if putResp.PrevKv != nil {
		oldConf = string(putResp.PrevKv.Value)
	}
	return
}

// 删除任务
func (confMgr *ConfMgr) DeleteConfig(confKey string) (oldConf string, err error) {
	var delResp *clientv3.DeleteResponse
	confKey = strings.TrimSpace(confKey)
	if confKey == "" {
		err = common.ERR_EMPTY_PARAMS
		return
	}
	// etcd中保存任务的key
	confKey = common.API_CONF_DIR + confKey
	// 从etcd中删除
	if delResp, err = confMgr.kv.Delete(context.TODO(), confKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	// 返回被删除的任务信息
	if len(delResp.PrevKvs) != 0 {
		oldConf = string(delResp.PrevKvs[0].Value)
	}
	return
}

// 获取单个配置详情
func (confMgr *ConfMgr) DetailConfig(confKey string) (confMap map[string]string, err error) {
	var (
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
	)
	confMap = make(map[string]string)
	confKey = strings.TrimSpace(confKey)
	if confKey == "" {
		err = common.ERR_EMPTY_PARAMS
		return
	}
	if getResp, err = confMgr.kv.Get(context.TODO(),common.API_CONF_DIR + confKey);err != nil {
		return
	}
	// 遍历所有任务, 进行反序列化
	for _, kvPair = range getResp.Kvs {
		if strings.HasPrefix(string(kvPair.Key), common.API_CONF_DIR) {
			confMap["conf_key"] = common.API_CONF_DIR + confKey
			confMap["conf_val"] = string(kvPair.Value)
			break
		}
	}
	return
}

// 获取配置列表
func (confMgr *ConfMgr) ListConfig(confKey string, page, perpage int64)(confArr []map[string]string, err error) {
	var (
		i int64
		dirKey, lastKey  string
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		firstOptions []clientv3.OpOption
		otherOptions []clientv3.OpOption
		tmpMap map[string]string
	)
	confKey = strings.TrimSpace(confKey)
	// 初始化数组空间
	confArr = make([]map[string]string, 0)
	// 配置保存的目录
	dirKey = common.API_CONF_DIR
	lastKey = dirKey + confKey
	firstOptions = []clientv3.OpOption {
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(perpage),
	}
	otherOptions = []clientv3.OpOption {
		clientv3.WithFromKey(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(perpage),
	}

	if page == 1 {
		if getResp, err = confMgr.kv.Get(context.TODO(),lastKey, firstOptions...);err != nil {
			return
		}
	} else {
		// for 循环页码
		for i = 1 ; i <= page; i++ {
			// 获取目录下的任务列表
			if getResp, err = confMgr.kv.Get(context.TODO(),lastKey, otherOptions...);err != nil {
				return
			}
			// 如果查不到值直接返回
			if len(getResp.Kvs) <= 1 {
				return
			}

			if int64(len(getResp.Kvs)) < perpage {
				if i < page {
					return
				} else {
					break
				}
			}

			// 遍历所有配置
			lastKey = ""
			for _, kvPair = range getResp.Kvs {
				if strings.HasPrefix(string(kvPair.Key), dirKey) {
					lastKey = dirKey + string(kvPair.Key)
				}
			}
			if lastKey == "" {
				return
			}
		}
	}

	// 遍历所有任务, 进行反序列化
	for _, kvPair = range getResp.Kvs {
		if strings.HasPrefix(string(kvPair.Key), dirKey) {
			tmpMap  = make(map[string]string)
			tmpMap["conf_key"] = string(kvPair.Key)
			tmpMap["conf_val"] = string(kvPair.Value)
			confArr = append(confArr, tmpMap)
		}
	}
	return
}

// 获取配置总数
func (confMgr *ConfMgr) CountConfig(confKey string) (countMap map[string]int64, err error) {
	var (
		getResp *clientv3.GetResponse
		opOptions []clientv3.OpOption
	)
	countMap = map[string]int64{"count": 0}
	// api配置保存的目录
	opOptions = []clientv3.OpOption {
		clientv3.WithPrefix(),
	}
	if getResp, err = confMgr.kv.Get(context.TODO(),common.API_CONF_DIR + confKey, opOptions...);err != nil {
		return
	}
	countMap["count"] = getResp.Count
	return
}