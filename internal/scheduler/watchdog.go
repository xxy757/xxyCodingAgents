// Package scheduler 的 watchdog 文件实现看门狗机制。
// Watchdog 定期检查所有活跃 Agent 的存活状态，
// 通过 tmux 会话探测、心跳超时和输出超时来检测死亡的 Agent，
// 并对可恢复的 Agent 触发检查点恢复。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

// Watchdog 是 Agent 存活监控器，定期检查并处理死亡的 Agent。
type Watchdog struct {
	cfg             *config.Config
	repos           *storage.Repos
	runtimeRegistry *agentruntime.AdapterRegistry
	terminal        *terminal.Manager
	scheduler       *Scheduler // 用于触发检查点恢复
	stop            chan struct{}
}

// NewWatchdog 创建看门狗实例。
func NewWatchdog(cfg *config.Config, repos *storage.Repos, registry *agentruntime.AdapterRegistry, tm *terminal.Manager, sched *Scheduler) *Watchdog {
	return &Watchdog{
		cfg:             cfg,
		repos:           repos,
		runtimeRegistry: registry,
		terminal:        tm,
		scheduler:       sched,
		stop:            make(chan struct{}),
	}
}

// Run 启动看门狗的定期检查循环。
func (w *Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	slog.Info("watchdog started", "check_interval", "15s")

	for {
		select {
		case <-ctx.Done():
			slog.Info("watchdog stopped")
			return
		case <-w.stop:
			slog.Info("watchdog stopped")
			return
		case <-ticker.C:
			w.check(ctx)
		}
	}
}

// Stop 停止看门狗。
func (w *Watchdog) Stop() {
	close(w.stop)
}

// check 执行一次 Agent 存活检查。
func (w *Watchdog) check(ctx context.Context) {
	agents, err := w.repos.AgentInstances.ListActiveWithTasks()
	if err != nil {
		slog.Error("watchdog: list active agents", "error", err)
		return
	}

	now := time.Now()
	// 获取超时配置
	heartbeatTimeout := time.Duration(w.cfg.Timeouts.HeartbeatTimeoutSeconds) * time.Second
	if heartbeatTimeout == 0 {
		heartbeatTimeout = 30 * time.Second
	}
	outputTimeout := time.Duration(w.cfg.Timeouts.OutputTimeoutSeconds) * time.Second
	if outputTimeout == 0 {
		outputTimeout = 900 * time.Second
	}

	for _, entry := range agents {
		agent := entry.Agent
		task := entry.Task

		// 只检查正在启动或运行中的 Agent
		if agent.Status != domain.AgentStatusRunning && agent.Status != domain.AgentStatusStarting {
			continue
		}

		// 通过运行时接口检查 Agent 进程是否存活
		rt := w.runtimeRegistry.GetOrDefault(agent.AgentKind)
		inspectResult, err := rt.Inspect(ctx, agent.TmuxSession)
		if err != nil {
			slog.Warn("watchdog: inspect agent", "agent_id", agent.ID, "error", err)
			continue
		}

		// 进程已死亡
		if !inspectResult.Running {
			w.handleDeadAgent(ctx, agent, task, "process_crashed")
			continue
		}

		// 更新心跳时间
		w.repos.AgentInstances.UpdateHeartbeat(agent.ID)

		// 仅在有实际输出时更新 last_output_at（避免静默 Agent 的输出超时被掩盖）
		if agent.TmuxSession != "" {
			if output, err := w.terminal.CapturePane(ctx, agent.TmuxSession); err == nil && len(strings.TrimSpace(output)) > 0 {
				w.repos.AgentInstances.UpdateLastOutputAt(agent.ID)
			}
		}

		// 检查心跳超时
		if agent.LastHeartbeatAt != nil {
			elapsed := now.Sub(*agent.LastHeartbeatAt)
			if elapsed > heartbeatTimeout {
				w.handleDeadAgent(ctx, agent, task, "heartbeat_timeout")
				continue
			}
		}

		// 检查输出超时
		if agent.LastOutputAt != nil {
			elapsed := now.Sub(*agent.LastOutputAt)
			if elapsed > outputTimeout {
				w.handleDeadAgent(ctx, agent, task, "output_timeout")
				continue
			}
		}
	}
}

// handleDeadAgent 处理死亡的 Agent，根据重启策略决定标记为失败还是可恢复。
// 若 Agent 有检查点且允许恢复，则触发检查点恢复流程。
func (w *Watchdog) handleDeadAgent(ctx context.Context, agent *domain.AgentInstance, task *domain.Task, reason string) {
	slog.Warn("watchdog: agent dead", "agent_id", agent.ID, "reason", reason, "tmux_session", agent.TmuxSession)

	// 若重启策略允许且有检查点，尝试恢复
	if (task.RestartPolicy == "always" || task.RestartPolicy == "on-failure") && agent.CheckpointID != nil {
		slog.Info("watchdog: attempting checkpoint recovery", "agent_id", agent.ID, "checkpoint_id", *agent.CheckpointID)
		w.repos.AgentInstances.UpdateStatus(agent.ID, domain.AgentStatusRecoverable)
		if w.scheduler != nil {
			if err := w.scheduler.recoverFromCheckpoint(ctx, task.ID); err != nil {
				slog.Error("watchdog: recovery failed, marking as failed", "agent_id", agent.ID, "error", err)
				w.repos.AgentInstances.UpdateStatus(agent.ID, domain.AgentStatusFailed)
				w.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
				w.scheduler.cleanupTaskArtifacts(task.ID)
			} else {
				return // 恢复成功，不需要进一步处理
			}
		}
	}

	// 无法恢复，标记为失败
	newStatus := domain.AgentStatusFailed
	if task.RestartPolicy == "always" || task.RestartPolicy == "on-failure" {
		newStatus = domain.AgentStatusRecoverable
	}

	w.repos.AgentInstances.UpdateStatus(agent.ID, newStatus)
	w.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)

	// 记录 Agent 失败事件
	w.repos.Events.Create(&domain.Event{
		ID:        uuid.New().String(),
		RunID:     agent.RunID,
		TaskID:    ptrString(task.ID),
		AgentID:   ptrString(agent.ID),
		EventType: "agent_failed",
		Message:   fmt.Sprintf("Agent %s marked as %s: %s", agent.ID[:8], newStatus, reason),
		Metadata:  fmt.Sprintf(`{"reason":"%s","tmux_session":"%s","restart_policy":"%s"}`, reason, agent.TmuxSession, task.RestartPolicy),
		CreatedAt: time.Now(),
	})
	if w.scheduler != nil {
		w.scheduler.cleanupTaskArtifacts(task.ID)
	}
}
