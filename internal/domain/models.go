// Package domain 定义了 AI Dev Platform 的核心领域模型。
// 包含项目、运行、任务、Agent 实例、终端会话、检查点、资源快照、事件等实体，
// 以及各种状态枚举和优先级定义。
package domain

import "time"

// Project 表示一个被管理的代码项目。
type Project struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	RepoURL     string    `json:"repo_url" db:"repo_url"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RunStatus 表示运行（Run）的生命周期状态。
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"   // 等待启动
	RunStatusRunning   RunStatus = "running"   // 正在执行
	RunStatusCompleted RunStatus = "completed" // 已成功完成
	RunStatusFailed    RunStatus = "failed"    // 执行失败
	RunStatusCancelled RunStatus = "cancelled" // 已被取消
)

// Run 表示一次工作流执行实例，包含多个任务。
type Run struct {
	ID                 string    `json:"id" db:"id"`
	ProjectID          string    `json:"project_id" db:"project_id"`
	WorkflowTemplateID string    `json:"workflow_template_id" db:"workflow_template_id"`
	Title              string    `json:"title" db:"title"`
	Description        string    `json:"description" db:"description"`
	Status             RunStatus `json:"status" db:"status"`
	ExternalKey        string    `json:"external_key,omitempty" db:"external_key"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// TaskStatus 表示任务（Task）的生命周期状态。
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"    // 已入队，等待调度
	TaskStatusAdmitted  TaskStatus = "admitted"  // 已接纳，准备启动 Agent
	TaskStatusRunning   TaskStatus = "running"   // 正在由 Agent 执行
	TaskStatusCompleted TaskStatus = "completed" // 执行完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
	TaskStatusEvicted   TaskStatus = "evicted"   // 因资源压力被驱逐
	TaskStatusBlocked   TaskStatus = "blocked"   // 被依赖阻塞，等待前置任务完成
)

// TaskPriority 表示任务的调度优先级。
type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"    // 低优先级，可被抢占
	PriorityNormal TaskPriority = "normal" // 普通优先级
	PriorityHigh   TaskPriority = "high"   // 高优先级
)

// ResourceClass 表示任务的资源消耗等级。
type ResourceClass string

const (
	ResourceClassLight  ResourceClass = "light"  // 轻量级，占用资源少
	ResourceClassMedium ResourceClass = "medium" // 中等资源消耗
	ResourceClassHeavy  ResourceClass = "heavy"  // 重型，占用大量资源
)

// Task 表示一个待执行的任务单元，是调度的最小单位。
type Task struct {
	ID              string        `json:"id" db:"id"`
	RunID           string        `json:"run_id" db:"run_id"`
	TaskSpecID      string        `json:"task_spec_id" db:"task_spec_id"`
	TaskType        string        `json:"task_type" db:"task_type"`
	AttemptNo       int           `json:"attempt_no" db:"attempt_no"`
	Status          TaskStatus    `json:"status" db:"status"`
	Priority        TaskPriority  `json:"priority" db:"priority"`
	QueueStatus     string        `json:"queue_status" db:"queue_status"`
	ResourceClass   ResourceClass `json:"resource_class" db:"resource_class"`
	Preemptible     bool          `json:"preemptible" db:"preemptible"`
	RestartPolicy   string        `json:"restart_policy" db:"restart_policy"`
	Title           string        `json:"title" db:"title"`
	Description     string        `json:"description" db:"description"`
	InputData       string        `json:"input_data,omitempty" db:"input_data"`
	OutputData      string        `json:"output_data,omitempty" db:"output_data"`
	WorkspacePath   string        `json:"workspace_path,omitempty" db:"workspace_path"`
	ParentTaskID    *string       `json:"parent_task_id,omitempty" db:"parent_task_id"`
	StartedAt       *time.Time    `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// AgentInstanceStatus 表示 Agent 实例的生命周期状态。
type AgentInstanceStatus string

const (
	AgentStatusStarting    AgentInstanceStatus = "starting"    // 正在启动
	AgentStatusRunning     AgentInstanceStatus = "running"     // 正常运行中
	AgentStatusPaused      AgentInstanceStatus = "paused"      // 已暂停
	AgentStatusStopped     AgentInstanceStatus = "stopped"     // 已停止
	AgentStatusFailed      AgentInstanceStatus = "failed"      // 执行失败
	AgentStatusRecoverable AgentInstanceStatus = "recoverable" // 可从检查点恢复
	AgentStatusOrphaned    AgentInstanceStatus = "orphaned"    // 孤立状态（未找到对应进程）
)

// AgentInstance 表示一个正在运行或曾经运行的 Agent 实例。
type AgentInstance struct {
	ID              string              `json:"id" db:"id"`
	RunID           string              `json:"run_id" db:"run_id"`
	TaskID          string              `json:"task_id" db:"task_id"`
	AgentSpecID     string              `json:"agent_spec_id" db:"agent_spec_id"`
	AgentKind       string              `json:"agent_kind" db:"agent_kind"`
	Status          AgentInstanceStatus  `json:"status" db:"status"`
	PID             *int                `json:"pid,omitempty" db:"pid"`
	TmuxSession     string              `json:"tmux_session,omitempty" db:"tmux_session"`
	WorkspacePath   string              `json:"workspace_path,omitempty" db:"workspace_path"`
	LastHeartbeatAt *time.Time          `json:"last_heartbeat_at,omitempty" db:"last_heartbeat_at"`
	LastOutputAt    *time.Time          `json:"last_output_at,omitempty" db:"last_output_at"`
	CheckpointID    *string             `json:"checkpoint_id,omitempty" db:"checkpoint_id"`
	Metadata        string              `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at" db:"updated_at"`
}

