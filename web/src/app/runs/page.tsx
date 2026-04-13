"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";

interface Run {
  id: string;
  project_id: string;
  title: string;
  status: string;
  created_at: string;
}

interface Project {
  id: string;
  name: string;
}

interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
}

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

  const statusColor = (status: string) => {
    switch (status) {
      case "running":
        return "bg-green-100 text-green-700";
      case "pending":
        return "bg-yellow-100 text-yellow-700";
      case "failed":
        return "bg-red-100 text-red-700";
      case "completed":
        return "bg-blue-100 text-blue-700";
      default:
        return "bg-gray-100 text-gray-700";
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">运行管理</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          新建 Run
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-gray-50 rounded-lg space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">项目</label>
            <select
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
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
            <label className="block text-sm font-medium mb-1">工作流模板</label>
            <select
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
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
            <label className="block text-sm font-medium mb-1">标题</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              placeholder="输入 Run 标题"
              required
            />
          </div>
          <button
            type="submit"
            disabled={creating}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
          >
            {creating ? "创建中..." : "创建"}
          </button>
        </form>
      )}

      {loading ? (
        <div className="text-gray-500">加载中...</div>
      ) : runs.length === 0 ? (
        <p className="text-gray-500">暂无运行记录</p>
      ) : (
        <table className="w-full border-collapse">
          <thead>
            <tr className="border-b text-left text-sm text-gray-600">
              <th className="pb-2">ID</th>
              <th className="pb-2">标题</th>
              <th className="pb-2">状态</th>
              <th className="pb-2">创建时间</th>
              <th className="pb-2">操作</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr key={run.id} className="border-b hover:bg-gray-50">
                <td className="py-2 text-sm font-mono">{run.id.slice(0, 8)}</td>
                <td className="py-2 text-sm">{run.title}</td>
                <td className="py-2 text-sm">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColor(run.status)}`}>
                    {run.status}
                  </span>
                </td>
                <td className="py-2 text-sm">{new Date(run.created_at).toLocaleString()}</td>
                <td className="py-2 text-sm">
                  <Link href={`/runs/${run.id}`} className="text-blue-600 hover:underline">
                    查看详情
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
