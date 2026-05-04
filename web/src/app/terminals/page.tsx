// terminals/page.tsx - 终端管理
// 卡片列表展示终端会话，支持创建新会话。
"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { apiFetch, type TerminalSession } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  Terminal,
  Plus,
  ArrowRight,
  Clock,
} from "@phosphor-icons/react/dist/ssr";

export default function TerminalsPage() {
  const [terminals, setTerminals] = useState<TerminalSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [taskId, setTaskId] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch<TerminalSession[]>("/api/terminals")
      .then(setTerminals)
      .catch(() => setTerminals([]))
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setError(null);
    try {
      const ts = await apiFetch<TerminalSession>("/api/terminals", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ task_id: taskId }),
      });
      setTerminals([ts, ...terminals]);
      setTaskId("");
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
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">终端</h1>
          <p className="text-sm text-zinc-500 mt-1">tmux 会话管理</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 px-4 py-2.5 bg-zinc-900 text-white rounded-xl text-sm font-medium
                     hover:bg-zinc-800 pressable transition-colors duration-200"
        >
          <Plus className="w-4 h-4" />
          新建终端
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="card-bezel p-6 space-y-4">
          <div>
            <label className="block text-xs font-medium text-zinc-500 mb-1.5">Task ID</label>
            <input
              type="text"
              value={taskId}
              onChange={(e) => setTaskId(e.target.value)}
              className="w-full bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-2.5 text-sm
                         placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-accent-500/20
                         focus:border-accent-500 transition-all"
              placeholder="输入关联的 Task ID"
              required
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
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="card-bezel p-5">
              <div className="skeleton h-4 w-32 mb-2" />
              <div className="skeleton h-3 w-48" />
            </div>
          ))}
        </div>
      ) : terminals.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="space-y-2 stagger">
          {terminals.map((ts) => (
            <Link
              key={ts.id}
              href={`/terminals/${ts.id}`}
              className="card-bezel p-5 flex items-center justify-between group block"
            >
              <div className="flex items-center gap-4 min-w-0">
                <div className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center shrink-0
                                group-hover:bg-accent-50 transition-colors duration-200">
                  <Terminal className="w-4 h-4 text-zinc-500 group-hover:text-accent-600 transition-colors" />
                </div>
                <div className="min-w-0">
                  <h3 className="text-sm font-semibold text-zinc-900 font-mono">{ts.tmux_session}</h3>
                  <div className="flex items-center gap-3 mt-1 text-xs text-zinc-400">
                    <span>Task: {ts.task_id.slice(0, 8)}</span>
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {new Date(ts.created_at).toLocaleString("zh-CN")}
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <StatusBadge status={ts.status} />
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
        <Terminal className="w-6 h-6 text-zinc-400" />
      </div>
      <p className="text-sm font-medium text-zinc-600">暂无终端会话</p>
      <p className="text-xs text-zinc-400 mt-1">终端会话会在 Agent 启动时自动创建</p>
    </div>
  );
}
