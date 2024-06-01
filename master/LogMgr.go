package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"go-cron/common"
	"strings"
	"time"
)

type LogMgr struct {
	esClient *elastic.Client
}

var (
	G_logMgr *LogMgr
)

func InitLogMgr() (err error) {
	var (
		exists bool
		esHost string
		cli *elastic.Client
		info *elastic.PingResult
		code int
		logger = logrus.New()
	)
	logger.SetFormatter(&prefixed.TextFormatter{})
	esHost = fmt.Sprintf("http://%s:%s",G_config.ElasticHost,G_config.ElasticPort)
	if cli, err = elastic.NewClient(
		elastic.SetURL(esHost),
		elastic.SetBasicAuth(G_config.ElasticUsername,G_config.ElasticPassword),
		elastic.SetTraceLog(logger),
	); err != nil {
		return
	}
	if info, code, err = cli.Ping(esHost).Do(context.Background());err != nil {
		return
	}
	fmt.Println(fmt.Sprintf("Elasticsearch returned with code %d and version %s", code, info.Version.Number))

	// 检查日志索引是否存在,不存在则创建
	if exists, err = cli.IndexExists(G_config.IndexName).Do(context.Background()); err != nil {
		return
	}
	if !exists {
		if _, err = cli.CreateIndex(G_config.IndexName).BodyJson(common.LOG_INDEX_SETTING).Do(context.Background()); err != nil {
			return
		}
	}

	G_logMgr = &LogMgr{
		esClient: cli,
	}
	return
}

func (logMgr *LogMgr) ListLog(jobLogFilter common.JobLogFilter, isCount bool) (result interface{}, err error) {
	var (
		searchRes *elastic.SearchResult
		hit *elastic.SearchHit
		boolQuery *elastic.BoolQuery
		rangeQuery *elastic.RangeQuery
		termQuery *elastic.TermQuery
		matchQuery *elastic.MatchQuery
		matchPhraseQuery *elastic.MatchPhraseQuery
		sortField = "start_time"
		sortOrder bool
		logArr = make([]*common.JobLog, 0)
		countMap = map[string]int64{"count": 0}
	)
	// 初始化 boolQuery
	boolQuery = elastic.NewBoolQuery()
	// 如果有ip地址查询
	if jobLogFilter.IpAddr != "" {
		termQuery = elastic.NewTermQuery("ip_addr", jobLogFilter.IpAddr)
		boolQuery.Must(termQuery)
	}
	// 如果有job名称查询
	if jobLogFilter.JobName != "" {
		matchPhraseQuery = elastic.NewMatchPhraseQuery("job_name", jobLogFilter.JobName)
		boolQuery.Must(matchPhraseQuery)
	}
	// 如果有 command 查询
	if jobLogFilter.Comand != "" {
		matchQuery = elastic.NewMatchQuery("command", jobLogFilter.Comand)
		boolQuery.Must(matchQuery)
	}
	// 如果有 traceID 查询
	if jobLogFilter.TraceID != "" {
		termQuery = elastic.NewTermQuery("traceID", jobLogFilter.TraceID)
		boolQuery.Must(termQuery)
	}
	// 如果有 msg 查询
	if jobLogFilter.Msg != "" {
		matchPhraseQuery = elastic.NewMatchPhraseQuery("msg", jobLogFilter.Msg)
		boolQuery.Must(matchPhraseQuery)
	}
	// 如果有 errorCode 查询
	if jobLogFilter.ErrorCode != 0 {
		// 约定 errorCode 为 10000 时筛选 errorCode=0 的结果
		if jobLogFilter.ErrorCode == 10000 {
			termQuery = elastic.NewTermQuery("errorCode", 0)
		} else {
			termQuery = elastic.NewTermQuery("errorCode", jobLogFilter.ErrorCode)
		}
		boolQuery.Must(termQuery)
	}
	// 如果有时间区间查询
	rangeQuery = elastic.NewRangeQuery("start_time").Gte(jobLogFilter.StartTime)
	if jobLogFilter.EndTime > 0 {
		rangeQuery.Lte(jobLogFilter.EndTime)
	}
	boolQuery.Must(rangeQuery)

	if jobLogFilter.SortField != "" {
		sortField = jobLogFilter.SortField
	}
	sortOrder = jobLogFilter.SortOrder

	if searchRes, err = G_logMgr.esClient.Search().
		Index(G_config.IndexName).
		Query(boolQuery).
		From(int((jobLogFilter.Page - 1) * jobLogFilter.Perpage)).
		Size(int(jobLogFilter.Perpage)).
		Sort(sortField, sortOrder). // false 表示逆序
		Do(context.Background()); err != nil {
		fmt.Println(fmt.Sprintf("search jobLog error: %s", err.Error()))
		return
	}

	if isCount {
		countMap["count"] = searchRes.Hits.TotalHits.Value
		result = countMap
	} else {
		for _, hit = range searchRes.Hits.Hits {
			var jobLog common.JobLog
			if err = json.Unmarshal(hit.Source, &jobLog); err != nil {
				fmt.Println(fmt.Sprintf("convert jobLog error: %s", err.Error()))
				continue
			} else {
				logArr = append(logArr, &jobLog)
			}
		}
		result = logArr
	}
	return
}

