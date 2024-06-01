package common

const (
	// 任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	// 任务操作目录
	JOB_ACTION_DIR = "/cron/action/"

	// 任务强杀目录
	JOB_KILLER_DIR = "/cron/killer/"

	// 任务锁目录
	JOB_LOCK_DIR = "/cron/lock/"

	// 服务注册目录
	JOB_WORKER_DIR = "/cron/workers/"

	// api配置文件目录
	API_CONF_DIR = "/api/conf/"

	// 保存任务事件
	JOB_EVENT_SAVE = 1

	// 删除任务事件
	JOB_EVENT_DELETE = 2

	// 强杀任务事件
	JOB_EVENT_KILL = 3

	// token保存目录
	TOKEN_CACHE_DIR = "/token/"

	JOB_STATUS_AVAILABLE = 1

	// 日志索引mapping
	// 		  "aliases": {
	//			"%s": {}
	//		  },
	LOG_INDEX_SETTING = `
		{
		  "settings": {
			"index": {
			  "number_of_shards": 3,
			  "number_of_replicas": 0
			}
		  },
		  "mappings": {
			"properties": {
			  "ip_addr": {
				"type": "ip"
			  },
			  "job_name": {
				"type": "text"
			  },
			  "command": {
				"type": "text"
			  },
			  "err": {
				"type": "text"
			  },
			  "plan_time": {
				"type": "date",
				"format": "strict_date_optional_time||epoch_millis||yyyy-MM-dd HH:mm:ss.SSS"
			  },
			  "schedule_time": {
				"type": "date",
				"format": "strict_date_optional_time||epoch_millis||yyyy-MM-dd HH:mm:ss.SSS"
			  },
			  "start_time": {
				"type": "date",
				"format": "strict_date_optional_time||epoch_millis||yyyy-MM-dd HH:mm:ss.SSS"
			  },
			  "end_time": {
				"type": "date",
				"format": "strict_date_optional_time||epoch_millis||yyyy-MM-dd HH:mm:ss.SSS"
			  },
			  "traceID": {
				"type": "keyword"
			  },
			  "errorCode": {
				"type": "integer"
			  },
			  "msg": {
				"type": "text"
			  }
			}
		  }
		}
	`
)