package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

// 定时任务
type Job struct {
	Name string `json:"name"` // 任务名
	Command string `json:"command"` // shell 命令
	CronExpr string `json:"cronExpr"` // crontab 表达式
	IpList []string `json:"ipList"` // 允许哪些服务器(ip地址)执行该job的
	Status int `json:"status"`
}

// 任务调度计划
type JobSchedulePlan struct {
	Job *Job // 要调度的任务信息
	Expr *cronexpr.Expression // 解析好的cronexp表达式
	NextTime time.Time // 下次执行时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job *Job // 任务信息
	PlanTime time.Time //理论上的调度时间
	RealTime time.Time //实际调度时间
	CancelCtx context.Context // 任务command的context
	CancelFunc context.CancelFunc // 用于取消command执行的cancel函数
}

// HTTP 接口应答
type Response struct {
	Errno int `json:"errorCode"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

type Output struct {
	TraceID string `json:"traceID"`
	Action string `json:"action"`
	ErrorCode int `json:"errorCode"`
	Message string `json:"msg"`
	Data interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // SAVE,DELETE
	Job *Job
}

// 任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo // 执行状态
	Output []byte // 脚本输出
	Err error // 脚本错误原因
	StartTime time.Time // 启动时间
	EndTime time.Time // 结束时间
}

// 任务执行日志
type JobLog struct {
	IPAddr  string `json:"ip_addr" bson:"ip_addr"`
	JobName string `json:"job_name" bson:"job_name"`
	Command string `json:"command" bson:"command"`
	Err string `json:"err" bson:"err"`
	PlanTime int64 `json:"plan_time" bson:"plan_time"`
	ScheduleTime int64 `json:"schedule_time" bson:"schedule_time"`
	StartTime int64 `json:"start_time" bson:"start_time"`
	EndTime int64 `json:"end_time" bson:"end_time"`
	// Output string `json:"output" bson:"output"` //脚本输出
	TraceId string `json:"traceID" bson:"traceID"`
	ErrorCode int `json:"errorCode" bson:"errorCode"`
	Msg string `json:"msg" bson:"msg"`
}

// 日志批次
type LogBatch struct {
	Logs []interface{}
}

// 任务日志过滤条件 查询时用到的过滤字段
type JobLogFilter struct {
	IpAddr    string `json:"ip_addr"`
	JobName   string `json:"job_name" bson:"jobName"`
	Comand    string `json:"command"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	TraceID   string `json:"traceID"`
	ErrorCode int64  `json:"errorCode"`
	Msg       string `json:"msg"`
	Page      int64  `json:"page"`
	Perpage   int64  `json:"perpage"`
	SortField string `json:"sort_field"`
	SortOrder bool   `json:"sort_order"`
}

// api请求日志
type ApiLog struct {
	ID           string     `json:"id" sql:"id"`
	TheTime      int64      `json:"request_time" sql:"request_time"`
	Domain       string     `json:"domain" sql:"domain"`
	HttpMethod   string     `json:"http_method" sql:"http_method"`
	RequestRoute string     `json:"request_route" sql:"request_route"`
	RequestParam string     `json:"request_param" sql:"request_param"`
	Status       int32	    `json:"http_code" sql:"http_code"`
	ResponseTime float64    `json:"response_time" sql:"response_time"`
	UserAgent    string     `json:"user_agent" sql:"user_agent"`
	RealIp       string     `json:"real_ip" sql:"real_ip"`
}

// api Row 请求日志
type ApiRowLog struct {
	ID           string     `json:"id" sql:"id"`
	TheTime      string     `json:"request_time" sql:"request_time"`
	Domain       string     `json:"domain" sql:"domain"`
	HttpMethod   string     `json:"http_method" sql:"http_method"`
	RequestRoute string     `json:"request_route" sql:"request_route"`
	RequestParam string     `json:"request_param" sql:"request_param"`
	Status       int32	    `json:"http_code" sql:"http_code"`
	ResponseTime float64    `json:"response_time" sql:"response_time"`
	UserAgent    string     `json:"user_agent" sql:"user_agent"`
	RealIp       string     `json:"real_ip" sql:"real_ip"`
}