// 按天数删除日志, 不可删除当天日志
func (logMgr *LogMgr) DeleteLog(day int) (err error) {
	if day < 1 {
		err = common.ERR_INTRADAY_MUST_SAVE
	}
	_, err = G_logMgr.esClient.DeleteByQuery().Index(G_config.IndexName).
		Body(fmt.Sprintf(`{"query":{"range":{"start_time":{"lt":%d}}}}`,
			time.Now().Unix() - 86400 * int64(day))).Do(context.Background())
	return
}

func (logMgr *LogMgr) ListApiLog(apiLogFilter common.ApiLogFilter, isCount bool) (result interface{}, err error) {
	var (
		searchRes *elastic.SearchResult
		hit *elastic.SearchHit
		boolQuery *elastic.BoolQuery
		rangeQuery *elastic.RangeQuery
		termQuery *elastic.TermQuery
		matchPhrasePrefixQuery *elastic.MatchPhrasePrefixQuery
		simpleQueryStringQuery *elastic.SimpleQueryStringQuery
		sortField = "request_time"
		sortOrder bool
		logArr = make([]*common.ApiLog, 0)
		countMap = map[string]int64{"count": 0}
	)
	// 初始化 boolQuery
	boolQuery = elastic.NewBoolQuery()
	// 如果有按id查询
	if apiLogFilter.ID != "" {
		termQuery = elastic.NewTermQuery("id", apiLogFilter.ID)
		boolQuery.Must(termQuery)
	}
	// 如果有domain查询
	if apiLogFilter.Domain != "" {
		termQuery = elastic.NewTermQuery("domain", apiLogFilter.Domain)
		boolQuery.Must(termQuery)
	}
	// 如果有 http_method 查询
	if apiLogFilter.HttpMethod != "" {
		termQuery = elastic.NewTermQuery("http_method", apiLogFilter.HttpMethod)
		boolQuery.Must(termQuery)
	}

	// 如果有 http_code 查询
	if apiLogFilter.Status != 0 {
		termQuery = elastic.NewTermQuery("http_code", apiLogFilter.Status)
		boolQuery.Must(termQuery)
	}

	// 如果有 response_time 查询
	if apiLogFilter.ResponseTime > 0 {
		termQuery = elastic.NewTermQuery("response_time", apiLogFilter.ResponseTime)
		boolQuery.Must(termQuery)
	}

	// 如果有 real_ip 查询
	if apiLogFilter.RealIp != "" {
		termQuery = elastic.NewTermQuery("real_ip", apiLogFilter.RealIp)
		boolQuery.Must(termQuery)
	}

	// 如果有 request_route 查询
	if apiLogFilter.RequestRoute != "" {
		matchPhrasePrefixQuery = elastic.NewMatchPhrasePrefixQuery("request_route", apiLogFilter.RequestRoute)
		boolQuery.Must(matchPhrasePrefixQuery)
	}

	// 如果有 request_param 查询
	if apiLogFilter.RequestParam != "" {
		simpleQueryStringQuery = elastic.NewSimpleQueryStringQuery(apiLogFilter.RequestParam).Field("request_param")
		boolQuery.Must(simpleQueryStringQuery)
	}

	// 如果有时间区间查询
	rangeQuery = elastic.NewRangeQuery("request_time").Gte(apiLogFilter.StartTime)
	if apiLogFilter.EndTime != "" && !strings.HasPrefix(apiLogFilter.EndTime, "1970-01-01"){
		rangeQuery.Lte(apiLogFilter.EndTime)
	}
	boolQuery.Must(rangeQuery)

	if apiLogFilter.SortField != "" {
		sortField = apiLogFilter.SortField
	}
	sortOrder = apiLogFilter.SortOrder

	if searchRes, err = G_logMgr.esClient.Search().
		Index(fmt.Sprintf("%s_log", apiLogFilter.Project)).
		Query(boolQuery).
		From(int((apiLogFilter.Page - 1) * apiLogFilter.Perpage)).
		Size(int(apiLogFilter.Perpage)).
		Sort(sortField, sortOrder). // false 表示逆序
		Do(context.Background()); err != nil {
		fmt.Println(fmt.Sprintf("search %s error: %s",fmt.Sprintf("%s_log", apiLogFilter.Project), err.Error()))
		return
	}

	if isCount {
		countMap["count"] = searchRes.Hits.TotalHits.Value
		result = countMap
	} else {
		for _, hit = range searchRes.Hits.Hits {
			var (
				apiLog common.ApiLog
				apiRowLog common.ApiRowLog
			)
			if err = json.Unmarshal(hit.Source, &apiRowLog); err != nil {
				fmt.Println(fmt.Sprintf("convert api row Log error: %s", err.Error()))
				continue
			} else {
				var (
					tmpTime time.Time
					theTime int64
				)
				if tmpTime, err = time.Parse("2006-01-02T15:04:05+08:00", apiRowLog.TheTime); err != nil {
					fmt.Println(fmt.Sprintf("error: %s", err.Error()))
					continue
				} else {
					theTime = tmpTime.UnixNano()/1e6
				}
				apiLog = common.ApiLog{
					ID:            apiRowLog.ID,
					TheTime:       theTime,
					Domain:        apiRowLog.Domain,
					HttpMethod:    apiRowLog.HttpMethod,
					RequestRoute:  apiRowLog.RequestRoute,
					RequestParam:  apiRowLog.RequestParam,
					Status:        apiRowLog.Status,
					ResponseTime:  apiRowLog.ResponseTime/1e3,
					UserAgent:     apiRowLog.UserAgent,
					RealIp:        apiRowLog.RealIp,
				}
				logArr = append(logArr, &apiLog)
			}
		}
		result = logArr
	}
	return
}

// 按天数删除日志, 不可删除当天日志
func (logMgr *LogMgr) DeleteApiLog(project string, day int) (err error) {
	var (
		tmInt int64
		dateStr string
	)
	if day < 1 {
		err = common.ERR_INTRADAY_MUST_SAVE
	}
	tmInt = common.GetMillisecond(time.Now().In(time.FixedZone("CST", 8*3600)).Unix() - 86400 * int64(day))
	dateStr = common.GetDateStr(tmInt, "2006-01-02T15:04:05+08:00")
	_, err = G_logMgr.esClient.DeleteByQuery().Index(fmt.Sprintf("%s_log", project)).
		Body(fmt.Sprintf(`{"query":{"range":{"request_time":{"lt":%q}}}}`, dateStr)).Do(context.Background())
	return
}
