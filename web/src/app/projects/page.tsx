// projects/page.tsx - 项目管理
// 卡片网格展示项目列表，支持创建新项目。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, type Project } from "@/lib/api";
import {
  FolderNotchOpenIcon as FolderNotchOpen,
  Plus,
  GitBranch,
  Clock,
} from "@phosphor-icons/react/dist/ssr";

function repoDisplayPath(url: string): string {
  try {
    return new URL(url).pathname.slice(1).replace(/\.git$/, "");
  } catch {
    // SSH 格式 git@host:org/repo.git 或本地路径
    const ssh = url.match(/[^:]+:(.+)/);
    if (ssh) return ssh[1].replace(/\.git$/, "");
    return url;
  }
}

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [description, setDescription] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch<Project[]>("/api/projects")
      .then(setProjects)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const p = await apiFetch<Project>("/api/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, repo_url: repoUrl, description }),
      });
      setProjects([p, ...projects]);
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
    <div className="space-y-6 animate-fade-up">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">项目</h1>
          <p className="text-sm text-zinc-500 mt-1">管理你的代码项目</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 px-4 py-2.5 bg-zinc-900 text-white rounded-xl text-sm font-medium
                     hover:bg-zinc-800 pressable transition-colors duration-200"
        >
          <Plus className="w-4 h-4" />
          新建项目
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">
          {error}
        </div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="card-bezel p-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1.5">项目名称</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                           focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
                required
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1.5">仓库地址</label>
              <input
                type="text"
                value={repoUrl}
                onChange={(e) => setRepoUrl(e.target.value)}
                placeholder="https://github.com/..."
                className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                           placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-accent-500/20
                           focus:border-accent-500 transition-all"
              />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-zinc-500 mb-1.5">描述</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                         focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500 transition-all"
            />
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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="card-bezel p-5">
              <div className="skeleton h-4 w-24 mb-3" />
              <div className="skeleton h-3 w-full mb-2" />
              <div className="skeleton h-3 w-2/3" />
            </div>
          ))}
        </div>
      ) : projects.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 stagger">
          {projects.map((p) => (
            <div key={p.id} className="card-bezel p-5 group">
              <div className="flex items-start gap-3">
                <div className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center shrink-0
                                group-hover:bg-accent-50 transition-colors duration-200">
                  <FolderNotchOpen className="w-4 h-4 text-zinc-500 group-hover:text-accent-600 transition-colors" />
                </div>
                <div className="min-w-0 flex-1">
                  <h3 className="text-sm font-semibold text-zinc-900 truncate">{p.name}</h3>
                  {p.description && (
                    <p className="text-xs text-zinc-500 mt-1 line-clamp-2">{p.description}</p>
                  )}
                </div>
              </div>
              <div className="mt-4 flex items-center gap-4 text-xs text-zinc-400">
                {p.repo_url && (
                  <span className="flex items-center gap-1">
                    <GitBranch className="w-3 h-3" />
                    {repoDisplayPath(p.repo_url)}
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  {new Date(p.created_at).toLocaleDateString("zh-CN")}
                </span>
              </div>
            </div>
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
        <FolderNotchOpen className="w-6 h-6 text-zinc-400" />
      </div>
      <p className="text-sm font-medium text-zinc-600">暂无项目</p>
      <p className="text-xs text-zinc-400 mt-1">创建第一个项目开始使用</p>
    </div>
  );
}
