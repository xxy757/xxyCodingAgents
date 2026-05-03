// agents/page.tsx - Agent 管理页面
// 展示 Agent 实例列表，支持暂停、恢复和停止操作。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, type AgentInstance } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";

// AgentsPage Agent 管理页面组件
export default function AgentsPage() {
  const [agents, setAgents] = useState<AgentInstance[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // 加载 Agent 列表
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

  // handleAction 执行 Agent 操作（暂停、恢复、停止）
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
        <div role="alert" className="mb-4 p-3 bg-error-50 border border-error-500 rounded text-error-700 text-sm">
          操作失败：{error}
        </div>
      )}

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="skeleton h-12 w-full" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <p className="text-neutral-500">暂无 Agent 实例</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-sm text-neutral-600">
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
                <tr key={agent.id} className="border-b border-neutral-100 hover:bg-neutral-50">
                  <td className="py-2 text-sm font-mono">{agent.id.slice(0, 8)}</td>
                  <td className="py-2 text-sm">{agent.agent_kind}</td>
                  <td className="py-2 text-sm">
                    <StatusBadge status={agent.status} />
                  </td>
                  <td className="py-2 text-sm font-mono">{agent.tmux_session || "-"}</td>
                  <td className="py-2 text-sm">{new Date(agent.created_at).toLocaleString()}</td>
                  <td className="py-2 text-sm space-x-2">
                    {agent.status === "running" && (
                      <button
                        onClick={() => handleAction(agent.id, "pause")}
                        aria-label={`暂停 Agent ${agent.id.slice(0, 8)}`}
                        className="px-3 py-2 bg-warning-500 text-white rounded-md text-sm hover:bg-warning-600 focus-visible:ring-2 focus-visible:ring-warning-500 focus-visible:outline-none"
                      >
                        暂停
                      </button>
                    )}
                    {agent.status === "paused" && (
                      <button
                        onClick={() => handleAction(agent.id, "resume")}
                        aria-label={`恢复 Agent ${agent.id.slice(0, 8)}`}
                        className="px-3 py-2 bg-primary-600 text-white rounded-md text-sm hover:bg-primary-700 focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
                      >
                        恢复
                      </button>
                    )}
                    {agent.status !== "stopped" && (
                      <button
                        onClick={() => handleAction(agent.id, "stop")}
                        aria-label={`停止 Agent ${agent.id.slice(0, 8)}`}
                        className="px-3 py-2 bg-error-600 text-white rounded-md text-sm hover:bg-error-700 focus-visible:ring-2 focus-visible:ring-error-500 focus-visible:outline-none"
                      >
                        停止
                      </button>
                    )}
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
