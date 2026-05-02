// Package main 是 AI Dev Platform 的程序入口。
// 负责初始化配置、数据库、调度器、看门狗等核心组件，并启动 HTTP API 服务。
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // 注册 pprof 性能分析 HTTP 端点
	"os"
	"os/signal"
	"syscall"

	"github.com/xxy757/xxyCodingAgents/internal/api"
	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/orchestrator"
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/scheduler"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
	"github.com/xxy757/xxyCodingAgents/internal/workspace"
)

func main() {
	// 解析命令行参数，指定配置文件路径
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// 初始化结构化日志，输出到 stdout，级别为 Info
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// 加载配置文件并应用环境变量覆盖和默认值
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	// 确保运行时所需的工作目录存在
	for _, dir := range []string{cfg.Runtime.WorkspaceRoot, cfg.Runtime.LogRoot, cfg.Runtime.CheckpointRoot} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("create directory", "dir", dir, "error", err)
			os.Exit(1)
		}
	}

	// 初始化 SQLite 数据库连接
	db, err := storage.NewDB(cfg.SQLite.Path, cfg.SQLite.WALMode, cfg.SQLite.BusyTimeoutMs)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 执行数据库迁移，确保所有表和索引已创建
	if err := db.RunMigrations(); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// 初始化数据仓库集合
	repos := storage.NewRepos(db)

	// 初始化 tmux 终端会话管理器
	termMgr := terminal.NewManager()

	// 初始化 Agent 运行时适配器注册表
	adapterRegistry := agentruntime.NewAdapterRegistry()

	// 初始化 Git 工作区管理器
	gitMgr := workspace.NewGitManager(cfg.Runtime.WorkspaceRoot)

	// 初始化编排器，负责任务运行的生命周期管理
	orch := orchestrator.NewOrchestrator(repos, gitMgr)

	// 启动时运行协调器，修复上次异常退出导致的 Agent 状态不一致
	reconciler := scheduler.NewReconciler(repos, termMgr)
	ctx, cancel := context.WithCancel(context.Background())
	if err := reconciler.Run(ctx); err != nil {
		slog.Error("startup reconciler", "error", err)
	}

	// 启动调度器，定时执行任务调度、资源监控和清理
	sched := scheduler.NewScheduler(cfg, repos, adapterRegistry, termMgr, orch)
	go sched.Run(ctx)

	// 启动看门狗，定期检查 Agent 存活状态
	watchdog := scheduler.NewWatchdog(cfg, repos, adapterRegistry, termMgr, sched)
	go watchdog.Run(ctx)

	// 创建 HTTP API 服务器
	server := api.NewServer(cfg, db, repos, orch, termMgr, adapterRegistry)

	// 监听系统信号，用于优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 在后台启动 API 服务器
	go func() {
		if err := server.Start(ctx); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	// 在后台启动 pprof 性能分析服务器
	go func() {
		pprofAddr := cfg.Server.PprofAddr
		if pprofAddr == "" {
			pprofAddr = "localhost:6060"
		}
		slog.Info("pprof server starting", "addr", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			slog.Error("pprof server error", "error", err)
		}
	}()

	// 等待关闭信号，然后依次停止调度器和看门狗
	sig := <-sigCh
	slog.Info("received signal, shutting down", "signal", sig)
	cancel()
	sched.Stop()
	watchdog.Stop()
}
