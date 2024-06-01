package worker

import (
	"fmt"
	"go-cron/common"
	"math/rand"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// 任务执行器
type Executor struct {
	OutputRegex *regexp.Regexp
}

var (
	G_executor *Executor
)

// 执行一个任务
func (executor *Executor) ExecuteJob (info *common.JobExecuteInfo) {
	go func() {
		var (
			cmd *exec.Cmd
			err error
			output []byte
			result *common.JobExecuteResult
			osBashStr string
			outputStrs, matches []string
			jobLock *common.JobLock
		)
		// 任务执行结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output: make([]byte,0),
		}

		// 初始化分布式锁
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)

		// 记录任务开始时间
		result.StartTime = time.Now()

		// 上锁
		// 随机睡眠(0~1秒)
		time.Sleep(time.Duration(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.Unlock()

		if err != nil { // 上锁失败
			result.Err = err
			result.EndTime = time.Now()
			fmt.Println(fmt.Sprintf("try lock error: %s", err.Error()))
		} else {
			// 上锁成功后,重置任务启动时间
			result.StartTime = time.Now()
			// 执行shell命令
			if runtime.GOOS == "linux" {
				osBashStr = "/bin/bash"
			} else if runtime.GOOS == "windows" {
				osBashStr = "c:\\cygwin64\\bin\\bash.exe"
			}
			cmd = exec.CommandContext(info.CancelCtx,osBashStr,"-c",
				fmt.Sprintf("%s | grep -E '{.*?traceID.*?errorCode.*?data.*?}'", info.Job.Command))

			// 执行并捕获输出
			output, err = cmd.CombinedOutput()
			outputStrs = strings.Split(strings.TrimRight(string(output), "\n"), "\n")
			output = []byte(fmt.Sprintf(`{"traceID":"unknown-trace-id","action":"%s","errorCode":50001,"msg":"without any output","data":[]}`, info.Job.Command))
			if len(outputStrs) > 0 {
				matches = executor.OutputRegex.FindStringSubmatch(outputStrs[len(outputStrs)-1])
				if len(matches) >= 2 {
					output = []byte(strings.Replace(matches[1], "\\", "", -1))
				}
			}
			// 任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
			// 2021-06-18 修改只记录抢锁成功的执行结果
			G_scheduler.PushJobResult(result)
		}
		// 任务完成后, 把执行的结果返回给Scheduler, Scheduler会从executingTable中删除执行记录
		// G_scheduler.PushJobResult(result)
	}()
}

// 初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{
		OutputRegex: regexp.MustCompile(`({.*?traceID.*?"data.*?})`),
	}
	return
}