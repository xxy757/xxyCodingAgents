// projects/page.tsx - 项目管理页面
// 展示项目列表，支持创建新项目。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, type Project } from "@/lib/api";

// ProjectsPage 项目管理页面组件
export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [description, setDescription] = useState("");
  const [creating, setCreating] = useState(false);

  // 页面加载时获取项目列表
  useEffect(() => {
    setLoading(true);
    apiFetch<Project[]>("/api/projects")
      .then(setProjects)
      .catch((e) => {
        setProjects([]);
        setError(e.message);
      })
      .finally(() => setLoading(false));
  }, []);

  // handleCreate 处理创建项目表单提交
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setCreating(true);
      setError(null);
      const project = await apiFetch<Project>("/api/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, repo_url: repoUrl, description }),
      });
      setProjects([project, ...projects]);
      setName("");
      setRepoUrl("");
      setDescription("");
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
        <h1 className="text-2xl font-bold">项目管理</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:outline-none"
        >
          新建项目
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {/* 新建项目表单 */}
      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-gray-50 rounded-lg space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">项目名称</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              placeholder="输入项目名称"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">仓库 URL</label>
            <input
              type="text"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              placeholder="https://github.com/..."
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">描述</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              rows={3}
              placeholder="项目描述"
            />
          </div>
          <button
            type="submit"
            disabled={creating}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-green-400 focus-visible:outline-none"
          >
            {creating ? "创建中..." : "创建"}
          </button>
        </form>
      )}

      {/* 项目列表表格 */}
      {loading ? (
        <div className="text-gray-500">加载中...</div>
      ) : projects.length === 0 ? (
        <p className="text-gray-500">暂无项目</p>
      ) : (
        <table className="w-full border-collapse">
          <thead>
            <tr className="border-b text-left text-sm text-gray-600">
              <th className="pb-2">ID</th>
              <th className="pb-2">名称</th>
              <th className="pb-2">仓库</th>
              <th className="pb-2">创建时间</th>
            </tr>
          </thead>
          <tbody>
            {projects.map((p) => (
              <tr key={p.id} className="border-b hover:bg-gray-50">
                <td className="py-2 text-sm font-mono">{p.id.slice(0, 8)}</td>
                <td className="py-2 text-sm font-medium">{p.name}</td>
                <td className="py-2 text-sm text-gray-500">{p.repo_url || "-"}</td>
                <td className="py-2 text-sm">{new Date(p.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
