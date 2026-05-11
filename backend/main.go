package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"backend/compliance"
	"backend/core"
	"backend/global"
	"backend/models"
	"backend/routers"
)

func init() {

	core.InitConfig()

	// 初始化日志
	global.Log = core.InitLogger()

	// 初始化数据库
	global.DB = core.InitXorm()

	// 初始化 Redis
	global.Redis = core.InitRedis()

	// 同步数据库
	err := core.SyncWithLog(global.DB, models.GetModels()...)
	if err != nil {
		global.Log.Error("同步数据库失败", "error", err)
		panic(err)
	}

	// 初始化默认数据
	if err := core.InitDefaultCategories(); err != nil {
		global.Log.Error("初始化默认分类失败", "error", err)
		panic(err)
	}

	// 确保全文索引（用于Hybrid检索预选）
	core.EnsureFullTextIndex()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var complianceService *compliance.Service
	var complianceWorker *compliance.Worker
	if global.DB != nil {
		service, err := compliance.NewService(global.DB, global.Redis, global.Config.Compliance)
		if err != nil {
			if global.Log != nil {
				global.Log.Error("初始化合规服务失败", "error", err)
			}
		} else {
			complianceService = service
			if err := service.EnsureReportDir(); err != nil && global.Log != nil {
				global.Log.Warn("创建合规报告目录失败", "error", err)
			}
			complianceWorker = compliance.NewWorker(service)
			if err := complianceWorker.Start(ctx); err != nil && global.Log != nil {
				global.Log.Warn("合规 Worker 启动失败", "error", err)
			} else if complianceWorker != nil {
				defer complianceWorker.Shutdown(context.Background())
			}
		}
	}

	addr := ":6636"
	h := routers.InitRouter(addr, complianceService)

	global.Log.Info("服务启动成功", "address", addr)
	h.Spin()
}
