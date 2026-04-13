package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
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
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	for _, dir := range []string{cfg.Runtime.WorkspaceRoot, cfg.Runtime.LogRoot, cfg.Runtime.CheckpointRoot} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("create directory", "dir", dir, "error", err)
			os.Exit(1)
		}
	}

	db, err := storage.NewDB(cfg.SQLite.Path, cfg.SQLite.WALMode, cfg.SQLite.BusyTimeoutMs)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	repos := storage.NewRepos(db)

	termMgr := terminal.NewManager()

	orch := orchestrator.NewOrchestrator(repos)

	reconciler := scheduler.NewReconciler(repos, termMgr)
	ctx, cancel := context.WithCancel(context.Background())
	if err := reconciler.Run(ctx); err != nil {
		slog.Error("startup reconciler", "error", err)
	}

	sched := scheduler.NewScheduler(cfg, repos, agentruntime.NewGenericShellAdapter(), termMgr)
	go sched.Run(ctx)

	watchdog := scheduler.NewWatchdog(cfg, repos, agentruntime.NewGenericShellAdapter(), termMgr)
	go watchdog.Run(ctx)

	server := api.NewServer(cfg, db, repos, orch, termMgr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(ctx); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

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

	sig := <-sigCh
	slog.Info("received signal, shutting down", "signal", sig)
	cancel()
	sched.Stop()
	watchdog.Stop()
}
