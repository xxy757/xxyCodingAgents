// Package runtime 定义 Agent 运行时的抽象接口和数据结构。
// 不同的 Agent 类型（如 Claude Code、通用 Shell）通过实现 AgentRuntime 接口
// 来提供启动、暂停、恢复、停止、检查点和检查等操作。
package runtime

import (
	"context"
)

// StartRequest 包含启动一个 Agent 所需的全部参数。
type StartRequest struct {
	AgentID        string            `json:"agent_id"`
	TaskID         string            `json:"task_id"`
	RunID          string            `json:"run_id"`
	AgentKind      string            `json:"agent_kind"`
	Command        string            `json:"command"`
	TmuxSession    string            `json:"tmux_session"`
	WorkspacePath  string            `json:"workspace_path"`
	Env            map[string]string `json:"env"`
}

// StartResult 包含 Agent 启动后返回的信息。
type StartResult struct {
	PID         int    `json:"pid"`
	TmuxSession string `json:"tmux_session"`
	TmuxPane    string `json:"tmux_pane"`
}

// CheckpointData 包含 Agent 执行状态的快照数据。
type CheckpointData struct {
	Phase     string `json:"phase"`      // 当前执行阶段
	StateData string `json:"state_data"` // 序列化的状态数据
}

// AgentStatus 包含 Agent 的当前运行状态。
type AgentStatus struct {
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	Phase     string `json:"phase"`
	OutputLen int    `json:"output_len"`
}

// AgentRuntime 是 Agent 运行时的抽象接口，定义了所有运行时操作。
type AgentRuntime interface {
	// Start 在指定 tmux 会话中启动 Agent 执行命令
	Start(ctx context.Context, req StartRequest) (*StartResult, error)
	// Pause 暂停指定 tmux 会话中的 Agent（发送 SIGTSTP）
	Pause(ctx context.Context, agentID string) error
	// Resume 恢复已暂停的 Agent（发送 fg 命令）
	Resume(ctx context.Context, agentID string) error
	// Stop 停止 Agent（终止 tmux 会话）
	Stop(ctx context.Context, agentID string) error
	// Checkpoint 创建 Agent 当前状态的检查点
	Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error)
	// Inspect 检查 Agent 的运行状态
	Inspect(ctx context.Context, agentID string) (*AgentStatus, error)
}
