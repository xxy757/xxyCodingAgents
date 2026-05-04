// runs/page.tsx - 运行管理
// 卡片网格展示运行列表，支持创建新运行。
"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { apiFetch, type Run, type Project, type WorkflowTemplate } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  PlayCircle,
  Plus,
  ArrowRight,
  Clock,
} from "@phosphor-icons/react/dist/ssr";

export default function RunsPage() {
  const [runs, setRuns] = useState<Run[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [templates, setTemplates] = useState<WorkflowTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [projectId, setProjectId] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    apiFetch<Run[]>("/api/runs")
      .then(setRuns)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
    apiFetch<Project[]>("/api/projects").then(setProjects).catch(() => {});
    apiFetch<WorkflowTemplate[]>("/api/workflow-templates").then(setTemplates).catch(() => {});
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
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
    <div className="space-y-6 animate-fade-up">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">运行</h1>
          <p className="text-sm text-zinc-500 mt-1">管理和监控任务执行</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 px-4 py-2.5 bg-zinc-900 text-white rounded-xl text-sm font-medium
                     hover:bg-zinc-800 pressable transition-colors duration-200"
        >
          <Plus className="w-4 h-4" />
          新建 Run
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="card-bezel p-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1.5">项目</label>
              <select
                value={projectId}
                onChange={(e) => setProjectId(e.target.value)}
                className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                           focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
                required
              >
                <option value="">选择项目</option>
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1.5">工作流模板</label>
              <select
                value={templateId}
                onChange={(e) => setTemplateId(e.target.value)}
                className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                           focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
              >
                <option value="">无模板</option>
                {templates.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1.5">标题</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                           focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
                required
              />
            </div>
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={creating}
              className="px-4 py-2.5 bg-accent-600 text-white rounded-xl text-sm font-medium
                         hover:bg-accent-700 disabled:opacity-50 pressable transition-colors"
            >
              {creating ? "创建中..." : "创建"}
            </button>
            <button
              type="button"
              onClick={() => setShowForm(false)}
              className="px-4 py-2.5 border border-zinc-200 rounded-xl text-sm text-zinc-600
                         hover:bg-zinc-50 pressable transition-colors"
            >
              取消
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="card-bezel p-5">
              <div className="flex items-center justify-between">
                <div className="skeleton h-4 w-32" />
                <div className="skeleton h-6 w-16 rounded-full" />
              </div>
            </div>
          ))}
        </div>
      ) : runs.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="space-y-2 stagger">
          {runs.map((run) => (
            <Link
              key={run.id}
              href={`/runs/${run.id}`}
              className="card-bezel p-5 flex items-center justify-between group block"
            >
              <div className="flex items-center gap-4 min-w-0">
                <div className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center shrink-0
                                group-hover:bg-accent-50 transition-colors duration-200">
                  <PlayCircle className="w-4 h-4 text-zinc-500 group-hover:text-accent-600 transition-colors" />
                </div>
                <div className="min-w-0">
                  <h3 className="text-sm font-semibold text-zinc-900 truncate">{run.title}</h3>
                  <div className="flex items-center gap-3 mt-1 text-xs text-zinc-400">
                    <span className="font-mono">{run.id.slice(0, 8)}</span>
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {new Date(run.created_at).toLocaleString("zh-CN")}
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <StatusBadge status={run.status} />
                <ArrowRight className="w-4 h-4 text-zinc-300 group-hover:text-zinc-600 group-hover:translate-x-0.5 transition-all" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="card-bezel p-12 text-center">
      <div className="w-12 h-12 rounded-2xl bg-zinc-100 flex items-center justify-center mx-auto mb-4">
        <PlayCircle className="w-6 h-6 text-zinc-400" />
      </div>
      <p className="text-sm font-medium text-zinc-600">暂无运行记录</p>
      <p className="text-xs text-zinc-400 mt-1">创建第一个 Run 开始执行任务</p>
    </div>
  );
}
