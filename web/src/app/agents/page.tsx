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
      // 操作成功后刷新列表
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
                  <StatusBadge status={agent.status} />
                </td>
                <td className="py-2 text-sm font-mono">{agent.tmux_session || "-"}</td>
                <td className="py-2 text-sm">{new Date(agent.created_at).toLocaleString()}</td>
                <td className="py-2 text-sm space-x-2">
                  {/* 运行中的 Agent 可暂停 */}
                  {agent.status === "running" && (
                    <button
                      onClick={() => handleAction(agent.id, "pause")}
                      className="px-3 py-2 bg-yellow-500 text-white rounded text-sm hover:bg-yellow-600 focus-visible:ring-2 focus-visible:ring-yellow-400 focus-visible:outline-none"
                    >
                      暂停
                    </button>
                  )}
                  {/* 已暂停的 Agent 可恢复 */}
                  {agent.status === "paused" && (
                    <button
                      onClick={() => handleAction(agent.id, "resume")}
                      className="px-3 py-2 bg-blue-500 text-white rounded text-sm hover:bg-blue-600 focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:outline-none"
                    >
                      恢复
                    </button>
                  )}
                  {/* 未停止的 Agent 可停止 */}
                  {agent.status !== "stopped" && (
                    <button
                      onClick={() => handleAction(agent.id, "stop")}
                      className="px-3 py-2 bg-red-500 text-white rounded text-sm hover:bg-red-600 focus-visible:ring-2 focus-visible:ring-red-400 focus-visible:outline-none"
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
