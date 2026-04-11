"use client";

import { useState, useEffect } from "react";
import { apiFetch } from "@/lib/api";

interface Run {
  id: string;
  project_id: string;
  title: string;
  status: string;
  created_at: string;
}

export default function RunsPage() {
  const [runs, setRuns] = useState<Run[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [projectId, setProjectId] = useState("");
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
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setCreating(true);
      setError(null);
      const run = await apiFetch<Run>("/api/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ project_id: projectId, title }),
      });
      setRuns([run, ...runs]);
      setTitle("");
      setProjectId("");
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
            <label className="block text-sm font-medium mb-1">项目 ID</label>
            <input
              type="text"
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              placeholder="输入项目 ID"
              required
            />
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
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr key={run.id} className="border-b hover:bg-gray-50">
                <td className="py-2 text-sm font-mono">{run.id.slice(0, 8)}</td>
                <td className="py-2 text-sm">{run.title}</td>
                <td className="py-2 text-sm">
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      run.status === "running"
                        ? "bg-green-100 text-green-700"
                        : run.status === "pending"
                        ? "bg-yellow-100 text-yellow-700"
                        : run.status === "failed"
                        ? "bg-red-100 text-red-700"
                        : "bg-gray-100 text-gray-700"
                    }`}
                  >
                    {run.status}
                  </span>
                </td>
                <td className="py-2 text-sm">{new Date(run.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
