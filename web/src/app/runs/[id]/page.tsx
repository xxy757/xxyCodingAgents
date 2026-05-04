// runs/[id]/page.tsx - 运行详情
// 任务列表 + 事件时间线 + 工作流图（ReactFlow）。
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, type Task, type Event, type Run, type Gate, approveGate, listGates, statusHex } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type NodeTypes,
  Position,
  Handle,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  ArrowLeft,
  ListChecks,
  Clock,
  GitBranch,
} from "@phosphor-icons/react/dist/ssr";

interface WorkflowGraph {
  nodes: {
    id: string;
    type: string;
    data: { label: string; status: string; task_type: string; task_id: string; gate_id?: string; gate_type?: string };
    position: { x: number; y: number };
  }[];
  edges: { id: string; source: string; target: string }[];
}

function statusBorderColor(status: string): string {
  return (statusHex[status] || statusHex.default).border;
}

function statusBgColor(status: string): string {
  return (statusHex[status] || statusHex.default).bg;
}

function TaskNode({ data }: { data: { label: string; status: string; task_type: string } }) {
  const borderColor = statusBorderColor(data.status);
  const bgColor = statusBgColor(data.status);
  return (
    <div
      style={{
        padding: "10px 16px",
        borderRadius: 10,
        border: `1.5px solid ${borderColor}`,
        background: bgColor,
        minWidth: 140,
        fontSize: 12,
        fontFamily: "var(--font-sans)",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: borderColor, width: 8, height: 8 }} />
      <div style={{ fontWeight: 600, marginBottom: 4, color: "#18181b" }}>{data.label}</div>
      <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <span
          style={{
            padding: "2px 8px",
            borderRadius: 6,
            background: borderColor,
            color: "#fff",
            fontSize: 10,
            fontWeight: 500,
          }}
        >
          {data.status || "pending"}
        </span>
        <span style={{ color: "#71717a", fontSize: 11 }}>{data.task_type}</span>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ background: borderColor, width: 8, height: 8 }} />
    </div>
  );
}

function gateStatusBorderColor(status: string): string {
  switch (status) {
    case "passed": return "#059669";
    case "failed": return "#dc2626";
    case "pending": return "#d97706";
    default: return "#a1a1aa";
  }
}

function GateNode({ data }: { data: { label: string; status: string; gate_type: string; gate_id: string; onApprove?: (gateId: string) => void } }) {
  const borderColor = gateStatusBorderColor(data.status);
  return (
    <div
      style={{
        padding: "8px 14px",
        borderRadius: 8,
        border: `1.5px dashed ${borderColor}`,
        background: data.status === "passed" ? "#ecfdf5" : data.status === "failed" ? "#fef2f2" : "#fffbeb",
        minWidth: 120,
        fontSize: 11,
        fontFamily: "var(--font-sans)",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: borderColor, width: 8, height: 8 }} />
      <div style={{ fontWeight: 600, marginBottom: 2, display: "flex", alignItems: "center", gap: 4, color: "#18181b" }}>
        {data.label}
      </div>
      <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
        <span style={{ padding: "2px 6px", borderRadius: 4, background: borderColor, color: "#fff", fontSize: 9, fontWeight: 500 }}>
          {data.status || "pending"}
        </span>
        <span style={{ color: "#71717a", fontSize: 9 }}>{data.gate_type}</span>
      </div>
      {data.gate_type === "manual" && data.status === "pending" && data.onApprove && (
        <button
          onClick={() => data.onApprove!(data.gate_id)}
          style={{ marginTop: 6, padding: "3px 10px", fontSize: 10, fontWeight: 500, background: "#059669", color: "#fff", border: "none", borderRadius: 6, cursor: "pointer" }}
        >
          通过
        </button>
      )}
      <Handle type="source" position={Position.Bottom} style={{ background: borderColor, width: 8, height: 8 }} />
    </div>
  );
}

const nodeTypes: NodeTypes = {
  task: TaskNode as any,
  gate: GateNode as any,
};

