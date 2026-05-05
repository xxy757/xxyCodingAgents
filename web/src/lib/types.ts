// types.ts - 领域模型 TypeScript 接口定义

// Project 项目实体
export interface Project {
  id: string;
  name: string;
  repo_url: string;
  description: string;
  created_at: string;
  updated_at: string;
}

// Run 运行实体
export interface Run {
  id: string;
  project_id: string;
  workflow_template_id: string;
  title: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// Task 任务实体
export interface Task {
  id: string;
  run_id: string;
  task_type: string;
  attempt_no: number;
  status: string;
  priority: string;
  queue_status: string;
  resource_class: string;
  title: string;
  description: string;
  input_data?: string;
  output_data?: string;
  created_at: string;
  updated_at: string;
}

// AgentInstance Agent 实例
export interface AgentInstance {
  id: string;
  run_id: string;
  task_id: string;
  agent_kind: string;
  status: string;
  pid: number | null;
  tmux_session: string;
  created_at: string;
  updated_at: string;
}

// Event 系统事件
export interface Event {
  id: string;
  run_id: string;
  task_id: string | null;
  agent_id: string | null;
  event_type: string;
  message: string;
  created_at: string;
}

// ResourceSnapshot 系统资源快照
export interface ResourceSnapshot {
  id: string;
  memory_percent: number;
  cpu_percent: number;
  disk_percent: number;
  active_agents: number;
  pressure_level: string;
  created_at: string;
}

// TerminalSession 终端会话
export interface TerminalSession {
  id: string;
  task_id: string;
  agent_id?: string;
  tmux_session: string;
  tmux_pane: string;
  status: string;
  log_file_path: string;
  created_at: string;
}

// WorkflowTemplate 工作流模板
export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  nodes_json: string;
  edges_json: string;
  on_failure: string;
}

// PromptDraft 提示词草稿
export interface PromptDraft {
  id: string;
  project_id: string;
  original_input: string;
  generated_prompt: string;
  final_prompt: string;
  task_type: string;
  status: string;
  run_id?: string;
  sent_at?: string;
  created_at: string;
  updated_at: string;
}

// Gate 质量门禁实体
export interface Gate {
  id: string;
  run_id: string;
  node_id: string;
  gate_type: string;
  status: string;
  config_json: string;
  verify_result: string;
  approved_by: string;
  approved_at: string | null;
  created_at: string;
  updated_at: string;
}

// TechStackOption 技术方案预设
export interface TechStackOption {
  id: string;
  label: string;
  context: string;
}

// HealthStatus 健康检查响应
export interface HealthStatus {
  status: string;
}

// WorkflowGraphData 工作流图数据（ReactFlow 兼容）
export interface WorkflowGraphData {
  nodes: WorkflowNodeData[];
  edges: WorkflowEdgeData[];
}

export interface WorkflowNodeData {
  id: string;
  type: string;
  data: {
    label: string;
    status: string;
    task_type: string;
    task_id: string;
    gate_id?: string;
    gate_type?: string;
  };
  position: { x: number; y: number };
}

export interface WorkflowEdgeData {
  id: string;
  source: string;
  target: string;
}

// SendPromptDraftResult 发送草稿结果
export interface SendPromptDraftResult {
  draft_id: string;
  run_id: string;
  task_id: string;
  status: string;
  warnings?: string[];
}

// DiagnosticsData 系统诊断数据
export interface DiagnosticsData {
  snapshot: ResourceSnapshot;
  tmux_sessions: string;
  active_agents: string[];
  config: {
    max_concurrent_agents: number;
    max_heavy_agents: number;
  };
}
