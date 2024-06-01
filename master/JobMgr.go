package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/etcd-io/etcd/clientv3"
	"go-cron/common"
	"strings"
	"time"
)

// 任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}

var (
	// 单例
	G_jobMgr *JobMgr
)

// 初始化
func InitJobMgr() (err error) {
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
	G_jobMgr = &JobMgr{
		client: client,
		kv: kv,
		lease: lease,
	}

	return
}

// 保存任务
func (jobMgr *JobMgr) SaveJob(job *common.Job)(oldJob *common.Job, err error){
	// 把任务保存到 /cron/jobs/任务名 -> json
	var (
		jobKey string
		jobValue []byte
		jobLock *common.JobLock
		putResp *clientv3.PutResponse
		oldJobObj common.Job
	)
	// etcd的保存key
	jobKey = common.JOB_SAVE_DIR + job.Name
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	// 初始化分布式锁
	jobLock = G_jobMgr.CreateJobLock(job.Name)
	if err = jobLock.TryLock(); err != nil {
		return
	}
	defer jobLock.Unlock()

	// 保存到etcd
	if putResp, err = jobMgr.kv.Put(context.TODO(), jobKey,string(jobValue), clientv3.WithPrevKV()); err != nil{
		return
	}

	// 如果是更新, 那么返回旧值
	if putResp.PrevKv != nil {
		// 对旧值做一个反序列化
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

// 删除任务
func (jobMgr *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey string
		jobLock *common.JobLock
		delResp *clientv3.DeleteResponse
		oldJobObj common.Job
	)

	// etcd中保存任务的key
	jobKey = common.JOB_SAVE_DIR + name

	// 初始化分布式锁
	jobLock = G_jobMgr.CreateJobLock(name)
	if err = jobLock.TryLock(); err != nil {
		return
	}
	defer jobLock.Unlock()

	// 从etcd中删除
	if delResp, err = jobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	// 返回被删除的任务信息
	if len(delResp.PrevKvs) != 0 {
		// 解析一下旧值,返回它
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj);err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

// 获取单个任务详情
func (jobMgr *JobMgr) DetailJob(jobName string) (job *common.Job, err error) {
	var (
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
	)
	if getResp, err = jobMgr.kv.Get(context.TODO(),common.JOB_SAVE_DIR + jobName); err != nil {
		return
	}
	// 遍历所有任务, 进行反序列化
	for _, kvPair = range getResp.Kvs {
		if strings.HasPrefix(string(kvPair.Key), common.JOB_SAVE_DIR) {
			job = &common.Job{}
			err = json.Unmarshal(kvPair.Value, job)
			break
		}
	}
	return
}

// 获取任务列表
func (jobMgr *JobMgr) ListJob(jobName string, page, perpage int64)(jobList []*common.Job, err error) {
	var (
		i int64
		dirKey, lastKey  string
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		job *common.Job
		firstOptions []clientv3.OpOption
		otherOptions []clientv3.OpOption
		//resKvs []*mvccpb.KeyValue
	)
	// 初始化数组空间
	jobList = make([]*common.Job, 0)
	// 任务保存的目录
	dirKey = common.JOB_SAVE_DIR
	lastKey = dirKey + jobName
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
		if getResp, err = jobMgr.kv.Get(context.TODO(),lastKey, firstOptions...);err != nil {
			return
		}
	} else {
		// for 循环页码
		for i = 1 ; i <= page; i++ {
			// 获取目录下的任务列表
			if getResp, err = jobMgr.kv.Get(context.TODO(),lastKey, otherOptions...);err != nil {
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

			// 遍历所有任务, 进行反序列化
			lastKey = ""
			for _, kvPair = range getResp.Kvs {
				job = &common.Job{}
				if strings.HasPrefix(string(kvPair.Key), dirKey) {
					if err = json.Unmarshal(kvPair.Value, job); err != nil {
						err = nil
						continue
					}
					lastKey = dirKey + job.Name
				}
			}
			if lastKey == "" {
				return
			}
		}
	}

	// 遍历所有任务, 进行反序列化
	for _, kvPair = range getResp.Kvs {
		job = &common.Job{}
		if strings.HasPrefix(string(kvPair.Key), dirKey) {
			if err = json.Unmarshal(kvPair.Value, job); err != nil {
				fmt.Println(fmt.Sprintf("unmarshal job error: %s", err.Error()))
				err = nil
				continue
			}
			jobList = append(jobList, job)
		}
	}
	return
}

// 模糊查询任务列表
func (jobMgr *JobMgr) ListFuzzyJob(jobName string, page, perpage int64) (jobList []*common.Job, err error) {
	var (
		start      int // 当前页起始位置
		end        int // 当前页结束位置
		total      int // 模糊匹配到的任务总数
		tmpJobList []*common.Job
	)
	// 初始化数组空间
	jobList = make([]*common.Job, 0)

	if tmpJobList, err = jobMgr.getFuzzyJob(jobName); err != nil {
		return
	}
	total = len(tmpJobList)
	// 页码小于1或总数为0,返回空结果
	if page < 1 || total == 0 {
		return
	}

	start = int((page - 1) * perpage)
	end = int(page * perpage)

	// 总数小于分页起始值,返回空结果
	if total < start {
		return
	} else {
		if total > end {
			jobList = tmpJobList[start:end]
		} else {
			jobList = tmpJobList[start:]
		}
	}

	return
}


// 获取任务总数
func (jobMgr *JobMgr) CountJob(jobName string) (countMap map[string]int64, err error) {
	var (
		getResp *clientv3.GetResponse
		opOptions []clientv3.OpOption
	)
	countMap = map[string]int64{"count": 0}
	// 任务保存的目录
	opOptions = []clientv3.OpOption {
		clientv3.WithPrefix(),
	}
	if getResp, err = jobMgr.kv.Get(context.TODO(),common.JOB_SAVE_DIR + jobName, opOptions...);err != nil {
		return
	}
	countMap["count"] = getResp.Count
	return
}

// 获取模糊查询任务总数
func (jobMgr *JobMgr) CountFuzzyJob(jobName string) (countMap map[string]int64, err error) {
	var tmpJobList []*common.Job
	if tmpJobList, err = jobMgr.getFuzzyJob(jobName); err != nil {
		return
	}
	countMap = map[string]int64{"count": int64(len(tmpJobList))}
	return
}

// 杀死任务
func (jobMgr *JobMgr) KillJob(name string)(err error) {
	// 更新一下 key=/cron/killer/任务名
	var (
		killerKey string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
	)

	// 通知worker杀死对应任务
	killerKey = common.JOB_KILLER_DIR + name

	// 让worker监听到一次put操作,创建一个租约让其稍后自动过期即可
	if leaseGrantResp, err = jobMgr.lease.Grant(context.TODO(),1); err != nil {
		return
	}

	// 租约ID
	leaseId = leaseGrantResp.ID

	// 设置killer标记
	if _, err = jobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}
	return
}

// 执行任务
func (jobMgr *JobMgr) ExecJob(name string)(err error) {
	var (
		jobKey, cronExpr, execCronExpr string
		status int
		job *common.Job
		jobLock *common.JobLock
		jobValue []byte
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		t time.Time
		delay = 2
	)
	// 获取 key=/cron/jobs/任务名
	jobKey = common.JOB_SAVE_DIR + name
	if getResp, err = jobMgr.kv.Get(context.TODO(), jobKey); err != nil {
		return
	}
	// 遍历所有任务, 进行反序列化
	for _, kvPair = range getResp.Kvs {
		if strings.HasPrefix(string(kvPair.Key), common.JOB_SAVE_DIR) {
			job = &common.Job{}
			err = json.Unmarshal(kvPair.Value, job)
			break
		}
	}
	if job == nil {
		err = common.ERR_NOT_FOUND
		return
	}
	cronExpr = job.CronExpr
	status = job.Status
	t = time.Now().Add(time.Second * time.Duration(delay))
	execCronExpr = fmt.Sprintf("%d %d %d * * * *", t.Second(), t.Minute(), t.Hour())
	job.CronExpr = execCronExpr
	job.Status   = common.JOB_STATUS_AVAILABLE
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	// 初始化分布式锁
	jobLock = G_jobMgr.CreateJobLock(name)
	if err = jobLock.TryLock(); err != nil {
		return
	}

	go func() {
		defer jobLock.Unlock()
		// 保存到etcd
		if _, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
			return
		}
		job.CronExpr = cronExpr
		job.Status   = status
		if jobValue, err = json.Marshal(job); err != nil {
			return
		}
		time.Sleep(time.Second * time.Duration(delay + 1))
		// 保存到etcd
		if _, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
			return
		}
	}()

	return
}

