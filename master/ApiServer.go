package master

import (
	"encoding/json"
	"errors"
	"go-cron/common"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 单例对象
var G_apiServer *ApiServer

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

type middleware func(http.Handler) http.Handler

type Router struct {
	middlewareChain [] middleware
	mux map[string] http.Handler
}

func NewRouter() *Router{
	return &Router{
		middlewareChain: make([]middleware, 0),
		mux: make(map[string] http.Handler),
	}
}

func (r *Router) Use(m middleware) {
	r.middlewareChain = append(r.middlewareChain, m)
}

func (r *Router) Add(route string, h http.Handler) {
	var (
		i int
		mergedHandler = h
	)
	for i = len(r.middlewareChain) - 1; i >= 0; i-- {
		mergedHandler = r.middlewareChain[i](mergedHandler)
	}
	r.mux[route] = mergedHandler
}

// POST 处理登录
// POST /login
func handleLogin(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		username, password string
		bytes []byte
		token string
		data map[string]interface{}
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 获取表单中的用户名、密码
	username = req.PostFormValue("username")
	password = req.PostFormValue("password")
	if username == G_config.Username && password == G_config.Password {
		// 生成 jwt token
		if token, err = GenToken(); err != nil {
			goto ERR
		}
		// 存储 token
		if _, err = G_jwtMgr.SaveToken(token, token); err != nil {
			goto ERR
		}
		// 返回正常应答
		data = make(map[string]interface{})
		data["username"] = username
		data["token"]    = token
		data["expired"]  = time.Now().Add(time.Duration(G_config.TokenExpiration)*time.Second).Unix()
		if bytes, err = common.BuildResponse(0, "ok", data); err == nil {
			resp.Write(bytes)
		}
	} else {
		err = errors.New("username or password invalid")
		goto ERR
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// GET 处理退出登录
// GET /logout
func handleLogout(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		bytes []byte
		claims *CustomClaims
	)
	if claims, err = ParseToken(req.Header.Get("Authorization")); err != nil {
		goto ERR
	}
	if claims.Username != G_config.Username {
		err = errors.New("not match system username")
		goto ERR
	}
	// 设置退出
	if _, err = G_jwtMgr.DeleteToken(req.Header.Get("Authorization")); err != nil {
		goto ERR
	}
	// 返回正常应答
	if bytes, err = common.BuildResponse(0, "ok", nil); err == nil {
		resp.Write(bytes)
	} else {
		goto ERR
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 保存任务接口
// POST /job/save job = {"name":"job1", "command":"echo hello","cronExpr":"*****", "ipList": ["172.17.12.100", "172.17.5.63"]}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		postJob, ipStr string
		job common.Job
		ipSlice = make([]string, 0)
		oldJob *common.Job
		bytes []byte
	)
	// 1.解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 2.取表单中的job字段
	postJob = req.PostForm.Get("job")
	// 3.反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil{
		goto ERR
	}
	// 4.判断任务的可执行服务器(ip地址)不能为空
	// 清洗 ip
	for _, ipStr = range job.IpList {
		if ipStr != "" {
			ipSlice = append(ipSlice, ipStr)
		}
	}
	job.IpList = ipSlice
	if job.IpList == nil || len(job.IpList) == 0 {
		err = common.ERR_EMPTY_EXEC_IP
		goto ERR
	}

	// 4.保存到etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	// 5. 返回正常应答
	if bytes, err = common.BuildResponse(0, "ok", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	// 6. 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 删除任务接口
// GET /job/delete name=job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		name string
		oldJob *common.Job
		bytes []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 删除的任务名
	name = req.Form.Get("name")
	if name == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	// 去删除任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}
	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取任务列表
// GET /job/list?name=job1&page=1&perpage=15
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		err error
		bytes []byte
		page int64 = 1
		perpage int64 = 15
		getPage, getPerpage int64
		jobName, pageStr, perpageStr string
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 获取请求参数
	pageStr = req.Form.Get("page")
	if pageStr != "" {
		if getPage, err = strconv.ParseInt(pageStr, 10 , 64); err != nil {
			goto ERR
		}
	}
	if getPage > 0 {
		page = getPage
	}
	perpageStr = req.Form.Get("perpage")
	if perpageStr != "" {
		if getPerpage, err = strconv.ParseInt(perpageStr, 10, 64); err != nil {
			goto ERR
		}
	}
	if getPerpage > 0 {
		perpage = getPerpage
	}

	jobName = req.Form.Get("name")
	// 获取任务列表
	/**
	if jobList, err = G_jobMgr.ListJob(jobName, page, perpage); err != nil {
		goto ERR
	}
	*/

	// 获取模糊匹配任务列表
	if jobList, err = G_jobMgr.ListFuzzyJob(jobName, page, perpage); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", jobList); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 统计任务数量
// GET /job/count?name=job1
func handleJobCount(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		bytes []byte
		jobName string
		countMap map[string]int64
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	jobName = req.Form.Get("name")
	// 获取任务数量
	/**
	if countMap, err = G_jobMgr.CountJob(jobName); err != nil {
		goto ERR
	}
	*/

	// 获取模糊匹配任务总数
	if countMap, err = G_jobMgr.CountFuzzyJob(jobName); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", countMap); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取任务详情
// GET /job/detail
func handleJobDetail(resp http.ResponseWriter, req *http.Request) {
	var (
		job *common.Job
		err error
		bytes []byte
		jobName string
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	jobName = req.Form.Get("name")
	if jobName == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	// 获取任务列表
	if job, err = G_jobMgr.DetailJob(jobName); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", job); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 保存api配置接口
// POST /conf/save  conf = {"orange": "text 字符串 app.env=dev mysql.port=3306"}
func handleConfSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		postConf, oldConf, confKey, confStr string
		bytes []byte
		confMap = make(map[string]string)
	)
	// 1.解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 2.取表单中的conf字段
	postConf = req.PostForm.Get("conf")
	// 3.反序列化conf
	if err = json.Unmarshal([]byte(postConf), &confMap); err != nil{
		goto ERR
	}
	for confKey, confStr = range confMap {break}
	// 4.保存到etcd
	if oldConf, err = G_confMgr.SaveConfig(confKey, confStr); err != nil {
		goto ERR
	}
	// 5. 返回正常应答
	if bytes, err = common.BuildResponse(0, "ok", oldConf); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	// 6. 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 删除api配置接口
// GET /conf/delete name=job1
func handleConfDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		confKey, oldConf string
		bytes []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	// 删除的配置名
	confKey = req.Form.Get("key")
	if confKey == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	// 去删除配置
	if oldConf, err = G_confMgr.DeleteConfig(confKey); err != nil {
		goto ERR
	}
	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", oldConf); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取api配置列表
// GET /conf/list?key=orange&page=1&perpage=15
func handleConfList(resp http.ResponseWriter, req *http.Request) {
	var (
		confArr []map[string]string
		err error
		bytes []byte
		page int64 = 1
		perpage int64 = 15
		getPage, getPerpage int64
		confKey, pageStr, perpageStr string
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 获取请求参数
	pageStr = req.Form.Get("page")
	if pageStr != "" {
		if getPage, err = strconv.ParseInt(pageStr, 10 , 64); err != nil {
			goto ERR
		}
	}
	if getPage > 0 {
		page = getPage
	}
	perpageStr = req.Form.Get("perpage")
	if perpageStr != "" {
		if getPerpage, err = strconv.ParseInt(perpageStr, 10, 64); err != nil {
			goto ERR
		}
	}
	if getPerpage > 0 {
		perpage = getPerpage
	}

	confKey = req.Form.Get("key")
	// 获取任务列表
	if confArr, err = G_confMgr.ListConfig(confKey, page, perpage); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", confArr); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 统计api配置数量
// GET /conf/count?key=orange
func handleConfCount(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		bytes []byte
		confKey string
		countMap map[string]int64
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	confKey = req.Form.Get("key")
	// 获取任务列表
	if countMap, err = G_confMgr.CountConfig(confKey); err != nil {
		goto ERR
	}
	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", countMap); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取api配置详情
// GET /conf/detail
func handleConfDetail(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		bytes []byte
		confKey string
		confMap map[string]string
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	confKey = req.Form.Get("key")
	if confKey == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	// 获取配置详情
	if confMap, err = G_confMgr.DetailConfig(confKey); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "ok", confMap); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 杀死任务
// POST /job/kill?name=job1
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		name string
		bytes []byte
	)
	// 解析url参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 要杀死的任务名
	name = req.Form.Get("name")
	if name == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	// 杀死任务
	if err = G_jobMgr.KillJob(name);err != nil {
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "ok", nil); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 立即执行任务
// POST /job/exec?name=job1
func handleJobExec(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		name string
		bytes []byte
	)
	// 解析url参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 要立即执行的任务名
	name = req.Form.Get("name")
	if name == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}

	// 修改etcd执行时间并立即执行
	if err = G_jobMgr.ExecJob(name); err != nil {
		goto ERR
	}
	// 返回正常应答
	if bytes, err = common.BuildResponse(0, "ok", nil); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取健康worker节点列表
// GET /worker/list
func handleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		workerArr []string
		err error
		bytes []byte
	)
	if workerArr, err = G_workerMgr.ListWorkers(); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "ok", workerArr); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 查询任务日志
// GET /job/log/list?name=job10&page=1&perpage=15&start_time=1600098798760&end_time=1600098798760&sort=start_time.desc
func handleJobLogList(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		ok bool
		in interface{}
		logArr []*common.JobLog
		bytes []byte
		jobLogFilter common.JobLogFilter
	)
	if jobLogFilter, err = arrangeJobFilter(req); err != nil {
		goto ERR
	}
	if in, err = G_logMgr.ListLog(jobLogFilter, false); err != nil {
		goto ERR
	}

	if logArr, ok = in.([]*common.JobLog); !ok {
		err = common.ERR_ASSERT_WRONG
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "ok", logArr); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

func arrangeJobFilter(req *http.Request) (jobLogFilter common.JobLogFilter, err error) {
	var (
		pageStr, perpageStr,startTimeStr, endTimeStr, errCodeStr, sortStr, sortField  string // 任务名字
		sortOrder bool
		sortSlice []string
		page int64 = 1
		perpage int64 = 15
		getPage, getPerpage, getStartTime, getEndTime, getErrCode, startTime, endTime, errCode int64
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		return
	}
	// 获取请求参数
	pageStr = req.Form.Get("page")
	if pageStr != "" {
		if getPage, err = strconv.ParseInt(pageStr, 10 , 64); err != nil {
			return
		}
	}
	if getPage > 0 {
		page = getPage
	}
	perpageStr = req.Form.Get("perpage")
	if perpageStr != "" {
		if getPerpage, err = strconv.ParseInt(perpageStr, 10, 64); err != nil {
			return
		}
	}
	if getPerpage > 0 {
		perpage = getPerpage
	}
	startTimeStr = req.Form.Get("start_time")
	if startTimeStr != "" {
		if getStartTime, err = strconv.ParseInt(startTimeStr, 10, 64); err != nil {
			return
		}
	}
	if getStartTime > 0 {
		startTime = common.GetMillisecond(getStartTime)
	}
	endTimeStr = req.Form.Get("end_time")
	if endTimeStr != "" {
		if getEndTime, err = strconv.ParseInt(endTimeStr, 10, 64); err != nil {
			return
		}
	}
	if getEndTime > 0 {
		endTime = common.GetMillisecond(getEndTime)
	}
	errCodeStr = req.Form.Get("errorCode")
	if errCodeStr != "" {
		if getErrCode, err = strconv.ParseInt(errCodeStr, 10, 64); err != nil {
			return
		}
	}
	if getErrCode > 0 {
		errCode = getErrCode
	}
	sortStr = req.Form.Get("sort")
	sortStr = strings.TrimSpace(sortStr)
	if sortStr != "" {
		sortSlice = strings.Split(sortStr, ".")
		if len(sortSlice) == 2 {
			if strings.ToLower(sortSlice[1]) == "asc" {
				sortField = sortSlice[0]
				sortOrder = true

			} else if strings.ToLower(sortSlice[1]) == "desc" {
				sortField = sortSlice[0]
				sortOrder = false
			}
		}
	}

	jobLogFilter = common.JobLogFilter{
		IpAddr: req.Form.Get("ip_addr"),
		JobName: req.Form.Get("name"),
		Comand: req.Form.Get("command"),
		StartTime: startTime,
		EndTime: endTime,
		TraceID: req.Form.Get("traceID"),
		ErrorCode: errCode,
		Msg: req.Form.Get("msg"),
		Page: page,
		Perpage: perpage,
		SortField: sortField,
		SortOrder: sortOrder,
	}
	return
}

// 查询任务日志数量
// GET /job/log/count?name=job10&page=1&perpage=15&start_time=1600098798760&end_time=1600098798760
func handleJobLogCount(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		ok bool
		in interface{}
		countMap map[string]int64
		bytes []byte
		jobLogFilter common.JobLogFilter
	)
	if jobLogFilter, err = arrangeJobFilter(req); err != nil {
		goto ERR
	}
	if in, err = G_logMgr.ListLog(jobLogFilter, true); err != nil {
		goto ERR
	}

	if countMap, ok = in.(map[string]int64); !ok {
		err = common.ERR_ASSERT_WRONG
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "ok", countMap); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

// 保留指定天数内的日志, 删除旧日志
// GET /job/log/delete
func handleJobLogDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		daysParam string // 保留天数
		days int
		bytes []byte
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 获取请求参数 /log/delete?days=7
	daysParam = req.Form.Get("days")
	if daysParam == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	if days, err = strconv.Atoi(daysParam); err != nil {
		goto ERR
	}
	if err = G_logMgr.DeleteLog(days); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "ok", nil); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

// 查询api日志
// GET /api/log/list?
// project=orange&
// id=xxxx&
// domain=xxx&
// http_method=xxx&
// http_code=xxx&
// response_time=123&
// real_ip=xxx&
// request_route=xxx&
// request_param=xxx&
// page=1&perpage=15&start_time=1600098798760&end_time=1600098798760&
// sort=request_time.asc
func handleApiLogList(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		ok bool
		in interface{}
		logArr []*common.ApiLog
		bytes []byte
		apiLogFilter common.ApiLogFilter
	)

	if apiLogFilter, err = arrangeApiFilter(req); err != nil {
		goto ERR
	}
	if in, err = G_logMgr.ListApiLog(apiLogFilter, false); err != nil {
		goto ERR
	}

	if logArr, ok = in.([]*common.ApiLog); !ok {
		err = common.ERR_ASSERT_WRONG
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "ok", logArr); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

func arrangeApiFilter(req *http.Request) (apiLogFilter common.ApiLogFilter, err error) {
	var (
		projectStr, project, pageStr, perpageStr, respTimeStr, statusCodeStr, sortStr, sortField, startTimeStr, endTimeStr  string // 任务名字
		sortOrder, ok  bool
		sortSlice []string
		statusCode int32
		page int64 = 1
		perpage int64 = 15
		getRespTime float64
		getPage, getPerpage, respTime, startTime, endTime, getStartTime, getEndTime, getStatusCode  int64
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		return
	}
	// 获取请求参数
	project = strings.TrimSpace(req.Form.Get("project"))
	for _, projectStr = range G_config.ProjectNames {
		if projectStr == project {
			ok = true
			break
		}
	}
	if !ok {
		err = common.ERR_INVALID_PARAMS
		return
	}
	pageStr = req.Form.Get("page")
	if pageStr != "" {
		if getPage, err = strconv.ParseInt(pageStr, 10 , 64); err != nil {
			return
		}
	}
	if getPage > 0 {
		page = getPage
	}
	perpageStr = req.Form.Get("perpage")
	if perpageStr != "" {
		if getPerpage, err = strconv.ParseInt(perpageStr, 10, 64); err != nil {
			return
		}
	}
	if getPerpage > 0 {
		perpage = getPerpage
	}
	startTimeStr = req.Form.Get("start_time")
	if startTimeStr != "" {
		if getStartTime, err = strconv.ParseInt(startTimeStr, 10, 64); err != nil {
			return
		}
	}
	if getStartTime > 0 {
		startTime = common.GetMillisecond(getStartTime)
	}
	endTimeStr = req.Form.Get("end_time")
	if endTimeStr != "" {
		if getEndTime, err = strconv.ParseInt(endTimeStr, 10, 64); err != nil {
			return
		}
	}
	if getEndTime > 0 {
		endTime = common.GetMillisecond(getEndTime)
	}
	respTimeStr = req.Form.Get("response_time")
	if respTimeStr != "" {
		if getRespTime, err = strconv.ParseFloat(respTimeStr, 64); err != nil {
			return
		}
	}
	if getRespTime > 0 {
		respTime = int64(getRespTime * 1e3)
	}
	statusCodeStr = req.Form.Get("http_code")
	if statusCodeStr != "" {
		if getStatusCode, err = strconv.ParseInt(statusCodeStr, 10, 32); err != nil {
			return
		}
	}
	if getStatusCode > 0 {
		statusCode = int32(getStatusCode)
	}
	sortStr = req.Form.Get("sort")
	sortStr = strings.TrimSpace(sortStr)
	if sortStr != "" {
		sortSlice = strings.Split(sortStr, ".")
		if len(sortSlice) == 2 {
			if strings.ToLower(sortSlice[1]) == "asc" {
				sortField = sortSlice[0]
				sortOrder = true

			} else if strings.ToLower(sortSlice[1]) == "desc" {
				sortField = sortSlice[0]
				sortOrder = false
			}
		}
	}

	apiLogFilter = common.ApiLogFilter{
		Project: project,
		ID: req.Form.Get("id"),
		Domain: req.Form.Get("domain"),
		HttpMethod: strings.ToUpper(req.Form.Get("http_method")),
		Status: statusCode,
		ResponseTime: respTime,
		RealIp: req.Form.Get("real_ip"),
		RequestRoute: strings.TrimSpace(req.Form.Get("request_route")),
		RequestParam: strings.TrimSpace(req.Form.Get("request_param")),
		StartTime: common.GetDateStr(startTime, "2006-01-02T15:04:05+08:00"),
		EndTime: common.GetDateStr(endTime, "2006-01-02T15:04:05+08:00"),
		Page: page,
		Perpage: perpage,
		SortField: sortField,
		SortOrder: sortOrder,
	}
	return
}

// 查询任务日志数量
// GET /api/log/count?
// project=orange&
// id=xxxx&
// domain=xxx&
// http_method=xxx&
// http_code=xxx&
// response_time=123&
// real_ip=xxx
func handleApiLogCount(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		ok bool
		in interface{}
		countMap map[string]int64
		bytes []byte
		apiLogFilter common.ApiLogFilter
	)
	if apiLogFilter, err = arrangeApiFilter(req); err != nil {
		goto ERR
	}
	if in, err = G_logMgr.ListApiLog(apiLogFilter, true); err != nil {
		goto ERR
	}

	if countMap, ok = in.(map[string]int64); !ok {
		err = common.ERR_ASSERT_WRONG
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "ok", countMap); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}

// 保留指定天数内的日志, 删除旧日志
// GET /api/log/delete?project=orange&days=7
func handleApiLogDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		ok bool
		daysParam, project, projectStr string // 保留天数
		days int
		bytes []byte
	)
	// 解析GET参数
	if err = req.ParseForm();err != nil {
		goto ERR
	}
	// 获取请求参数
	project = strings.TrimSpace(req.Form.Get("project"))
	for _, projectStr = range G_config.ProjectNames {
		if projectStr == project {
			ok = true
			break
		}
	}
	if !ok {
		err = common.ERR_INVALID_PARAMS
		goto ERR
	}
	// 获取请求参数 /log/delete?project=orange&days=7
	daysParam = req.Form.Get("days")
	if daysParam == "" {
		err = common.ERR_EMPTY_PARAMS
		goto ERR
	}
	if days, err = strconv.Atoi(daysParam); err != nil {
		goto ERR
	}
	if err = G_logMgr.DeleteApiLog(project, days); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "ok", nil); err == nil {
		resp.Write(bytes)
	}

	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}

}


