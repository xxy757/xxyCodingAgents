// Package scheduler 的 reconciler 文件实现启动时的状态协调器。
// 当服务重启时，Reconciler 会检查所有"活跃"状态的 Agent 实例，
// 通过探测 tmux 会话是否仍然存活来修正不一致的状态。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

// Reconciler 是启动时的状态协调器，修复因异常退出导致的状态不一致。
type Reconciler struct {
	repos    *storage.Repos
	terminal TerminalChecker
}

// TerminalChecker 定义检查终端会话是否存在的接口。
type TerminalChecker interface {
	SessionExists(ctx context.Context, name string) bool
}

// NewReconciler 创建协调器实例。
func NewReconciler(repos *storage.Repos, terminal TerminalChecker) *Reconciler {
	return &Reconciler{
		repos:    repos,
		terminal: terminal,
	}
}

// Run 执行一次性的启动协调，检查所有活跃 Agent 并修正状态。
func (r *Reconciler) Run(ctx context.Context) error {
	slog.Info("startup reconciler starting")
	defer slog.Info("startup reconciler completed")

	// 需要检查的活跃状态列表
	activeStatuses := []domain.AgentInstanceStatus{
		domain.AgentStatusStarting,
		domain.AgentStatusRunning,
		domain.AgentStatusPaused,
	}

	for _, status := range activeStatuses {
		agents, err := r.repos.AgentInstances.ListByStatus(status)
		if err != nil {
			slog.Error("reconciler list agents", "status", status, "error", err)
			continue
		}

		for _, agent := range agents {
			r.reconcileAgent(ctx, agent)
		}
	}

	return nil
}

// reconcileAgent 协调单个 Agent 的状态。
// 根据tmux会话是否存在和是否有检查点，决定 Agent 的新状态。
func (r *Reconciler) reconcileAgent(ctx context.Context, agent *domain.AgentInstance) {
	logger := slog.With("agent_id", agent.ID, "tmux_session", agent.TmuxSession, "current_status", agent.Status)

	// 检查 tmux 会话是否仍然存活
	tmuxAlive := false
	if agent.TmuxSession != "" && r.terminal != nil {
		tmuxAlive = r.terminal.SessionExists(ctx, agent.TmuxSession)
	}

	var newStatus domain.AgentInstanceStatus
	var reason string

	switch {
	case tmuxAlive && agent.Status == domain.AgentStatusPaused:
		// tmux 存活且原状态为暂停，保持暂停
		newStatus = domain.AgentStatusPaused
		reason = "tmux session alive, preserving paused state"
	case tmuxAlive:
		// tmux 存活，标记为运行中
		newStatus = domain.AgentStatusRunning
		reason = "tmux session still alive after restart"
	case agent.CheckpointID != nil && *agent.CheckpointID != "":
		// tmux 不存在但有检查点，标记为可恢复
		newStatus = domain.AgentStatusRecoverable
		reason = "process gone but checkpoint exists"
	default:
		// tmux 不存在且无检查点，标记为失败
		newStatus = domain.AgentStatusFailed
		reason = "process gone and no checkpoint"
	}

	// 仅在状态变化时更新
	if newStatus != agent.Status {
		if err := r.repos.AgentInstances.UpdateStatus(agent.ID, newStatus); err != nil {
			logger.Error("reconciler update status", "error", err)
			return
		}
		logger.Info("reconciler corrected status", "new_status", newStatus, "reason", reason)
	}

	// 记录协调事件
	event := &domain.Event{
		ID:        uuid.New().String(),
		RunID:     agent.RunID,
		TaskID:    &agent.TaskID,
		AgentID:   &agent.ID,
		EventType: domain.EventTypeReconcile,
		Message:   reason,
		Metadata:  fmt.Sprintf(`{"old_status":"%s","new_status":"%s"}`, agent.Status, newStatus),
		CreatedAt: time.Now(),
	}
	if err := r.repos.Events.Create(event); err != nil {
		logger.Error("reconciler create event", "error", err)
	}
}
