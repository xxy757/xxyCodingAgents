// runs/[id]/page.tsx - 运行详情页面
// 展示单个运行的任务列表、事件时间线和工作流图（ReactFlow）。
// 支持任务重试和取消操作。
"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
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

// Task 任务数据接口
interface Task {
  id: string;
  run_id: string;
  task_type: string;
  title: string;
  status: string;
  priority: string;
  resource_class: string;
  attempt_no: number;
  created_at: string;
}

// Event 事件数据接口
interface Event {
  id: string;
  run_id: string;
  task_id?: string;
  agent_id?: string;
  event_type: string;
  message: string;
  created_at: string;
}

// Run 运行数据接口
interface Run {
  id: string;
  project_id: string;
  title: string;
  status: string;
  created_at: string;
}

// WorkflowGraph 工作流图数据接口（ReactFlow 兼容格式）
interface WorkflowGraph {
  nodes: {
    id: string;
    type: string;
    data: { label: string; status: string; task_type: string; task_id: string };
    position: { x: number; y: number };
  }[];
  edges: { id: string; source: string; target: string }[];
}

// statusBorderColor 根据任务状态返回边框颜色
function statusBorderColor(status: string): string {
  switch (status) {
    case "running":
      return "#22c55e";
    case "completed":
      return "#3b82f6";
    case "failed":
      return "#ef4444";
    case "queued":
      return "#eab308";
    case "blocked":
      return "#a855f7";
    case "evicted":
      return "#f97316";
    default:
      return "#9ca3af";
  }
}

// statusBgColor 根据任务状态返回背景颜色
function statusBgColor(status: string): string {
  switch (status) {
    case "running":
      return "#f0fdf4";
    case "completed":
      return "#eff6ff";
    case "failed":
      return "#fef2f2";
    case "queued":
      return "#fefce8";
    case "blocked":
      return "#faf5ff";
    case "evicted":
      return "#fff7ed";
    default:
      return "#f9fafb";
  }
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
        <span style={{ color: "#6b7280" }}>{data.task_type}</span>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ background: borderColor }} />
    </div>
  );
}

// 注册自定义节点类型
const nodeTypes: NodeTypes = {
  task: TaskNode as any,
};

// RunDetailPage 运行详情页面组件
export default function RunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState<string>("");
  const [run, setRun] = useState<Run | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
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

  // taskStatusColor 根据任务状态返回表格中的样式类名
  const taskStatusColor = (status: string) => {
    switch (status) {
      case "running":
        return "bg-green-100 text-green-700";
      case "queued":
        return "bg-yellow-100 text-yellow-700";
      case "completed":
        return "bg-blue-100 text-blue-700";
      case "failed":
        return "bg-red-100 text-red-700";
      case "blocked":
        return "bg-purple-100 text-purple-700";
      case "evicted":
        return "bg-orange-100 text-orange-700";
      default:
        return "bg-gray-100 text-gray-700";
    }
  };

  if (!id) return <div className="p-6 text-gray-500">加载中...</div>;

  if (error && !run) {
    return (
      <div className="p-6">
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
        <button onClick={() => router.back()} className="text-blue-600 hover:underline">
          返回
        </button>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* 页头：返回按钮、标题和状态 */}
      <div className="flex items-center gap-4 mb-6">
        <button onClick={() => router.back()} className="text-blue-600 hover:underline text-sm">
          ← 返回
        </button>
        <h1 className="text-2xl font-bold">{run?.title || `Run ${id.slice(0, 8)}`}</h1>
        {run && (
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${taskStatusColor(run.status)}`}>
            {run.status}
          </span>
        )}
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {/* 标签页切换 */}
      <div className="flex gap-2 mb-4">
        <button
          onClick={() => setActiveTab("tasks")}
          className={`px-4 py-2 rounded text-sm ${
            activeTab === "tasks" ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700 hover:bg-gray-200"
          }`}
        >
          任务列表 ({tasks.length})
        </button>
        <button
          onClick={() => setActiveTab("timeline")}
          className={`px-4 py-2 rounded text-sm ${
            activeTab === "timeline" ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700 hover:bg-gray-200"
          }`}
        >
          事件时间线 ({events.length})
        </button>
        <button
          onClick={() => setActiveTab("workflow")}
          className={`px-4 py-2 rounded text-sm ${
            activeTab === "workflow" ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700 hover:bg-gray-200"
          }`}
        >
          工作流
        </button>
      </div>

      {/* 任务列表标签页 */}
      {activeTab === "tasks" && (
        <div>
          {tasks.length === 0 ? (
            <p className="text-gray-500">暂无任务</p>
          ) : (
            <table className="w-full border-collapse">
              <thead>
                <tr className="border-b text-left text-sm text-gray-600">
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
                  <tr key={task.id} className="border-b hover:bg-gray-50">
                    <td className="py-2 text-sm font-mono">{task.id.slice(0, 8)}</td>
                    <td className="py-2 text-sm">{task.title}</td>
                    <td className="py-2 text-sm">{task.task_type}</td>
                    <td className="py-2 text-sm">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${taskStatusColor(task.status)}`}>
                        {task.status}
                      </span>
                    </td>
                    <td className="py-2 text-sm">{task.priority}</td>
                    <td className="py-2 text-sm">{task.resource_class}</td>
                    <td className="py-2 text-sm">{task.attempt_no}</td>
                    <td className="py-2 text-sm space-x-2">
                      {(task.status === "failed" || task.status === "evicted") && (
                        <button
                          onClick={() => handleRetry(task.id)}
                          className="text-blue-600 hover:underline"
                        >
                          重试
                        </button>
                      )}
                      {(task.status === "queued" || task.status === "running") && (
                        <button
                          onClick={() => handleCancel(task.id)}
                          className="text-red-600 hover:underline"
                        >
                          取消
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* 事件时间线标签页 */}
      {activeTab === "timeline" && (
        <div>
          {events.length === 0 ? (
            <p className="text-gray-500">暂无事件</p>
          ) : (
            <div className="space-y-3">
              {events.map((event) => (
                <div key={event.id} className="flex items-start gap-3 p-3 bg-gray-50 rounded">
                  <div className="text-xs text-gray-400 min-w-[140px] pt-0.5">
                    {new Date(event.created_at).toLocaleString()}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="px-1.5 py-0.5 bg-gray-200 rounded text-xs">{event.event_type}</span>
                      <span className="text-sm">{event.message}</span>
                    </div>
                    {event.task_id && (
                      <div className="text-xs text-gray-400 mt-1">
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
            <p className="text-gray-500">暂无工作流数据</p>
          ) : (
            <div style={{ width: "100%", height: 500, border: "1px solid #e5e7eb", borderRadius: 8 }}>
              <ReactFlow
                nodes={workflowGraph.nodes.map((n) => ({
                  id: n.id,
                  type: "task",
                  data: n.data,
                  position: { x: n.position.x, y: n.position.y },
                }))}
                edges={workflowGraph.edges.map((e) => ({
                  id: e.id,
                  source: e.source,
                  target: e.target,
                  animated: true,
                  style: { stroke: "#94a3b8", strokeWidth: 2 },
                }))}
                nodeTypes={nodeTypes}
                fitView
                minZoom={0.3}
                maxZoom={2}
              >
                <Background color="#e5e7eb" gap={16} />
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