// 跨域请求中间件
//func CorsInterceptor(h http.HandlerFunc) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Access-Control-Allow-Origin", "*")  // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
//		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
//		w.Header().Add("Access-Control-Allow-Credentials", "true") //设置为true，允许ajax异步请求带cookie信息
//		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE") //允许请求方法
//		w.Header().Set("content-type", "application/json;charset=UTF-8")             //返回数据格式是json
//		if r.Method == "OPTIONS" {
//			w.WriteHeader(http.StatusNoContent)
//			return
//		}
//		h(w, r)
//	}
//}

// jwt token 中间件
//func HttpInterceptor(h http.HandlerFunc) http.HandlerFunc {
//	return http.HandlerFunc(
//		func(w http.ResponseWriter, r *http.Request) {
//			var (
//				claims *CustomClaims
//				err error
//				token string
//				bytes []byte
//			)
//			// 获取实际token
//			if token, err = G_jwtMgr.GetToken(r.Header.Get("Authorization")); err != nil || token == "" {
//				err = common.ERR_GET_EMPTY_TOKEN
//				goto ERR
//			}
//			if claims, err = ParseToken(token); err != nil {
//				goto ERR
//			}
//			if claims.Username != G_config.Username {
//				err = common.ERR_DIFF_USERNAME
//				goto ERR
//			}
//			// 如果真实token已过期
//			if time.Now().Unix() > claims.ExpiresAt + int64(G_config.TokenExpiration) {
//				err = common.ERR_TOKEN_EXPIRED
//				goto ERR
//			}
//			// 指定时间范围需重设token
//			if time.Now().Unix() > claims.ExpiresAt {
//				if token, err = GenToken(); err != nil {
//					goto ERR
//				}
//				// 设置新的token
//				if token, err = G_jwtMgr.SaveToken(r.Header.Get("Authorization"), token); err != nil {
//					goto ERR
//				}
//			}
//			h(w, r)
//		ERR:
//			if err != nil {
//				w.WriteHeader(http.StatusForbidden)
//				if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
//					w.Write(bytes)
//				}
//				return
//			}
//	})
//}

