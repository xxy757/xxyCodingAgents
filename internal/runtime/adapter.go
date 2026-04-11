package runtime

import (
	"context"
)

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

type StartResult struct {
	PID         int    `json:"pid"`
	TmuxSession string `json:"tmux_session"`
	TmuxPane    string `json:"tmux_pane"`
}

type CheckpointData struct {
	Phase     string `json:"phase"`
	StateData string `json:"state_data"`
}

type AgentStatus struct {
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	Phase     string `json:"phase"`
	OutputLen int    `json:"output_len"`
}

type AgentRuntime interface {
	Start(ctx context.Context, req StartRequest) (*StartResult, error)
	Pause(ctx context.Context, agentID string) error
	Resume(ctx context.Context, agentID string) error
	Stop(ctx context.Context, agentID string) error
	Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error)
	Inspect(ctx context.Context, agentID string) (*AgentStatus, error)
}
