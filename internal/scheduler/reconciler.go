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

type Reconciler struct {
	repos    *storage.Repos
	terminal TerminalChecker
}

type TerminalChecker interface {
	SessionExists(ctx context.Context, name string) bool
}

func NewReconciler(repos *storage.Repos, terminal TerminalChecker) *Reconciler {
	return &Reconciler{
		repos:    repos,
		terminal: terminal,
	}
}

func (r *Reconciler) Run(ctx context.Context) error {
	slog.Info("startup reconciler starting")
	defer slog.Info("startup reconciler completed")

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

func (r *Reconciler) reconcileAgent(ctx context.Context, agent *domain.AgentInstance) {
	logger := slog.With("agent_id", agent.ID, "tmux_session", agent.TmuxSession, "current_status", agent.Status)

	tmuxAlive := false
	if agent.TmuxSession != "" && r.terminal != nil {
		tmuxAlive = r.terminal.SessionExists(ctx, agent.TmuxSession)
	}

	var newStatus domain.AgentInstanceStatus
	var reason string

	switch {
	case tmuxAlive && agent.Status == domain.AgentStatusPaused:
		newStatus = domain.AgentStatusPaused
		reason = "tmux session alive, preserving paused state"
	case tmuxAlive:
		newStatus = domain.AgentStatusRunning
		reason = "tmux session still alive after restart"
	case agent.CheckpointID != nil && *agent.CheckpointID != "":
		newStatus = domain.AgentStatusRecoverable
		reason = "process gone but checkpoint exists"
	default:
		newStatus = domain.AgentStatusFailed
		reason = "process gone and no checkpoint"
	}

	if newStatus != agent.Status {
		if err := r.repos.AgentInstances.UpdateStatus(agent.ID, newStatus); err != nil {
			logger.Error("reconciler update status", "error", err)
			return
		}
		logger.Info("reconciler corrected status", "new_status", newStatus, "reason", reason)
	}

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