// 创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string)(jobLock *common.JobLock){
	// 返回一把锁
	jobLock = common.InitJobLock(jobName, common.JOB_ACTION_DIR, jobMgr.kv, jobMgr.lease)
	return
}

// 获取所有任务
func (jobMgr *JobMgr) getAllJob() (jobList []*common.Job, err error) {
	var (
		ok				bool
		i               int64
		perpage         int64
		dirKey, lastKey string
		getResp         *clientv3.GetResponse
		kvPair          *mvccpb.KeyValue
		job             *common.Job
		firstOptions    []clientv3.OpOption
		otherOptions    []clientv3.OpOption
		jobMap          map[string]*common.Job
		name            string
		nameList        []string
	)
	// 初始化数组空间
	jobList = make([]*common.Job, 0)
	jobMap = make(map[string]*common.Job)
	nameList = make([]string, 0)
	// 任务保存的目录
	dirKey = common.JOB_SAVE_DIR
	lastKey = dirKey
	perpage = 100
	firstOptions = []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(perpage),
	}
	otherOptions = []clientv3.OpOption{
		clientv3.WithFromKey(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(perpage),
	}

	// for 循环页码
	for i = 1; ; i++ {
		// 获取目录下的任务列表
		if i == 1 {
			if getResp, err = jobMgr.kv.Get(context.TODO(), lastKey, firstOptions...); err != nil {
				return
			}
		} else {
			if getResp, err = jobMgr.kv.Get(context.TODO(), lastKey, otherOptions...); err != nil {
				return
			}
		}

		// 遍历所有任务, 进行反序列化
		lastKey = ""
		for _, kvPair = range getResp.Kvs {
			job = &common.Job{}
			if strings.HasPrefix(string(kvPair.Key), dirKey) {
				if err = json.Unmarshal(kvPair.Value, job); err != nil {
					err = nil
					continue
				}
				lastKey = dirKey + job.Name
				if _, ok = jobMap[job.Name]; !ok {
					jobMap[job.Name] = job
					nameList = append(nameList, job.Name)
				}
			}
		}
		if lastKey == "" {
			break
		}

		// 如果查不到值直接返回
		if (len(getResp.Kvs) <= 1) || (int64(len(getResp.Kvs)) < perpage) {
			break
		}

	}

	for _, name = range nameList {
		jobList = append(jobList, jobMap[name])
	}

	return
}

