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

// ==================== 状态颜色映射 ====================

// statusColors 将任务/Agent 状态映射到 Tailwind CSS 类名（使用语义色 token）
export const statusColors: Record<string, string> = {
  pending: 'bg-warning-50 text-warning-600',
  queued: 'bg-neutral-100 text-neutral-600',
  admitted: 'bg-primary-50 text-primary-700',
  running: 'bg-success-50 text-success-700',
  completed: 'bg-success-50 text-success-700',
  failed: 'bg-error-50 text-error-700',
  cancelled: 'bg-neutral-200 text-neutral-600',
  evicted: 'bg-warning-50 text-warning-600',
  paused: 'bg-warning-50 text-warning-600',
  stopped: 'bg-neutral-200 text-neutral-600',
  starting: 'bg-primary-50 text-primary-700',
  recoverable: 'bg-info-50 text-info-600',
  orphaned: 'bg-error-50 text-error-700',
  active: 'bg-success-50 text-success-700',
  detached: 'bg-warning-50 text-warning-600',
  closed: 'bg-neutral-200 text-neutral-600',
};

// pressureColors 将资源压力等级映射到 Tailwind 文字颜色类名
export const pressureColors: Record<string, string> = {
  normal: 'text-success-600',
  warn: 'text-warning-600',
  high: 'text-warning-600',
  critical: 'text-error-600',
};

// draftStatusColors 将草稿状态映射到 Tailwind CSS 类名
export const draftStatusColors: Record<string, string> = {
  draft: 'bg-warning-50 text-warning-600',
  sent: 'bg-success-50 text-success-700',
};

// statusHex 将任务状态映射到十六进制颜色值（用于 ReactFlow 等需要 inline style 的场景）
export const statusHex: Record<string, { border: string; bg: string }> = {
  running:   { border: "#22c55e", bg: "#f0fdf4" },
  completed: { border: "#3b82f6", bg: "#eff6ff" },
  failed:    { border: "#ef4444", bg: "#fef2f2" },
  queued:    { border: "#eab308", bg: "#fefce8" },
  blocked:   { border: "#a855f7", bg: "#faf5ff" },
  evicted:   { border: "#f97316", bg: "#fff7ed" },
  pending:   { border: "#eab308", bg: "#fefce8" },
  cancelled: { border: "#9ca3af", bg: "#f9fafb" },
  stopped:   { border: "#9ca3af", bg: "#f9fafb" },
  default:   { border: "#9ca3af", bg: "#f9fafb" },
};

// ==================== 提示词草稿 API ====================

// generatePromptDraft 生成提示词草稿
export function generatePromptDraft(
  projectId: string,
  originalInput: string,
  taskType?: string,
): Promise<PromptDraft> {
  return apiFetch<PromptDraft>('/api/prompt-drafts/generate', {
    method: 'POST',
    body: JSON.stringify({ project_id: projectId, original_input: originalInput, task_type: taskType }),
  });
}

// updatePromptDraft 更新提示词草稿的 final_prompt
export function updatePromptDraft(
  id: string,
  finalPrompt: string,
  taskType?: string,
): Promise<PromptDraft> {
  return apiFetch<PromptDraft>(`/api/prompt-drafts/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ final_prompt: finalPrompt, task_type: taskType }),
  });
}

// sendPromptDraft 确认并发送提示词草稿，创建 Run/Task
export function sendPromptDraft(
  id: string,
): Promise<{ draft_id: string; run_id: string; task_id: string; status: string }> {
  return apiFetch(`/api/prompt-drafts/${id}/send`, {
    method: 'POST',
  });
}

// listPromptDrafts 列出指定项目的提示词草稿
export function listPromptDrafts(projectId: string): Promise<PromptDraft[]> {
  return apiFetch<PromptDraft[]>(`/api/prompt-drafts?project_id=${encodeURIComponent(projectId)}`);
}

// ==================== 质量门禁 ====================

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

// gateStatusColors 将门禁状态映射到语义色 Tailwind 类名
export const gateStatusColors: Record<string, string> = {
  pending: 'bg-warning-50 text-warning-600',
  passed: 'bg-success-50 text-success-700',
  failed: 'bg-error-50 text-error-700',
  skipped: 'bg-neutral-200 text-neutral-600',
};

// approveGate 通过一个门禁
export function approveGate(id: string, approvedBy?: string): Promise<Gate> {
  return apiFetch<Gate>(`/api/gates/${id}/approve`, {
    method: 'POST',
    body: JSON.stringify({ approved_by: approvedBy || 'user' }),
  });
}

// listGates 列出指定运行的所有门禁
export function listGates(runId: string): Promise<Gate[]> {
  return apiFetch<Gate[]>(`/api/gates?run_id=${encodeURIComponent(runId)}`);
}

// getGate 获取单个门禁详情
export function getGate(id: string): Promise<Gate> {
  return apiFetch<Gate>(`/api/gates/${id}`);
}
