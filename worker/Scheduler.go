package worker

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/log"
	"go-cron/common"
	"time"
)

// 任务调度
type Scheduler struct {
	jobEventChan chan *common.JobEvent // etcd任务事件队列
	jobPlanTable map[string]*common.JobSchedulePlan // 任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo
	jobResultChan chan *common.JobExecuteResult
}

var (
	G_scheduler *Scheduler
)

// 处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
		err error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job)
		if err != nil {
			return
		}
		if jobSchedulePlan == nil {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		} else {
			scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
		}
	case common.JOB_EVENT_DELETE:
		delete(scheduler.jobPlanTable, jobEvent.Job.Name)
	case common.JOB_EVENT_KILL: // 强杀任务事件
		// 取消Command执行, 判断任务是否在执行中
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc() // 触发command杀死shell子进程,任务得到退出
		}
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	// 调度和执行是两件事情
	var (
		err error
		localIP, jobIP string
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting, canExecute bool
	)
	// 如果本机 ip 不在该job的可执行列表中, 则直接返回
	if localIP, err = common.GetLocalIP(); err != nil {
		fmt.Println(fmt.Sprintf("尝试执行时,获取本机 ip 错误: %s", err.Error()))
		return
	}
	for _, jobIP = range jobPlan.Job.IpList {
		if localIP == jobIP {
			canExecute = true
			break
		}
	}
	if !canExecute {
		return
	}

	// 执行的任务可能运行很久,通过jobExcutingTable去重

	// 如果任务正在执行,跳过本次调度
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name];jobExecuting {
		//fmt.Println("尚未退出,跳过执行:",jobPlan.Job.Name)
		return
	}
	// 构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)
	// 保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	fmt.Println("执行任务:",jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan *common.JobSchedulePlan
		now time.Time
		nearTime *time.Time
	)
	scheduleAfter = 1 * time.Second // 预定义调度间隔
	// 如果任务表为空,随便睡眠多久
	if len(scheduler.jobPlanTable) == 0 {
		return
	}
	// 当前时间
	now = time.Now()
	// 遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//fmt.Println(fmt.Sprintf("执行任务: %s", jobPlan.Job.Name))
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) // 更新下次执行时间
		}

		// 统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime){
			nearTime = &jobPlan.NextTime
		}
	}
	// 下次调度间隔(最近要执行的任务调度时间 - 当前时间)
	if nearTime != nil {
		scheduleAfter = (*nearTime).Sub(now)
	}
	return
}

// 处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	var (
		err error
		jobLog *common.JobLog
	)

	// 删除执行状态
	delete(scheduler.jobExecutingTable,result.ExecuteInfo.Job.Name)

	// 生成执行日志
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		var output = &common.Output{}
		if err = json.Unmarshal(result.Output, &output); err != nil {
			log.Error(fmt.Sprintf("unmarshal output error: %s", err.Error()))
			return
		}
		jobLog = &common.JobLog{
			JobName: result.ExecuteInfo.Job.Name,
			Command: result.ExecuteInfo.Job.Command,
			//Output: string(result.Output),
			PlanTime: result.ExecuteInfo.PlanTime.UnixNano()/1e6,
			ScheduleTime: result.ExecuteInfo.RealTime.UnixNano()/1e6,
			StartTime: result.StartTime.UnixNano()/1e6,
			EndTime: result.EndTime.UnixNano()/1e6,
			TraceId: output.TraceID,
			ErrorCode: output.ErrorCode,
			Msg: output.Message,
		}
		if result.Err != nil {
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
		G_logSink.Append(jobLog)
	}
	// fmt.Println("任务执行完成: ",result.ExecuteInfo.Job.Name, string(result.Output), result.Err)
}

// 调度协程
func (scheduler *Scheduler) schedulerLoop() {
	var (
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult *common.JobExecuteResult
	)

	// 初始化一次
	scheduleAfter = scheduler.TrySchedule()

	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	// 定时任务common.Job
	for {
		select {
		case jobEvent = <- scheduler.jobEventChan:
			// 对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		case <- scheduleTimer.C: // 最近的任务到期了
		case jobResult = <- scheduler.jobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		// 调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		// 重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 推送任务变化事件
func(scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

// 初始化任务调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan: make(chan *common.JobEvent, 1000),
		jobPlanTable: make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan: make(chan *common.JobExecuteResult, 1000),
	}
	go G_scheduler.schedulerLoop()
	return
}

// 回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}