// 模糊查询任务列表
func (jobMgr *JobMgr) getFuzzyJob(jobName string) (jobList []*common.Job, err error) {
	var (
		job        *common.Job
		tmpJobList []*common.Job
		matchJob   []*common.Job
		prefixJob  []*common.Job
		fuzzyJob   []*common.Job
	)
	// 初始化数组
	jobList = make([]*common.Job, 0)
	// 名称完全匹配
	matchJob = make([]*common.Job, 0)
	// 名称前缀匹配
	prefixJob = make([]*common.Job, 0)
	// 名称模糊匹配
	fuzzyJob = make([]*common.Job, 0)

	if tmpJobList, err = jobMgr.getAllJob(); err != nil {
		return
	}

	if jobName == "" {
		jobList = tmpJobList
		return
	}

	for _, job = range tmpJobList {
		if job.Name == jobName {
			matchJob = append(matchJob, job)
		} else if strings.Contains(job.Name, jobName) {
			if strings.HasPrefix(job.Name, jobName) {
				prefixJob = append(prefixJob, job)
			} else {
				fuzzyJob = append(fuzzyJob, job)
			}
		}
	}

	for _, job = range matchJob {
		jobList = append(jobList, job)
	}

	for _, job = range prefixJob {
		jobList = append(jobList, job)
	}

	for _, job = range fuzzyJob {
		jobList = append(jobList, job)
	}

	return
}
