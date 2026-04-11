package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

type ClaudeCodeAdapter struct{}

func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	return &ClaudeCodeAdapter{}
}

func (a *ClaudeCodeAdapter) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if req.TmuxSession == "" {
		return nil, fmt.Errorf("tmux session is required")
	}

	args := []string{"send-keys", "-t", req.TmuxSession, req.Command, "Enter"}
	cmd := exec.CommandContext(ctx, "tmux", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("send command to tmux: %w, output: %s", err, string(out))
	}

	pid, err := a.getTmuxSessionPID(req.TmuxSession)
	if err != nil {
		slog.Warn("get tmux session pid", "session", req.TmuxSession, "error", err)
	}

	slog.Info("agent started", "agent_id", req.AgentID, "tmux_session", req.TmuxSession, "pid", pid)
	return &StartResult{
		PID:         pid,
		TmuxSession: req.TmuxSession,
	}, nil
}

func (a *ClaudeCodeAdapter) Pause(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "C-z").Run()
}

func (a *ClaudeCodeAdapter) Resume(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "fg", "Enter").Run()
}

func (a *ClaudeCodeAdapter) Stop(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "kill-session", "-t", tmuxSession).Run()
}

func (a *ClaudeCodeAdapter) Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error) {
	return &CheckpointData{
		Phase:     "running",
		StateData: "{}",
	}, nil
}

func (a *ClaudeCodeAdapter) Inspect(ctx context.Context, tmuxSession string) (*AgentStatus, error) {
	if tmuxSession == "" {
		return &AgentStatus{Running: false}, nil
	}
	err := exec.CommandContext(ctx, "tmux", "has-session", "-t", tmuxSession).Run()
	return &AgentStatus{Running: err == nil}, nil
}

func (a *ClaudeCodeAdapter) getTmuxSessionPID(session string) (int, error) {
	out, err := exec.Command("tmux", "list-panes", "-t", session, "-F", "#{pane_pid}").Output()
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("parse pid %q: %w", pidStr, err)
	}
	return pid, nil
}

type GenericShellAdapter struct{}

func NewGenericShellAdapter() *GenericShellAdapter {
	return &GenericShellAdapter{}
}

func (a *GenericShellAdapter) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	args := []string{"send-keys", "-t", req.TmuxSession, req.Command, "Enter"}
	cmd := exec.CommandContext(ctx, "tmux", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("send command to tmux: %w, output: %s", err, string(out))
	}
	return &StartResult{
		TmuxSession: req.TmuxSession,
	}, nil
}

func (a *GenericShellAdapter) Pause(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "C-z").Run()
}

func (a *GenericShellAdapter) Resume(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "fg", "Enter").Run()
}

func (a *GenericShellAdapter) Stop(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "kill-session", "-t", tmuxSession).Run()
}

func (a *GenericShellAdapter) Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error) {
	return &CheckpointData{Phase: "unknown", StateData: "{}"}, nil
}

func (a *GenericShellAdapter) Inspect(ctx context.Context, tmuxSession string) (*AgentStatus, error) {
	if tmuxSession == "" {
		return &AgentStatus{Running: false}, nil
	}
	err := exec.CommandContext(ctx, "tmux", "has-session", "-t", tmuxSession).Run()
	return &AgentStatus{Running: err == nil}, nil
}
