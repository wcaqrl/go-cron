package main

import (
	"flag"
	"fmt"
	"go-cron/master"
	"runtime"
)

var (
	confFile string // 配置文件路径
)

// 解析命令行参数
func initArgs() {
	// master -config ./master.json
	flag.StringVar(&confFile, "config", "./master.json", "指定master.json")
	flag.Parse()
}

// 初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var err error
	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 加载配置
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	// 初始化服务发现
	if err = master.InitWorkerMgr(); err != nil {
		goto ERR
	}

	// 日志管理器
	if err = master.InitLogMgr(); err != nil {
		goto ERR
	}

	// 任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	// 配置管理器
	if err = master.InitConfMgr(); err != nil {
		goto ERR
	}

	// 鉴权管理器
	if err = master.InitJwtMgr(); err != nil {
		goto ERR
	}

	// 启动Api服务
	if err = master.InitApiServer();err != nil {
		goto ERR
	}

	select {}
ERR:
	fmt.Println(fmt.Sprintf(err.Error()))
	return
}