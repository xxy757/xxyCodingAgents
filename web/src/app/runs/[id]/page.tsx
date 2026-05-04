// runs/[id]/page.tsx - 运行详情页面
// 展示单个运行的任务列表、事件时间线和工作流图（ReactFlow）。
// 支持任务重试和取消操作。
"use client";

import { useEffect, useState, useCallback } from "react";
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

// WorkflowGraph 工作流图数据接口（ReactFlow 兼容格式）
interface WorkflowGraph {
  nodes: {
    id: string;
    type: string;
    data: { label: string; status: string; task_type: string; task_id: string; gate_id?: string; gate_type?: string };
    position: { x: number; y: number };
  }[];
  edges: { id: string; source: string; target: string }[];
}

// statusBorderColor 根据任务状态返回边框颜色（使用 api.ts 中的统一映射）
function statusBorderColor(status: string): string {
  return (statusHex[status] || statusHex.default).border;
}

// statusBgColor 根据任务状态返回背景颜色（使用 api.ts 中的统一映射）
function statusBgColor(status: string): string {
  return (statusHex[status] || statusHex.default).bg;
}

// TaskNode 是工作流图中的任务节点组件
function TaskNode({ data }: { data: { label: string; status: string; task_type: string } }) {
  const borderColor = statusBorderColor(data.status);
  const bgColor = statusBgColor(data.status);
  return (
    <div
      style={{
        padding: "10px 16px",
        borderRadius: 8,
        border: `2px solid ${borderColor}`,
        background: bgColor,
        minWidth: 140,
        fontSize: 12,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: borderColor }} />
      <div style={{ fontWeight: 600, marginBottom: 4 }}>{data.label}</div>
      <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <span
          style={{
            padding: "1px 6px",
            borderRadius: 4,
            background: borderColor,
            color: "#fff",
            fontSize: 10,
          }}
        >
          {data.status || "pending"}
        </span>
        <span style={{ color: "#64748b" }}>{data.task_type}</span>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ background: borderColor }} />
    </div>
  );
}

// gateStatusBorderColor 根据门禁状态返回边框颜色（使用语义色 Token）
function gateStatusBorderColor(status: string): string {
  switch (status) {
    case "passed":
      return "#22c55e"; // success-500
    case "failed":
      return "#ef4444"; // error-500
    case "pending":
      return "#eab308"; // warning-500
    default:
      return "#94a3b8"; // neutral-400
  }
}

// GateNode 是工作流图中的门禁节点组件（菱形样式）
function GateNode({ data }: { data: { label: string; status: string; gate_type: string; gate_id: string; onApprove?: (gateId: string) => void } }) {
  const borderColor = gateStatusBorderColor(data.status);
  return (
    <div
      style={{
        padding: "8px 14px",
        borderRadius: 4,
        border: `2px dashed ${borderColor}`,
        background: data.status === "passed" ? "#f0fdf4" : data.status === "failed" ? "#fef2f2" : "#fefce8", // success-50 / error-50 / warning-50
        minWidth: 120,
        fontSize: 11,
        transform: "rotate(0deg)",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: borderColor }} />
      <div style={{ fontWeight: 600, marginBottom: 2, display: "flex", alignItems: "center", gap: 4 }}>
        <span style={{ color: "#ca8a04" }}>&#9670;</span>
        {data.label}
      </div>
      <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
        <span
          style={{
            padding: "1px 5px",
            borderRadius: 3,
            background: borderColor,
            color: "#fff",
            fontSize: 9,
          }}
        >
          {data.status || "pending"}
        </span>
        <span style={{ color: "#64748b", fontSize: 9 }}>{data.gate_type}</span>
      </div>
      {data.gate_type === "manual" && data.status === "pending" && data.onApprove && (
        <button
          onClick={() => data.onApprove!(data.gate_id)}
          style={{
            marginTop: 4,
            padding: "2px 8px",
            fontSize: 10,
            background: "#2563eb", // primary-600
            color: "#fff",
            border: "none",
            borderRadius: 3,
            cursor: "pointer",
          }}
        >
          通过
        </button>
      )}
      <Handle type="source" position={Position.Bottom} style={{ background: borderColor }} />
    </div>
  );
}

