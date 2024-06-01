package worker

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"go-cron/common"
	"time"
)

type LogSink struct {
	esClient *elastic.Client
	logChan chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

// 批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch){
	fmt.Println(fmt.Sprintf("go to saveLogs"))
	var (
		in interface{}
		err error
		ok bool
		jobLog *common.JobLog
		bulkService = logSink.esClient.Bulk()
		res *elastic.BulkUpdateRequest
	)
	for _, in = range batch.Logs {
		if jobLog, ok = in.(*common.JobLog); !ok || jobLog == nil {
			fmt.Println(fmt.Sprintf("not a job log"))
			continue
		}
		jobLog.IPAddr = G_config.LocalIP
		res = elastic.NewBulkUpdateRequest().Index(G_config.IndexName).Id(uuid.NewV4().String()).Doc(jobLog).DocAsUpsert(true)
		bulkService.Add(res)
	}
	if bulkService.NumberOfActions() > 0 {
		if _, err = bulkService.Do(context.Background()); err != nil {
			fmt.Println(fmt.Sprintf("write log error: %s", err.Error()))
		} else {
			fmt.Println(fmt.Sprintf("write log into elasticsearch success"))
		}
	}
}

// 日志存储协程
func (logSink *LogSink) writeLoop() {
	var (
		jobLog *common.JobLog
		logBatch *common.LogBatch // 当前批次
		commitTimer *time.Timer
		timeoutBatch *common.LogBatch // 超时批次
	)
	for {
		select {
		case jobLog = <- logSink.logChan:
			if logBatch == nil {
				fmt.Println(fmt.Sprintf("empty batch"))
				logBatch = &common.LogBatch{}
				// 让这个批次超时自动提交(给1秒时间)
				commitTimer = time.AfterFunc(
					time.Duration(G_config.JobLogCommitTimeout) * time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch))
			}
			logBatch.Logs = append(logBatch.Logs, jobLog)
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				fmt.Println(fmt.Sprintf("满batch"))
				logSink.saveLogs(logBatch)
				logBatch = nil
				commitTimer.Stop()
			}
		case timeoutBatch = <- logSink.autoCommitChan: // 过期的批次
			// 判断过期批次是否仍旧是当前批次
			if timeoutBatch != logBatch {
				// 说明过期的batch已经被提交过了(到期时刚好满足批次数量)
				continue
			}
			// 把批次写入到mongo中去
			fmt.Println(fmt.Sprintf("write a small batch"))
			logSink.saveLogs(timeoutBatch)
			logBatch = nil
		}
	}
}

func InitLogSink() (err error) {
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

	G_logSink = &LogSink{
		esClient: cli,
		logChan: make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}
	// 启动处理协程
	go G_logSink.writeLoop()
	return
}

// 发送日志
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:
		// 队列满了就丢弃
	}
}