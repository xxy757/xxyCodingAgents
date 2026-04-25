// api.ts - API 客户端工具库
// 封装与后端 API 的通信，定义所有领域模型的 TypeScript 接口和通用状态映射。

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

// apiFetch 是统一的 API 请求函数，自动添加 JSON 头并处理错误响应
export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `API error: ${res.status}`);
  }
  return res.json();
}

// ==================== 领域模型接口 ====================

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
  tmux_session: string;
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

// ==================== 状态颜色映射 ====================

// statusColors 将任务/Agent 状态映射到 Tailwind CSS 类名
export const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  queued: 'bg-gray-100 text-gray-800',
  admitted: 'bg-blue-100 text-blue-800',
  running: 'bg-green-100 text-green-800',
  completed: 'bg-emerald-100 text-emerald-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-200 text-gray-600',
  evicted: 'bg-orange-100 text-orange-800',
  paused: 'bg-yellow-100 text-yellow-800',
  stopped: 'bg-gray-200 text-gray-600',
  starting: 'bg-blue-100 text-blue-800',
  recoverable: 'bg-purple-100 text-purple-800',
  orphaned: 'bg-red-100 text-red-800',
  active: 'bg-green-100 text-green-800',
  detached: 'bg-yellow-100 text-yellow-800',
  closed: 'bg-gray-200 text-gray-600',
};

// pressureColors 将资源压力等级映射到 Tailwind 文字颜色类名
export const pressureColors: Record<string, string> = {
  normal: 'text-green-600',
  warn: 'text-yellow-600',
  high: 'text-orange-600',
  critical: 'text-red-600',
};
