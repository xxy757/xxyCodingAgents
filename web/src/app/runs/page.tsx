// runs/page.tsx - 运行管理页面
// 展示运行列表，支持创建新运行（可关联项目和工作流模板）。
"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { apiFetch, type Run, type Project, type WorkflowTemplate } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";

// RunsPage 运行管理页面组件
export default function RunsPage() {
  const [runs, setRuns] = useState<Run[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [templates, setTemplates] = useState<WorkflowTemplate[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [projectId, setProjectId] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [creating, setCreating] = useState(false);

  // 加载运行列表、项目列表和工作流模板列表
  useEffect(() => {
    setLoading(true);
    apiFetch<Run[]>("/api/runs")
      .then(setRuns)
      .catch((e) => {
        setRuns([]);
        setError(e.message);
      })
      .finally(() => setLoading(false));

    apiFetch<Project[]>("/api/projects")
      .then(setProjects)
      .catch(() => {});

    apiFetch<WorkflowTemplate[]>("/api/workflow-templates")
      .then(setTemplates)
      .catch(() => {});
  }, []);

  // handleCreate 处理创建运行表单提交
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setCreating(true);
      setError(null);
      const body: Record<string, string> = { title };
      if (projectId) body.project_id = projectId;
      if (templateId) body.workflow_template_id = templateId;
      const run = await apiFetch<Run>("/api/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      setRuns([run, ...runs]);
      setTitle("");
      setProjectId("");
      setTemplateId("");
      setShowForm(false);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">运行管理</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
        >
          新建 Run
        </button>
      </div>

      {error && (
        <div role="alert" className="mb-4 p-3 bg-error-50 border border-error-500 rounded text-error-700 text-sm">
          操作失败：{error}
        </div>
      )}

      {/* 新建运行表单 */}
      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-neutral-50 rounded-lg border border-neutral-200 space-y-3">
          <div>
            <label htmlFor="run-project" className="block text-sm font-medium mb-1">项目</label>
            <select
              id="run-project"
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              className="w-full border border-neutral-300 rounded-md px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
              required
            >
              <option value="">选择项目</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="run-template" className="block text-sm font-medium mb-1">工作流模板</label>
            <select
              id="run-template"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full border border-neutral-300 rounded-md px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
            >
              <option value="">无模板（手动管理）</option>
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} — {t.description}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="run-title" className="block text-sm font-medium mb-1">标题</label>
            <input
              id="run-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border border-neutral-300 rounded-md px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
              placeholder="输入 Run 标题"
              required
            />
          </div>
          <button
            type="submit"
            disabled={creating}
            className="px-4 py-2 bg-success-600 text-white rounded-md hover:bg-success-700 disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-success-500 focus-visible:outline-none"
          >
            {creating ? "创建中..." : "创建"}
          </button>
        </form>
      )}

      {/* 运行列表表格 */}
      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="skeleton h-12 w-full" />
          ))}
        </div>
      ) : runs.length === 0 ? (
        <p className="text-neutral-500">暂无运行记录</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-sm text-neutral-600">
                <th className="pb-2">ID</th>
                <th className="pb-2">标题</th>
                <th className="pb-2">状态</th>
                <th className="pb-2">创建时间</th>
                <th className="pb-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => (
                <tr key={run.id} className="border-b border-neutral-100 hover:bg-neutral-50">
                  <td className="py-2 text-sm font-mono">{run.id.slice(0, 8)}</td>
                  <td className="py-2 text-sm">{run.title}</td>
                  <td className="py-2 text-sm">
                    <StatusBadge status={run.status} />
                  </td>
                  <td className="py-2 text-sm">{new Date(run.created_at).toLocaleString()}</td>
                  <td className="py-2 text-sm">
                    <Link href={`/runs/${run.id}`} className="text-primary-600 hover:underline focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none rounded">
                      查看详情
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
