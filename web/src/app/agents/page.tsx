"use client";

import { useState, useEffect } from "react";
import { apiFetch } from "@/lib/api";

interface AgentInstance {
  id: string;
  run_id: string;
  task_id: string;
  agent_kind: string;
  status: string;
  tmux_session: string;
  created_at: string;
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<AgentInstance[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    apiFetch<AgentInstance[]>("/api/agents")
      .then(setAgents)
      .catch((e) => {
        setAgents([]);
        setError(e.message);
      })
      .finally(() => setLoading(false));
  }, []);

  const handleAction = async (id: string, action: string) => {
    try {
      setError(null);
      await apiFetch(`/api/agents/${id}/${action}`, { method: "POST" });
      const updated = await apiFetch<AgentInstance[]>("/api/agents");
      setAgents(updated);
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Agent 管理</h1>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {loading ? (
        <div className="text-gray-500">加载中...</div>
      ) : agents.length === 0 ? (
        <p className="text-gray-500">暂无 Agent 实例</p>
      ) : (
        <table className="w-full border-collapse">
          <thead>
            <tr className="border-b text-left text-sm text-gray-600">
              <th className="pb-2">ID</th>
              <th className="pb-2">类型</th>
              <th className="pb-2">状态</th>
              <th className="pb-2">tmux 会话</th>
              <th className="pb-2">创建时间</th>
              <th className="pb-2">操作</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((agent) => (
              <tr key={agent.id} className="border-b hover:bg-gray-50">
                <td className="py-2 text-sm font-mono">{agent.id.slice(0, 8)}</td>
                <td className="py-2 text-sm">{agent.agent_kind}</td>
                <td className="py-2 text-sm">
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      agent.status === "running"
                        ? "bg-green-100 text-green-700"
                        : agent.status === "paused"
                        ? "bg-yellow-100 text-yellow-700"
                        : agent.status === "stopped" || agent.status === "failed"
                        ? "bg-red-100 text-red-700"
                        : "bg-gray-100 text-gray-700"
                    }`}
                  >
                    {agent.status}
                  </span>
                </td>
                <td className="py-2 text-sm font-mono">{agent.tmux_session || "-"}</td>
                <td className="py-2 text-sm">{new Date(agent.created_at).toLocaleString()}</td>
                <td className="py-2 text-sm space-x-2">
                  {agent.status === "running" && (
                    <button
                      onClick={() => handleAction(agent.id, "pause")}
                      className="px-2 py-1 bg-yellow-500 text-white rounded text-xs hover:bg-yellow-600"
                    >
                      暂停
                    </button>
                  )}
                  {agent.status === "paused" && (
                    <button
                      onClick={() => handleAction(agent.id, "resume")}
                      className="px-2 py-1 bg-blue-500 text-white rounded text-xs hover:bg-blue-600"
                    >
                      恢复
                    </button>
                  )}
                  {agent.status !== "stopped" && (
                    <button
                      onClick={() => handleAction(agent.id, "stop")}
                      className="px-2 py-1 bg-red-500 text-white rounded text-xs hover:bg-red-600"
                    >
                      停止
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
