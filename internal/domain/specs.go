// Package domain 的 specs 文件定义任务规格、Agent 规格和工作流模板等描述性模型。
// 这些模型用于描述"如何执行"，而非"执行状态"（状态由 models.go 中的实体跟踪）。
package domain

// TaskSpec 描述一类任务的执行规范，包括运行时类型、命令模板、超时和重试策略等。
type TaskSpec struct {
	ID              string        `json:"id" db:"id"`
	Name            string        `json:"name" db:"name"`
	TaskType        string        `json:"task_type" db:"task_type"`
	RuntimeType     string        `json:"runtime_type" db:"runtime_type"`
	CommandTemplate string        `json:"command_template" db:"command_template"`
	TimeoutSeconds  int           `json:"timeout_seconds" db:"timeout_seconds"`
	RetryPolicy     string        `json:"retry_policy" db:"retry_policy"`
	ResourceClass   ResourceClass `json:"resource_class" db:"resource_class"`
	CanPause        bool          `json:"can_pause" db:"can_pause"`
	CanCheckpoint   bool          `json:"can_checkpoint" db:"can_checkpoint"`
	RequiredInputs  string        `json:"required_inputs,omitempty" db:"required_inputs"`
	ExpectedOutputs string        `json:"expected_outputs,omitempty" db:"expected_outputs"`
}

// AgentSpec 描述一类 Agent 的能力和运行参数。
type AgentSpec struct {
	ID                string   `json:"id" db:"id"`
	Name              string   `json:"name" db:"name"`
	AgentKind         string   `json:"agent_kind" db:"agent_kind"`
	SupportedTaskTypes string  `json:"supported_task_types" db:"supported_task_types"`
	DefaultCommand    string   `json:"default_command" db:"default_command"`
	MaxConcurrency    int      `json:"max_concurrency" db:"max_concurrency"`
	ResourceWeight    float64  `json:"resource_weight" db:"resource_weight"`
	HeartbeatMode     string   `json:"heartbeat_mode" db:"heartbeat_mode"`
	OutputParser      string   `json:"output_parser" db:"output_parser"`
}

// WorkflowTemplate 描述一个工作流模板，由节点和边组成的 DAG 定义任务执行顺序。
type WorkflowTemplate struct {
	ID           string                `json:"id" db:"id"`
	Name         string                `json:"name" db:"name"`
	Description  string                `json:"description,omitempty" db:"description"`
	NodesJSON    string                `json:"nodes_json" db:"nodes_json"`
	EdgesJSON    string                `json:"edges_json" db:"edges_json"`
	OnFailure    string                `json:"on_failure" db:"on_failure"`
}

// WorkflowNode 表示工作流中的一个节点，关联一个 TaskSpec。
type WorkflowNode struct {
	ID         string `json:"id"`
	TaskSpecID string `json:"task_spec_id"`
	Label      string `json:"label"`
}

// WorkflowEdge 表示工作流中两个节点之间的依赖关系（有向边）。
type WorkflowEdge struct {
	From string `json:"from"` // 上游节点 ID
	To   string `json:"to"`   // 下游节点 ID
}
