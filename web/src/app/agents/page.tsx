// agents/page.tsx - Agent 管理
// 卡片网格展示 Agent 实例，支持暂停、恢复和停止操作。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, type AgentInstance } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  Robot,
  Pause,
  Play,
  Stop,
  Terminal,
} from "@phosphor-icons/react/dist/ssr";

export default function AgentsPage() {
  const [agents, setAgents] = useState<AgentInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch<AgentInstance[]>("/api/agents")
      .then(setAgents)
      .catch(() => setAgents([]))
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
    <div className="space-y-6 animate-fade-up">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">Agent</h1>
        <p className="text-sm text-zinc-500 mt-1">管理和监控 Agent 实例</p>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
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
      ) : agents.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 stagger">
          {agents.map((agent) => (
            <div key={agent.id} className="card-bezel p-5">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center">
                    <Robot className="w-4 h-4 text-zinc-500" />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-zinc-900 font-mono">{agent.id.slice(0, 8)}</h3>
                    <p className="text-xs text-zinc-400">{agent.agent_kind}</p>
                  </div>
                </div>
                <StatusBadge status={agent.status} />
              </div>

              {agent.tmux_session && (
                <div className="flex items-center gap-2 text-xs text-zinc-500 mb-4 px-3 py-2 bg-zinc-50 rounded-lg">
                  <Terminal className="w-3 h-3" />
                  <span className="font-mono">{agent.tmux_session}</span>
                </div>
              )}

              <div className="flex items-center gap-4 text-xs text-zinc-400 mb-4">
                <span>创建: {new Date(agent.created_at).toLocaleString("zh-CN")}</span>
              </div>

              <div className="flex gap-2">
                {agent.status === "running" && (
                  <button
                    onClick={() => handleAction(agent.id, "pause")}
                    className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-amber-50 text-amber-700
                               border border-amber-200/60 rounded-lg hover:bg-amber-100 pressable transition-colors"
                  >
                    <Pause className="w-3 h-3" />
                    暂停
                  </button>
                )}
                {agent.status === "paused" && (
                  <button
                    onClick={() => handleAction(agent.id, "resume")}
                    className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-accent-50 text-accent-700
                               border border-accent-200/60 rounded-lg hover:bg-accent-100 pressable transition-colors"
                  >
                    <Play className="w-3 h-3" />
                    恢复
                  </button>
                )}
                {agent.status !== "stopped" && agent.status !== "failed" && (
                  <button
                    onClick={() => handleAction(agent.id, "stop")}
                    className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-red-50 text-red-700
                               border border-red-200/60 rounded-lg hover:bg-red-100 pressable transition-colors"
                  >
                    <Stop className="w-3 h-3" />
                    停止
                  </button>
                )}
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
        <Robot className="w-6 h-6 text-zinc-400" />
      </div>
      <p className="text-sm font-medium text-zinc-600">暂无 Agent 实例</p>
      <p className="text-xs text-zinc-400 mt-1">Agent 会在任务调度时自动创建</p>
    </div>
  );
}