//func timeInterceptor(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
//		timeStart := time.Now()
//		// next handler
//		next.ServeHTTP(wr, r)
//		timeElapsed := time.Since(timeStart)
//		fmt.Println(timeElapsed)
//	})
//}

func CorsInterceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")  // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
		w.Header().Add("Access-Control-Allow-Credentials", "true") //设置为true，允许ajax异步请求带cookie信息
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, TRACE, HEAD, CONNECT") //允许请求方法
		w.Header().Set("content-type", "application/json;charset=UTF-8")             //返回数据格式是json
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func TokenInterceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var (
				claims *CustomClaims
				err error
				token string
				bytes []byte
			)
			// 获取实际token
			token = strings.TrimSpace(strings.Replace(r.Header.Get("Authorization"), "Bearer", "", -1))
			if token, err = G_jwtMgr.GetToken(token); err != nil || token == "" {
				err = common.ERR_GET_EMPTY_TOKEN
				goto ERR
			}
			if claims, err = ParseToken(token); err != nil {
				goto ERR
			}
			if claims.Username != G_config.Username {
				err = common.ERR_DIFF_USERNAME
				goto ERR
			}
			// 如果真实token已过期
			if time.Now().Unix() > claims.ExpiresAt + int64(G_config.TokenExpiration) {
				err = common.ERR_TOKEN_EXPIRED
				goto ERR
			}
			// 指定时间范围需重设token
			if time.Now().Unix() > claims.ExpiresAt {
				if token, err = GenToken(); err != nil {
					goto ERR
				}
				// 设置新的token
				if token, err = G_jwtMgr.SaveToken(r.Header.Get("Authorization"), token); err != nil {
					goto ERR
				}
			}
			h.ServeHTTP(w, r)
		ERR:
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
					w.Write(bytes)
				}
				return
			}
		})
}

func ParamsInterceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var (
				err error
				bytes []byte
				vals = r.URL.Query()  //url.Values
				key string
			)
			// 解析GET参数
			if err = r.ParseForm();err != nil {
				goto ERR
			}
			// 禁止客户端传递null
			for key = range vals {
				if strings.ToLower(r.Form.Get(key)) == "null" {
					err = common.ERR_EMPTY_PARAMS
					goto ERR
				}
			}
			h.ServeHTTP(w, r)
		ERR:
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
					w.Write(bytes)
				}
				return
			}
		})
}

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux *http.ServeMux
		r *Router
		listener net.Listener
		httpServer *http.Server
		k string
		staticDir http.Dir
		staticHandler, h http.Handler // 静态文件的http回调
	)
	// 配置路由
	mux = http.NewServeMux()
	r = NewRouter()
	r.Use(CorsInterceptor)
	r.Add("/login", http.HandlerFunc(handleLogin))
	r.Use(TokenInterceptor)
	r.Use(ParamsInterceptor)
	r.Add("/logout", http.HandlerFunc(handleLogout))
	r.Add("/job/save", http.HandlerFunc(handleJobSave))
	r.Add("/job/delete", http.HandlerFunc(handleJobDelete))
	r.Add("/job/detail", http.HandlerFunc(handleJobDetail))
	r.Add("/job/list", http.HandlerFunc(handleJobList))
	r.Add("/job/count", http.HandlerFunc(handleJobCount))
	r.Add("/job/kill", http.HandlerFunc(handleJobKill))
	r.Add("/job/exec", http.HandlerFunc(handleJobExec))
	r.Add("/job/log/list", http.HandlerFunc(handleJobLogList))
	r.Add("/job/log/count", http.HandlerFunc(handleJobLogCount))
	r.Add("/job/log/delete", http.HandlerFunc(handleJobLogDelete))
	r.Add("/api/log/list", http.HandlerFunc(handleApiLogList))
	r.Add("/api/log/count", http.HandlerFunc(handleApiLogCount))
	r.Add("/api/log/delete", http.HandlerFunc(handleApiLogDelete))
	r.Add("/worker/list", http.HandlerFunc(handleWorkerList))
	r.Add("/conf/save", http.HandlerFunc(handleConfSave))
	r.Add("/conf/delete", http.HandlerFunc(handleConfDelete))
	r.Add("/conf/detail", http.HandlerFunc(handleConfDetail))
	r.Add("/conf/list", http.HandlerFunc(handleConfList))
	r.Add("/conf/count", http.HandlerFunc(handleConfCount))
	for k, h = range r.mux { mux.Handle(k, h) }

	// 静态文件目录
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/",http.StripPrefix("/",staticHandler))
	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout: time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler: mux,
	}

	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	// 启动TCP监听
	if listener, err = net.Listen("tcp", ":" + strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	// 启动服务端
	go httpServer.Serve(listener)

	return
}