// 注册自定义节点类型
const nodeTypes: NodeTypes = {
  task: TaskNode as any,
  gate: GateNode as any,
};

// RunDetailPage 运行详情页面组件
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

  // 解析异步参数获取运行 ID
  useEffect(() => {
    params.then((p) => setId(p.id));
  }, [params]);

  // 加载运行详情、任务列表、事件时间线和工作流图
  useEffect(() => {
    if (!id) return;
    apiFetch<Run>(`/api/runs/${id}`)
      .then(setRun)
      .catch((e) => setError(e.message));

    apiFetch<Task[]>(`/api/runs/${id}/tasks`)
      .then(setTasks)
      .catch(() => {});

    apiFetch<Event[]>(`/api/runs/${id}/timeline`)
      .then(setEvents)
      .catch(() => {});

    apiFetch<WorkflowGraph>(`/api/runs/${id}/workflow`)
      .then(setWorkflowGraph)
      .catch(() => {});

    listGates(id).then(setGates).catch(() => {});
  }, [id]);

  // handleRetry 重试失败或被驱逐的任务
  const handleRetry = async (taskId: string) => {
    try {
      await apiFetch(`/api/tasks/${taskId}/retry`, { method: "POST" });
      apiFetch<Task[]>(`/api/runs/${id}/tasks`).then(setTasks);
    } catch (e: any) {
      setError(e.message);
    }
  };

  // handleCancel 取消排队或运行中的任务
  const handleCancel = async (taskId: string) => {
    try {
      await apiFetch(`/api/tasks/${taskId}/cancel`, { method: "POST" });
      apiFetch<Task[]>(`/api/runs/${id}/tasks`).then(setTasks);
    } catch (e: any) {
      setError(e.message);
    }
  };

  // handleApproveGate 通过一个门禁
  const handleApproveGate = async (gateId: string) => {
    try {
      await approveGate(gateId, "user");
      listGates(id).then(setGates);
      // 刷新工作流图
      apiFetch<WorkflowGraph>(`/api/runs/${id}/workflow`).then(setWorkflowGraph).catch(() => {});
    } catch (e: any) {
      setError(e.message);
    }
  };

  if (!id) return <div className="p-6 text-neutral-500">加载中...</div>;

  if (error && !run) {
    return (
      <div className="p-6">
        <div role="alert" className="mb-4 p-3 bg-error-50 border border-error-500 rounded text-error-700 text-sm">{error}</div>
        <button onClick={() => router.back()} className="text-primary-600 hover:underline focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none rounded">
          返回
        </button>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* 页头：返回按钮、标题和状态 */}
      <div className="flex items-center gap-4 mb-6">
        <button onClick={() => router.back()} className="text-primary-600 hover:underline text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none rounded">
          ← 返回
        </button>
        <h1 className="text-2xl font-bold">{run?.title || `Run ${id.slice(0, 8)}`}</h1>
        {run && <StatusBadge status={run.status} />}
      </div>

      {error && (
        <div role="alert" className="mb-4 p-3 bg-error-50 border border-error-500 rounded text-error-700 text-sm">{error}</div>
      )}

      {/* 标签页切换 */}
      <div className="flex gap-2 mb-4">
        <button
          onClick={() => setActiveTab("tasks")}
          className={`px-4 py-2 rounded text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none ${
            activeTab === "tasks" ? "bg-primary-600 text-white" : "bg-neutral-100 text-neutral-700 hover:bg-neutral-200"
          }`}
        >
          任务列表 ({tasks.length})
        </button>
        <button
          onClick={() => setActiveTab("timeline")}
          className={`px-4 py-2 rounded text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none ${
            activeTab === "timeline" ? "bg-primary-600 text-white" : "bg-neutral-100 text-neutral-700 hover:bg-neutral-200"
          }`}
        >
          事件时间线 ({events.length})
        </button>
        <button
          onClick={() => setActiveTab("workflow")}
          className={`px-4 py-2 rounded text-sm focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none ${
            activeTab === "workflow" ? "bg-primary-600 text-white" : "bg-neutral-100 text-neutral-700 hover:bg-neutral-200"
          }`}
        >
          工作流
        </button>
      </div>

      {/* 任务列表标签页 */}
      {activeTab === "tasks" && (
        <div>
          {tasks.length === 0 ? (
            <p className="text-neutral-500">暂无任务</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="border-b border-neutral-200 text-left text-sm text-neutral-600">
                    <th className="pb-2">ID</th>
                    <th className="pb-2">标题</th>
                    <th className="pb-2">类型</th>
                    <th className="pb-2">状态</th>
                    <th className="pb-2">优先级</th>
                    <th className="pb-2">资源</th>
                    <th className="pb-2">尝试</th>
                    <th className="pb-2">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {tasks.map((task) => (
                    <tr key={task.id} className="border-b border-neutral-100 hover:bg-neutral-50">
                      <td className="py-2 text-sm font-mono">{task.id.slice(0, 8)}</td>
                      <td className="py-2 text-sm">{task.title}</td>
                      <td className="py-2 text-sm">{task.task_type}</td>
                      <td className="py-2 text-sm">
                        <StatusBadge status={task.status} />
                      </td>
                      <td className="py-2 text-sm">{task.priority}</td>
                      <td className="py-2 text-sm">{task.resource_class}</td>
                      <td className="py-2 text-sm">{task.attempt_no}</td>
                      <td className="py-2 text-sm space-x-2">
                        {(task.status === "failed" || task.status === "evicted") && (
                          <button
                            onClick={() => handleRetry(task.id)}
                            className="text-primary-600 hover:underline focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none rounded"
                          >
                            重试
                          </button>
                        )}
                        {(task.status === "queued" || task.status === "running") && (
                          <button
                            onClick={() => handleCancel(task.id)}
                            className="text-error-600 hover:underline focus-visible:ring-2 focus-visible:ring-error-500 focus-visible:outline-none rounded"
                          >
                            取消
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
      )}

      {/* 事件时间线标签页 */}
      {activeTab === "timeline" && (
        <div>
          {events.length === 0 ? (
            <p className="text-neutral-500">暂无事件</p>
          ) : (
            <div className="space-y-3">
              {events.map((event) => (
                <div key={event.id} className="flex items-start gap-3 p-3 bg-neutral-50 rounded-lg border border-neutral-200">
                  <div className="text-xs text-neutral-400 min-w-[140px] pt-0.5">
                    {new Date(event.created_at).toLocaleString()}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="px-1.5 py-0.5 bg-neutral-200 rounded text-xs">{event.event_type}</span>
                      <span className="text-sm">{event.message}</span>
                    </div>
                    {event.task_id && (
                      <div className="text-xs text-neutral-400 mt-1">
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

      {/* 工作流图标签页（ReactFlow） */}
      {activeTab === "workflow" && (
        <div>
          {!workflowGraph || workflowGraph.nodes.length === 0 ? (
            <p className="text-neutral-500">暂无工作流数据</p>
          ) : (
            <div style={{ width: "100%", height: 500, border: "1px solid #e2e8f0", borderRadius: 8 }}>
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
                  style: { stroke: "#94a3b8", strokeWidth: 2 }, // neutral-400
                }))}
                nodeTypes={nodeTypes}
                fitView
                minZoom={0.3}
                maxZoom={2}
              >
                <Background color="#e2e8f0" gap={16} />
                <Controls />
                <MiniMap
                  nodeColor={(node) => statusBorderColor((node.data as any)?.status || "")}
                  maskColor="rgba(0,0,0,0.1)"
                />
              </ReactFlow>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
