// api.ts - API 客户端，按资源分组的 API 函数

import type {
  Project, Run, Task, AgentInstance, Event, ResourceSnapshot,
  TerminalSession, WorkflowTemplate, PromptDraft, Gate,
  HealthStatus, DiagnosticsData, WorkflowGraphData, SendPromptDraftResult,
  TechStackOption,
} from './types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

/** 统一的 API 请求函数，自动添加 JSON 头并处理错误响应 */
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

// ==================== 项目 ====================
export const projectsApi = {
  list: () => apiFetch<Project[]>('/api/projects'),
  get: (id: string) => apiFetch<Project>(`/api/projects/${id}`),
  create: (data: { name: string; repo_url?: string; description?: string }) =>
    apiFetch<Project>('/api/projects', { method: 'POST', body: JSON.stringify(data) }),
};

// ==================== 运行 ====================
export const runsApi = {
  list: () => apiFetch<Run[]>('/api/runs'),
  get: (id: string) => apiFetch<Run>(`/api/runs/${id}`),
  create: (data: { title: string; project_id?: string; workflow_template_id?: string }) =>
    apiFetch<Run>('/api/runs', { method: 'POST', body: JSON.stringify(data) }),
  getTimeline: (id: string) => apiFetch<Event[]>(`/api/runs/${id}/timeline`),
  getTasks: (id: string) => apiFetch<Task[]>(`/api/runs/${id}/tasks`),
  getWorkflow: (id: string) => apiFetch<WorkflowGraphData>(`/api/runs/${id}/workflow`),
};

// ==================== 任务 ====================
export const tasksApi = {
  retry: (id: string) => apiFetch<Task>(`/api/tasks/${id}/retry`, { method: 'POST' }),
  cancel: (id: string) => apiFetch<Task>(`/api/tasks/${id}/cancel`, { method: 'POST' }),
};

// ==================== Agent ====================
export const agentsApi = {
  list: () => apiFetch<AgentInstance[]>('/api/agents'),
  get: (id: string) => apiFetch<AgentInstance>(`/api/agents/${id}`),
  pause: (id: string) => apiFetch<AgentInstance>(`/api/agents/${id}/pause`, { method: 'POST' }),
  resume: (id: string) => apiFetch<AgentInstance>(`/api/agents/${id}/resume`, { method: 'POST' }),
  stop: (id: string) => apiFetch<AgentInstance>(`/api/agents/${id}/stop`, { method: 'POST' }),
};

// ==================== 终端 ====================
export const terminalsApi = {
  list: () => apiFetch<TerminalSession[]>('/api/terminals'),
  get: (id: string) => apiFetch<TerminalSession>(`/api/terminals/${id}`),
  create: (data: { task_id: string }) =>
    apiFetch<TerminalSession>('/api/terminals', { method: 'POST', body: JSON.stringify(data) }),
};

// ==================== 系统 ====================
export const systemApi = {
  metrics: () => apiFetch<ResourceSnapshot>('/api/system/metrics'),
  diagnostics: () => apiFetch<DiagnosticsData>('/api/system/diagnostics'),
  healthz: () => apiFetch<HealthStatus>('/healthz'),
  readyz: () => apiFetch<HealthStatus>('/readyz'),
};

// ==================== 技术方案 ====================
export const techStacksApi = {
  list: () => apiFetch<TechStackOption[]>('/api/tech-stacks'),
};

// ==================== 提示词草稿 ====================
export const promptDraftsApi = {
  list: (projectId: string) =>
    apiFetch<PromptDraft[]>(`/api/prompt-drafts?project_id=${encodeURIComponent(projectId)}`),
  generate: (projectId: string, originalInput: string, taskType?: string, techStackId?: string) =>
    apiFetch<PromptDraft>('/api/prompt-drafts/generate', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, original_input: originalInput, task_type: taskType, tech_stack_id: techStackId }),
    }),
  update: (id: string, finalPrompt: string, taskType?: string) =>
    apiFetch<PromptDraft>(`/api/prompt-drafts/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ final_prompt: finalPrompt, task_type: taskType }),
    }),
  send: (id: string) =>
    apiFetch<SendPromptDraftResult>(`/api/prompt-drafts/${id}/send`, { method: 'POST' }),
};

// ==================== 质量门禁 ====================
export const gatesApi = {
  list: (runId: string) =>
    apiFetch<Gate[]>(`/api/gates?run_id=${encodeURIComponent(runId)}`),
  get: (id: string) => apiFetch<Gate>(`/api/gates/${id}`),
  approve: (id: string, approvedBy?: string) =>
    apiFetch<Gate>(`/api/gates/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ approved_by: approvedBy || 'user' }),
    }),
};

// ==================== 工作流模板 ====================
export const workflowTemplatesApi = {
  list: () => apiFetch<WorkflowTemplate[]>('/api/workflow-templates'),
};
