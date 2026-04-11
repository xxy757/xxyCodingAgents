package domain

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

type WorkflowTemplate struct {
	ID           string                `json:"id" db:"id"`
	Name         string                `json:"name" db:"name"`
	Description  string                `json:"description,omitempty" db:"description"`
	NodesJSON    string                `json:"nodes_json" db:"nodes_json"`
	EdgesJSON    string                `json:"edges_json" db:"edges_json"`
	OnFailure    string                `json:"on_failure" db:"on_failure"`
}

type WorkflowNode struct {
	ID         string `json:"id"`
	TaskSpecID string `json:"task_spec_id"`
	Label      string `json:"label"`
}

type WorkflowEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