// Workspace 表示 Agent 使用的文件系统工作区。
type Workspace struct {
	ID        string    `json:"id" db:"id"`
	TaskID    string    `json:"task_id" db:"task_id"`
	ProjectID string    `json:"project_id" db:"project_id"`
	Path      string    `json:"path" db:"path"`
	Branch    string    `json:"branch,omitempty" db:"branch"`
	CommitSHA string    `json:"commit_sha,omitempty" db:"commit_sha"`
	SizeBytes int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TerminalSessionStatus 表示终端会话的状态。
type TerminalSessionStatus string

const (
	TerminalStatusActive   TerminalSessionStatus = "active"   // 活跃中
	TerminalStatusDetached TerminalSessionStatus = "detached"` // 已分离
	TerminalStatusClosed   TerminalSessionStatus = "closed"    // 已关闭
)

// TerminalSession 表示一个 tmux 终端会话，可通过 WebSocket 交互。
type TerminalSession struct {
	ID            string                `json:"id" db:"id"`
	TaskID        string                `json:"task_id" db:"task_id"`
	AgentID       *string               `json:"agent_id,omitempty" db:"agent_id"`
	TmuxSession   string                `json:"tmux_session" db:"tmux_session"`
	TmuxPane      string                `json:"tmux_pane,omitempty" db:"tmux_pane"`
	Status        TerminalSessionStatus `json:"status" db:"status"`
	LogFilePath   string                `json:"log_file_path,omitempty" db:"log_file_path"`
	CreatedAt     time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at" db:"updated_at"`
}

// Checkpoint 表示 Agent 执行状态的快照，用于故障恢复。
type Checkpoint struct {
	ID         string    `json:"id" db:"id"`
	AgentID    string    `json:"agent_id" db:"agent_id"`
	TaskID     string    `json:"task_id" db:"task_id"`
	RunID      string    `json:"run_id" db:"run_id"`
	Phase      string    `json:"phase" db:"phase"`
	StateData  string    `json:"state_data" db:"state_data"`
	Reason     string    `json:"reason" db:"reason"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ResourceSnapshot 记录某一时刻的系统资源使用情况。
type ResourceSnapshot struct {
	ID            string    `json:"id" db:"id"`
	MemoryPercent float64   `json:"memory_percent" db:"memory_percent"`
	CPUPercent    float64   `json:"cpu_percent" db:"cpu_percent"`
	DiskPercent   float64   `json:"disk_percent" db:"disk_percent"`
	ActiveAgents  int       `json:"active_agents" db:"active_agents"`
	PressureLevel string    `json:"pressure_level" db:"pressure_level"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// EventType 表示系统事件的类型。
type EventType string

const (
	EventTypeTaskStarted    EventType = "task_started"     // 任务已启动
	EventTypeTaskCompleted  EventType = "task_completed"   // 任务已完成
	EventTypeTaskFailed     EventType = "task_failed"      // 任务失败
	EventTypeTaskCancelled  EventType = "task_cancelled"   // 任务已取消
	EventTypeTaskEvicted    EventType = "task_evicted"     // 任务被驱逐
	EventTypeAgentStarted   EventType = "agent_started"    // Agent 已启动
	EventTypeAgentPaused    EventType = "agent_paused"     // Agent 已暂停
	EventTypeAgentResumed   EventType = "agent_resumed"    // Agent 已恢复
	EventTypeAgentStopped   EventType = "agent_stopped"    // Agent 已停止
	EventTypeAgentFailed    EventType = "agent_failed"     // Agent 失败
	EventTypeAgentHeartbeat EventType = "agent_heartbeat"  // Agent 心跳
	EventTypeCheckpoint     EventType = "checkpoint_created"` // 检查点已创建
	EventTypePressureChange EventType = "pressure_change"  // 资源压力变化
	EventTypeReconcile      EventType = "reconcile"        // 协调事件
)

// Event 记录系统中发生的各类事件，用于审计和时间线展示。
type Event struct {
	ID        string    `json:"id" db:"id"`
	RunID     string    `json:"run_id" db:"run_id"`
	TaskID    *string   `json:"task_id,omitempty" db:"task_id"`
	AgentID   *string   `json:"agent_id,omitempty" db:"agent_id"`
	EventType EventType `json:"event_type" db:"event_type"`
	Message   string    `json:"message" db:"message"`
	Metadata  string    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CommandLog 记录 Agent 执行的命令及其输出，用于审计追踪。
type CommandLog struct {
	ID        string    `json:"id" db:"id"`
	TaskID    string    `json:"task_id" db:"task_id"`
	AgentID   *string   `json:"agent_id,omitempty" db:"agent_id"`
	Command   string    `json:"command" db:"command"`
	ExitCode  *int      `json:"exit_code,omitempty" db:"exit_code"`
	Output    string    `json:"output,omitempty" db:"output"`
	Duration  int64     `json:"duration_ms" db:"duration_ms"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