// api请求日志过滤条件 查询时用到的过滤字段
type ApiLogFilter struct {
	Project      string     `json:"project"`
	ID           string     `json:"id"`
	Domain       string     `json:"domain"`
	HttpMethod   string     `json:"http_method"`
	Status       int32	    `json:"http_code"`
	ResponseTime int64      `json:"response_time"`
	RealIp       string     `json:"real_ip"`
	RequestRoute string     `json:"request_route"`
	RequestParam string     `json:"request_param"`
	StartTime    string     `json:"start_time"`
	EndTime      string     `json:"end_time"`
	Page         int64      `json:"page"`
	Perpage      int64      `json:"perpage"`
	SortField    string     `json:"sort_field"`
	SortOrder    bool       `json:"sort_order"`
}

// 任务日志排序条件
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` // {startTime: -1}
}

// 应答方法
func BuildResponse(errno int, msg string, data interface{})(resp []byte, err error) {
	// 1.定义一个response
	var response Response
	response.Errno = errno
	response.Msg = msg
	response.Data = data

	// 2.序列化json
	resp, err = json.Marshal(response)
	return
}

// 反序列化Job
func UnpackJob(value []byte)(ret *Job, err error) {
	var job = &Job{}
	if err = json.Unmarshal(value,job);err != nil {
		return
	}
	ret = job
	return
}

// 从etcd的key中提取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

// 提取worker的ip
func ExtractWorkerIP(regKey string) string {
	return strings.TrimPrefix(regKey, JOB_WORKER_DIR)
}

func BuildJobEvent(eventType int, job *Job)(jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job: job,
	}
}

// 构造任务执行计划
func BuildJobSchedulePlan(job *Job)(jobSchedulePlan *JobSchedulePlan, err error){
	// 不可用状态的任务
	if job.Status != JOB_STATUS_AVAILABLE {
		jobSchedulePlan = nil
		return
	}
	var expr *cronexpr.Expression
	// 解析Job的cron表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	// 生成任务调度计划对象
	jobSchedulePlan = &JobSchedulePlan{
		Job: job,
		Expr: expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

// 构造执行状态信息
func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan)(jobExcuteInfo *JobExecuteInfo) {
	jobExcuteInfo = &JobExecuteInfo{
		Job: jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime,
		RealTime: time.Now(),
	}
	jobExcuteInfo.CancelCtx, jobExcuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}

// 获取本机网卡ip
func GetLocalIP()(ipv4 string, err error) {
	var (
		addrs []net.Addr
		addr net.Addr
		ipNet *net.IPNet
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo(虚拟)的网卡, 需要获取的是物理网卡
	for _, addr = range addrs {
		// 这个网络地址是ip地址 ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback(){
			// 跳过ipv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // 192.168.1.208
				ipv4 = strings.TrimSpace(ipv4)
				return
			}
		}
	}
	err = ERR_NO_LOCAL_IP_FOUND
	return
}

// 将 10 位的时间戳转成 13 位
func GetMillisecond(timeInt int64) int64 {
	var (
		timeIntStr string
		deltaLen int
	)
	// 先将时间转成字符串计算长度
	timeIntStr = strconv.FormatInt(timeInt,10)
	if len(timeIntStr) >= 10 &&  len(timeIntStr) <= 13{
		deltaLen = 13 - len(timeIntStr)
		return timeInt * int64(math.Pow10(deltaLen))
	} else if len(timeIntStr) > 13 {
		deltaLen = len(timeIntStr) - 13
		return int64(float64(timeInt)/math.Pow10(deltaLen))
	}
	return timeInt
}

// 将毫秒时间戳转成指定格式的日期字符串, 如: 2006-01-02T15:04:05+08:00
func GetDateStr(tUnix int64, patt string) string {
	return time.Unix(0, tUnix*int64(time.Millisecond)).In(time.FixedZone("CST", 8*3600)).Format(patt)
}