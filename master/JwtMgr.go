package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/dgrijalva/jwt-go"
	"github.com/etcd-io/etcd/clientv3"
	"go-cron/common"
	"strings"
	"time"
)

// 任务管理器
type JwtMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}

var (
	// 单例
	G_jwtMgr *JwtMgr
)

//自定义Claims
type CustomClaims struct {
	Username   string
	jwt.StandardClaims
}

// 初始化
func InitJwtMgr() (err error) {
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
	G_jwtMgr = &JwtMgr{
		client: client,
		kv: kv,
		lease: lease,
	}
	return
}

// 生成token
func GenToken()(token string, err error){
	var (
		claims = &CustomClaims{
			Username: G_config.Username,
			StandardClaims: jwt.StandardClaims{
				// 设置过期时间
				ExpiresAt: time.Now().Add(time.Duration(G_config.TokenExpiration)*time.Second).Unix(),
			},
		}
	)
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(G_config.JWTSecretKey))
	return
}

// 解析token
func ParseToken(tokenStr string)(claims *CustomClaims, err error){
	var (
		ok bool
		token *jwt.Token
	)
	tokenStr = strings.TrimSpace(strings.Replace(tokenStr, "Bearer", "", -1))
	// 忽略token验证错误,只解析token
	token, _ = jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok = token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(G_config.JWTSecretKey), nil
	})
	if claims, ok = token.Claims.(*CustomClaims); !ok {
		err = common.ERR_ASSERT_CLAIMS_FAIL
	}
	return
}

// 查询token
func (jwtMgr *JwtMgr) GetToken(tokenKey string) (token string, err error) {
	var (
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
	)
	if getResp, err = jwtMgr.kv.Get(context.TODO(),common.TOKEN_CACHE_DIR + tokenKey);err != nil {
		return
	}
	for _, kvPair = range getResp.Kvs {
		token = string(kvPair.Value)
		break
	}
	return
}

// 保存token
func (jwtMgr *JwtMgr) SaveToken(tokenKey, tokenVal string)(oldToken string, err error){
	// 把任务保存到 /cron/jobs/任务名 -> json
	var putResp *clientv3.PutResponse
	if tokenVal == "" {
		tokenVal = tokenKey
	}
	// 当前只需要一个管理员,只允许一处登录,因此删除所有的token
	if _, err = jwtMgr.kv.Delete(context.TODO(), common.TOKEN_CACHE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	// 保存到etcd
	if putResp, err = jwtMgr.kv.Put(context.TODO(), common.TOKEN_CACHE_DIR + tokenKey, tokenVal, clientv3.WithPrevKV()); err != nil{
		return
	}
	// 如果是更新, 那么返回旧值
	if putResp.PrevKv != nil {
		// 对旧值做一个反序列化
		oldToken = putResp.PrevKv.String()
	}
	return
}

// 删除token
func (jwtMgr *JwtMgr) DeleteToken(tokenKey string) (oldToken string, err error) {
	var (
		delResp *clientv3.DeleteResponse
	)
	// 从etcd中删除
	if delResp, err = jwtMgr.kv.Delete(context.TODO(), common.TOKEN_CACHE_DIR + tokenKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	// 返回被删除的token信息
	if len(delResp.PrevKvs) != 0 {
		// 解析一下旧值,返回它
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldToken);err != nil {
			err = nil
			return
		}
	}
	return
}
