// Package runtime 的 adapters 文件提供 AgentRuntime 接口的具体实现。
// 包含 ClaudeCodeAdapter（Claude Code 专用）和 GenericShellAdapter（通用 Shell）两种适配器，
// 都通过 tmux 会话管理 Agent 进程的生命周期。
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// ClaudeCodeAdapter 实现 AgentRuntime 接口，针对 Claude Code 类型 Agent。
// 通过 tmux 会话发送命令并捕获输出。
type ClaudeCodeAdapter struct{}

// NewClaudeCodeAdapter 创建 Claude Code 适配器实例。
func NewClaudeCodeAdapter() *ClaudeCodeAdapter {
	return &ClaudeCodeAdapter{}
}

// Start 在指定 tmux 会话中发送启动命令。
func (a *ClaudeCodeAdapter) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if req.TmuxSession == "" {
		return nil, fmt.Errorf("tmux session is required")
	}

	// 向 tmux 会话发送启动命令
	args := []string{"send-keys", "-t", req.TmuxSession, req.Command, "Enter"}
	cmd := exec.CommandContext(ctx, "tmux", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("send command to tmux: %w, output: %s", err, string(out))
	}

	// 获取 tmux 会话的进程 ID
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

// Pause 暂停 Agent，发送 Ctrl+Z 信号到 tmux 会话。
func (a *ClaudeCodeAdapter) Pause(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "C-z").Run()
}

// Resume 恢复已暂停的 Agent，发送 fg 命令到 tmux 会话。
func (a *ClaudeCodeAdapter) Resume(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "fg", "Enter").Run()
}

// Stop 停止 Agent，终止整个 tmux 会话。
func (a *ClaudeCodeAdapter) Stop(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "kill-session", "-t", tmuxSession).Run()
}

// Checkpoint 创建 Agent 状态快照，捕获 tmux 终端输出和运行状态。
func (a *ClaudeCodeAdapter) Checkpoint(ctx context.Context, tmuxSession string) (*CheckpointData, error) {
	// 捕获 tmux 终端最近 100 行输出
	output := ""
	if tmuxSession != "" {
		out, err := exec.CommandContext(ctx, "tmux", "capture-pane", "-t", tmuxSession, "-p", "-S", "-100").Output()
		if err == nil {
			output = string(out)
		}
	}
	// 检查 Agent 当前运行状态
	status := AgentStatus{Running: false}
	inspectResult, err := a.Inspect(ctx, tmuxSession)
	if err == nil {
		status = *inspectResult
	}
	stateData := fmt.Sprintf(`{"tmux_output_length":%d,"running":%v}`, len(output), status.Running)
	return &CheckpointData{
		Phase:     "running",
		StateData: stateData,
	}, nil
}

// Inspect 检查 Agent 是否仍在运行（通过检查 tmux 会话是否存在）。
func (a *ClaudeCodeAdapter) Inspect(ctx context.Context, tmuxSession string) (*AgentStatus, error) {
	if tmuxSession == "" {
		return &AgentStatus{Running: false}, nil
	}
	err := exec.CommandContext(ctx, "tmux", "has-session", "-t", tmuxSession).Run()
	return &AgentStatus{Running: err == nil}, nil
}

// getTmuxSessionPID 获取 tmux 会话中主进程的 PID。
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

// GenericShellAdapter 实现 AgentRuntime 接口，适用于通用 Shell 命令。
type GenericShellAdapter struct{}

// NewGenericShellAdapter 创建通用 Shell 适配器实例。
func NewGenericShellAdapter() *GenericShellAdapter {
	return &GenericShellAdapter{}
}

// Start 在指定 tmux 会话中发送命令。
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

// Pause 暂停 Agent，发送 Ctrl+Z 信号。
func (a *GenericShellAdapter) Pause(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "C-z").Run()
}

// Resume 恢复已暂停的 Agent，发送 fg 命令。
func (a *GenericShellAdapter) Resume(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "send-keys", "-t", tmuxSession, "fg", "Enter").Run()
}

// Stop 终止 tmux 会话。
func (a *GenericShellAdapter) Stop(ctx context.Context, tmuxSession string) error {
	if tmuxSession == "" {
		return nil
	}
	return exec.CommandContext(ctx, "tmux", "kill-session", "-t", tmuxSession).Run()
}

// Checkpoint 返回空检查点（通用 Shell 不支持真实检查点）。
func (a *GenericShellAdapter) Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error) {
	return &CheckpointData{Phase: "unknown", StateData: "{}"}, nil
}

// Inspect 检查 tmux 会话是否存在。
func (a *GenericShellAdapter) Inspect(ctx context.Context, tmuxSession string) (*AgentStatus, error) {
	if tmuxSession == "" {
		return &AgentStatus{Running: false}, nil
	}
	err := exec.CommandContext(ctx, "tmux", "has-session", "-t", tmuxSession).Run()
	return &AgentStatus{Running: err == nil}, nil
}
