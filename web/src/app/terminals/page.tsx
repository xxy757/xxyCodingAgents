"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";

interface TerminalSession {
  id: string;
  task_id: string;
  tmux_session: string;
  status: string;
  created_at: string;
}

export default function TerminalsPage() {
  const [terminals, setTerminals] = useState<TerminalSession[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [taskId, setTaskId] = useState("");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    setLoading(true);
    apiFetch<TerminalSession[]>("/api/terminals")
      .then(setTerminals)
      .catch((e) => {
        setTerminals([]);
        setError(e.message);
      })
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setCreating(true);
      setError(null);
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
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">终端管理</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          新建终端
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-gray-50 rounded-lg space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">Task ID</label>
            <input
              type="text"
              value={taskId}
              onChange={(e) => setTaskId(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
              placeholder="输入关联的 Task ID"
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
      ) : terminals.length === 0 ? (
        <p className="text-gray-500">暂无终端会话</p>
      ) : (
        <table className="w-full border-collapse">
          <thead>
            <tr className="border-b text-left text-sm text-gray-600">
              <th className="pb-2">ID</th>
              <th className="pb-2">Task ID</th>
              <th className="pb-2">tmux 会话</th>
              <th className="pb-2">状态</th>
              <th className="pb-2">创建时间</th>
              <th className="pb-2">操作</th>
            </tr>
          </thead>
          <tbody>
            {terminals.map((ts) => (
              <tr key={ts.id} className="border-b hover:bg-gray-50">
                <td className="py-2 text-sm font-mono">{ts.id.slice(0, 8)}</td>
                <td className="py-2 text-sm font-mono">{ts.task_id.slice(0, 8)}</td>
                <td className="py-2 text-sm font-mono">{ts.tmux_session}</td>
                <td className="py-2 text-sm">
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      ts.status === "active"
                        ? "bg-green-100 text-green-700"
                        : "bg-gray-100 text-gray-700"
                    }`}
                  >
                    {ts.status}
                  </span>
                </td>
                <td className="py-2 text-sm">{new Date(ts.created_at).toLocaleString()}</td>
                <td className="py-2 text-sm">
                  <Link
                    href={`/terminals/${ts.id}`}
                    className="text-blue-600 hover:underline"
                  >
                    打开
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
