package domain

import "time"

type Project struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	RepoURL     string    `json:"repo_url" db:"repo_url"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

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

type TaskStatus string

const (
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusAdmitted   TaskStatus = "admitted"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusEvicted    TaskStatus = "evicted"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityNormal TaskPriority = "normal"
	PriorityHigh   TaskPriority = "high"
)

type ResourceClass string

const (
	ResourceClassLight  ResourceClass = "light"
	ResourceClassMedium ResourceClass = "medium"
	ResourceClassHeavy  ResourceClass = "heavy"
)

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

type AgentInstanceStatus string

const (
	AgentStatusStarting    AgentInstanceStatus = "starting"
	AgentStatusRunning     AgentInstanceStatus = "running"
	AgentStatusPaused      AgentInstanceStatus = "paused"
	AgentStatusStopped     AgentInstanceStatus = "stopped"
	AgentStatusFailed      AgentInstanceStatus = "failed"
	AgentStatusRecoverable AgentInstanceStatus = "recoverable"
	AgentStatusOrphaned    AgentInstanceStatus = "orphaned"
)

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

type TerminalSessionStatus string

const (
	TerminalStatusActive   TerminalSessionStatus = "active"
	TerminalStatusDetached TerminalSessionStatus = "detached"
	TerminalStatusClosed   TerminalSessionStatus = "closed"
)

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

type ResourceSnapshot struct {
	ID            string    `json:"id" db:"id"`
	MemoryPercent float64   `json:"memory_percent" db:"memory_percent"`
	CPUPercent    float64   `json:"cpu_percent" db:"cpu_percent"`
	DiskPercent   float64   `json:"disk_percent" db:"disk_percent"`
	ActiveAgents  int       `json:"active_agents" db:"active_agents"`
	PressureLevel string    `json:"pressure_level" db:"pressure_level"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type EventType string

const (
	EventTypeTaskStarted    EventType = "task_started"
	EventTypeTaskCompleted  EventType = "task_completed"
	EventTypeTaskFailed     EventType = "task_failed"
	EventTypeTaskCancelled  EventType = "task_cancelled"
	EventTypeTaskEvicted    EventType = "task_evicted"
	EventTypeAgentStarted   EventType = "agent_started"
	EventTypeAgentPaused    EventType = "agent_paused"
	EventTypeAgentResumed   EventType = "agent_resumed"
	EventTypeAgentStopped   EventType = "agent_stopped"
	EventTypeAgentFailed    EventType = "agent_failed"
	EventTypeAgentHeartbeat EventType = "agent_heartbeat"
	EventTypeCheckpoint     EventType = "checkpoint_created"
	EventTypePressureChange EventType = "pressure_change"
	EventTypeReconcile      EventType = "reconcile"
)

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
