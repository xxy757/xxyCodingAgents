// page.tsx - 系统仪表盘首页
// 展示系统资源指标（Agent 数量、内存、CPU、磁盘、压力等级）和服务状态。
// 每 5 秒自动刷新数据。
"use client";

import { useState, useEffect } from "react";
import { apiFetch } from "@/lib/api";

// Metrics 系统资源指标接口
interface Metrics {
  memory_percent: number;
  cpu_percent: number;
  disk_percent: number;
  active_agents: number;
  pressure_level: string;
}

// HealthStatus 健康状态接口
interface HealthStatus {
  status: string;
}

// DashboardPage 仪表盘首页组件
export default function DashboardPage() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [ready, setReady] = useState<HealthStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  // 并行获取指标和健康状态，每 5 秒刷新一次
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [m, h, r] = await Promise.allSettled([
          apiFetch<Metrics>("/api/system/metrics"),
          apiFetch<HealthStatus>("/healthz"),
          apiFetch<HealthStatus>("/readyz"),
        ]);
        if (m.status === "fulfilled") setMetrics(m.value);
        if (h.status === "fulfilled") setHealth(h.value);
        if (r.status === "fulfilled") setReady(r.value);
      } catch (e: any) {
        setError(e.message);
      }
    };
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  // 指标卡片配置
  const cards = [
    {
      label: "活跃 Agent",
      value: metrics?.active_agents ?? "-",
      color: "bg-blue-50 text-blue-700",
    },
    {
      label: "内存使用",
      value: metrics ? `${metrics.memory_percent.toFixed(1)}%` : "-",
      color: "bg-purple-50 text-purple-700",
    },
    {
      label: "CPU 使用",
      value: metrics ? `${metrics.cpu_percent.toFixed(1)}%` : "-",
      color: "bg-orange-50 text-orange-700",
    },
    {
      label: "磁盘使用",
      value: metrics ? `${metrics.disk_percent.toFixed(1)}%` : "-",
      color: "bg-pink-50 text-pink-700",
    },
    {
      label: "压力等级",
      value: metrics?.pressure_level ?? "-",
      color:
        metrics?.pressure_level === "normal"
          ? "bg-green-50 text-green-700"
          : metrics?.pressure_level === "warn"
          ? "bg-yellow-50 text-yellow-700"
          : metrics?.pressure_level
          ? "bg-red-50 text-red-700"
          : "bg-gray-50 text-gray-700",
    },
  ];

  // 服务状态卡片配置
  const serviceCards = [
    {
      label: "后端服务",
      status: health?.status === "ok" ? "running" : "unknown",
    },
    {
      label: "数据库",
      status: ready?.status === "ready" ? "running" : "unknown",
    },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">系统仪表盘</h1>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
      )}

      {/* 指标卡片网格 */}
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
        {cards.map((card) => (
          <div key={card.label} className={`p-4 rounded-lg ${card.color}`}>
            <div className="text-sm opacity-75">{card.label}</div>
            <div className="text-2xl font-bold mt-1">{card.value}</div>
          </div>
        ))}
      </div>

      {/* 服务状态 */}
      <h2 className="text-lg font-semibold mb-4">服务状态</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {serviceCards.map((card) => (
          <div key={card.label} className="p-4 bg-white border rounded-lg flex items-center justify-between">
            <span className="font-medium">{card.label}</span>
            <span
              className={`px-2 py-0.5 rounded text-xs font-medium ${
                card.status === "running"
                  ? "bg-green-100 text-green-700"
                  : "bg-gray-100 text-gray-700"
              }`}
            >
              {card.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