export default function RunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState<string>("");
  const [run, setRun] = useState<Run | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [gates, setGates] = useState<Gate[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"tasks" | "timeline" | "workflow">("tasks");
  const [workflowGraph, setWorkflowGraph] = useState<WorkflowGraph | null>(null);
  const router = useRouter();

  useEffect(() => { params.then((p) => setId(p.id)); }, [params]);

  useEffect(() => {
    if (!id) return;
    apiFetch<Run>(`/api/runs/${id}`).then(setRun).catch((e) => setError(e.message));
    apiFetch<Task[]>(`/api/runs/${id}/tasks`).then(setTasks).catch(() => {});
    apiFetch<Event[]>(`/api/runs/${id}/timeline`).then(setEvents).catch(() => {});
    apiFetch<WorkflowGraph>(`/api/runs/${id}/workflow`).then(setWorkflowGraph).catch(() => {});
    listGates(id).then(setGates).catch(() => {});
  }, [id]);

  const handleRetry = async (taskId: string) => {
    try {
      await apiFetch(`/api/tasks/${taskId}/retry`, { method: "POST" });
      apiFetch<Task[]>(`/api/runs/${id}/tasks`).then(setTasks);
    } catch (e: any) { setError(e.message); }
  };

  const handleCancel = async (taskId: string) => {
    try {
      await apiFetch(`/api/tasks/${taskId}/cancel`, { method: "POST" });
      apiFetch<Task[]>(`/api/runs/${id}/tasks`).then(setTasks);
    } catch (e: any) { setError(e.message); }
  };

  const handleApproveGate = async (gateId: string) => {
    try {
      await approveGate(gateId, "user");
      listGates(id).then(setGates);
      apiFetch<WorkflowGraph>(`/api/runs/${id}/workflow`).then(setWorkflowGraph).catch(() => {});
    } catch (e: any) { setError(e.message); }
  };

  if (!id) return <div className="text-sm text-zinc-400 py-12 text-center">加载中...</div>;

  if (error && !run) {
    return (
      <div className="space-y-4">
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
        <button onClick={() => router.back()} className="text-sm text-accent-600 hover:underline">返回</button>
      </div>
    );
  }

  const tabs = [
    { key: "tasks" as const, label: "任务", count: tasks.length, icon: ListChecks },
    { key: "timeline" as const, label: "时间线", count: events.length, icon: Clock },
    { key: "workflow" as const, label: "工作流", count: null, icon: GitBranch },
  ];

  return (
    <div className="space-y-6 animate-fade-up">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.back()}
          className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center hover:bg-zinc-200 pressable transition-colors"
        >
          <ArrowLeft className="w-4 h-4 text-zinc-600" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight text-zinc-900 truncate">
            {run?.title || `Run ${id.slice(0, 8)}`}
          </h1>
        </div>
        {run && <StatusBadge status={run.status} />}
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-zinc-100 rounded-xl w-fit">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                activeTab === tab.key
                  ? "bg-white text-zinc-900 shadow-xs"
                  : "text-zinc-500 hover:text-zinc-700"
              }`}
            >
              <Icon className="w-4 h-4" />
              {tab.label}
              {tab.count !== null && (
                <span className={`text-xs px-1.5 py-0.5 rounded-full ${
                  activeTab === tab.key ? "bg-zinc-100 text-zinc-600" : "bg-zinc-200/60 text-zinc-400"
                }`}>
                  {tab.count}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Tasks Tab */}
      {activeTab === "tasks" && (
        <div>
          {tasks.length === 0 ? (
            <div className="card-bezel p-12 text-center">
              <p className="text-sm text-zinc-400">暂无任务</p>
            </div>
          ) : (
            <div className="space-y-2 stagger">
              {tasks.map((task) => (
                <div key={task.id} className="card-bezel p-4 flex items-center justify-between">
                  <div className="flex items-center gap-4 min-w-0">
                    <div className="min-w-0">
                      <h3 className="text-sm font-semibold text-zinc-900">{task.title || task.task_type}</h3>
                      <div className="flex items-center gap-3 mt-1 text-xs text-zinc-400">
                        <span className="font-mono">{task.id.slice(0, 8)}</span>
                        <span>{task.task_type}</span>
                        <span>{task.priority}</span>
                        <span>{task.resource_class}</span>
                        <span>尝试 #{task.attempt_no}</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    <StatusBadge status={task.status} />
                    {(task.status === "failed" || task.status === "evicted") && (
                      <button
                        onClick={() => handleRetry(task.id)}
                        className="px-3 py-1.5 text-xs font-medium text-accent-700 bg-accent-50 border border-accent-200/60
                                   rounded-lg hover:bg-accent-100 pressable transition-colors"
                      >
                        重试
                      </button>
                    )}
                    {(task.status === "queued" || task.status === "running") && (
                      <button
                        onClick={() => handleCancel(task.id)}
                        className="px-3 py-1.5 text-xs font-medium text-red-700 bg-red-50 border border-red-200/60
                                   rounded-lg hover:bg-red-100 pressable transition-colors"
                      >
                        取消
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Timeline Tab */}
      {activeTab === "timeline" && (
        <div>
          {events.length === 0 ? (
            <div className="card-bezel p-12 text-center">
              <p className="text-sm text-zinc-400">暂无事件</p>
            </div>
          ) : (
            <div className="space-y-2 stagger">
              {events.map((event) => (
                <div key={event.id} className="card-bezel p-4 flex items-start gap-4">
                  <div className="text-xs text-zinc-400 min-w-[130px] pt-0.5 font-mono">
                    {new Date(event.created_at).toLocaleString("zh-CN")}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-0.5 bg-zinc-100 rounded-md text-xs font-medium text-zinc-600">
                        {event.event_type}
                      </span>
                      <span className="text-sm text-zinc-700">{event.message}</span>
                    </div>
                    {event.task_id && (
                      <div className="text-xs text-zinc-400 mt-1 font-mono">
                        Task: {event.task_id.slice(0, 8)}
                        {event.agent_id && ` | Agent: ${event.agent_id.slice(0, 8)}`}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Workflow Tab */}
      {activeTab === "workflow" && (
        <div className="card-bezel overflow-hidden">
          {!workflowGraph || workflowGraph.nodes.length === 0 ? (
            <div className="p-12 text-center">
              <p className="text-sm text-zinc-400">暂无工作流数据</p>
            </div>
          ) : (
            <div style={{ width: "100%", height: 500 }}>
              <ReactFlow
                nodes={workflowGraph.nodes.map((n) => ({
                  id: n.id,
                  type: n.type || "task",
                  data: {
                    ...n.data,
                    onApprove: n.type === "gate" && n.data.gate_type === "manual" && n.data.status !== "passed" ? handleApproveGate : undefined,
                  },
                  position: { x: n.position.x, y: n.position.y },
                }))}
                edges={workflowGraph.edges.map((e) => ({
                  id: e.id,
                  source: e.source,
                  target: e.target,
                  animated: true,
                  style: { stroke: "#d4d4d8", strokeWidth: 1.5 },
                }))}
                nodeTypes={nodeTypes}
                fitView
                minZoom={0.3}
                maxZoom={2}
              >
                <Background color="#e4e4e7" gap={20} />
                <Controls />
                <MiniMap
                  nodeColor={(node) => statusBorderColor((node.data as any)?.status || "")}
                  maskColor="rgba(0,0,0,0.06)"
                  style={{ borderRadius: 8 }}
                />
              </ReactFlow>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
