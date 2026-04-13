package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

type Watchdog struct {
	cfg      *config.Config
	repos    *storage.Repos
	runtime  agentruntime.AgentRuntime
	terminal *terminal.Manager
	stop     chan struct{}
}

func NewWatchdog(cfg *config.Config, repos *storage.Repos, rt agentruntime.AgentRuntime, tm *terminal.Manager) *Watchdog {
	return &Watchdog{
		cfg:      cfg,
		repos:    repos,
		runtime:  rt,
		terminal: tm,
		stop:     make(chan struct{}),
	}
}

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

func (w *Watchdog) Stop() {
	close(w.stop)
}

func (w *Watchdog) check(ctx context.Context) {
	agents, err := w.repos.AgentInstances.ListActiveWithTasks()
	if err != nil {
		slog.Error("watchdog: list active agents", "error", err)
		return
	}

	now := time.Now()
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

		if agent.Status != domain.AgentStatusRunning && agent.Status != domain.AgentStatusStarting {
			continue
		}

		inspectResult, err := w.runtime.Inspect(ctx, agent.TmuxSession)
		if err != nil {
			slog.Warn("watchdog: inspect agent", "agent_id", agent.ID, "error", err)
			continue
		}

		if !inspectResult.Running {
			w.handleDeadAgent(ctx, agent, task, "process_crashed")
			continue
		}

		w.repos.AgentInstances.UpdateHeartbeat(agent.ID)

		if agent.LastHeartbeatAt != nil {
			elapsed := now.Sub(*agent.LastHeartbeatAt)
			if elapsed > heartbeatTimeout {
				w.handleDeadAgent(ctx, agent, task, "heartbeat_timeout")
				continue
			}
		}

		if agent.LastOutputAt != nil {
			elapsed := now.Sub(*agent.LastOutputAt)
			if elapsed > outputTimeout {
				w.handleDeadAgent(ctx, agent, task, "output_timeout")
				continue
			}
		}
	}
}

func (w *Watchdog) handleDeadAgent(ctx context.Context, agent *domain.AgentInstance, task *domain.Task, reason string) {
	slog.Warn("watchdog: agent dead", "agent_id", agent.ID, "reason", reason, "tmux_session", agent.TmuxSession)

	newStatus := domain.AgentStatusFailed
	if task.RestartPolicy == "always" || task.RestartPolicy == "on-failure" {
		newStatus = domain.AgentStatusRecoverable
	}

	w.repos.AgentInstances.UpdateStatus(agent.ID, newStatus)
	w.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)

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
}
