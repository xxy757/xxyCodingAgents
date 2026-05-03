// system/page.tsx - 系统监控页面
// 展示系统资源使用率、压力状态、tmux 会话和配置参数。
// 每 3 秒自动刷新指标数据。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, ResourceSnapshot, pressureColors } from "@/lib/api";

// SystemPage 系统监控页面组件
export default function SystemPage() {
  const [metrics, setMetrics] = useState<ResourceSnapshot | null>(null);
  const [diagnostics, setDiagnostics] = useState<any>(null);

  // 加载系统指标和诊断信息，每 3 秒刷新指标
  useEffect(() => {
    apiFetch<ResourceSnapshot>("/api/system/metrics").then(setMetrics).catch(console.error);
    apiFetch<any>("/api/system/diagnostics").then(setDiagnostics).catch(console.error);
    const interval = setInterval(() => {
      apiFetch<ResourceSnapshot>("/api/system/metrics").then(setMetrics).catch(() => {});
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold text-gray-900 mb-6">系统监控</h2>

      {/* 资源使用率卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <MetricCard
          title="内存使用率"
          value={metrics ? `${metrics.memory_percent.toFixed(1)}%` : "-"}
          bar={metrics?.memory_percent || 0}
          color="blue"
        />
        <MetricCard
          title="CPU 使用率"
          value={metrics ? `${metrics.cpu_percent.toFixed(1)}%` : "-"}
          bar={metrics?.cpu_percent || 0}
          color="green"
        />
        <MetricCard
          title="磁盘使用率"
          value={metrics ? `${metrics.disk_percent.toFixed(1)}%` : "-"}
          bar={metrics?.disk_percent || 0}
          color="purple"
        />
      </div>

      {/* 压力状态和 tmux 会话 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold mb-4">压力状态</h3>
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">当前压力等级</span>
              <span className={`text-lg font-bold ${pressureColors[metrics?.pressure_level || "normal"]}`}>
                {(metrics?.pressure_level || "normal").toUpperCase()}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600">活跃 Agent 数</span>
              <span className="text-lg font-bold">{metrics?.active_agents ?? 0}</span>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold mb-4">tmux 会话</h3>
          <pre className="text-sm text-gray-600 bg-gray-50 p-3 rounded overflow-x-auto">
            {diagnostics?.tmux_sessions || "无活跃会话"}
          </pre>
        </div>
      </div>

      {/* 调度器配置参数 */}
      <div className="mt-6 bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">配置参数</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <ConfigItem label="最大并发 Agent" value={diagnostics?.config?.max_concurrent_agents ?? "-"} />
          <ConfigItem label="最大重型任务" value={diagnostics?.config?.max_heavy_agents ?? "-"} />
        </div>
      </div>
    </div>
  );
}

// MetricCard 带进度条的资源指标卡片
function MetricCard({ title, value, bar, color }: { title: string; value: string; bar: number; color: string }) {
  const barColors: Record<string, string> = { blue: "bg-blue-500", green: "bg-green-500", purple: "bg-purple-500" };
  return (
    <div className="bg-white rounded-lg shadow p-6">
      <p className="text-sm text-gray-600 mb-1">{title}</p>
      <p className="text-3xl font-bold mb-3">{value}</p>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div className={`h-2 rounded-full ${barColors[color]}`} style={{ width: `${Math.min(bar, 100)}%` }} />
      </div>
    </div>
  );
}

// ConfigItem 配置参数展示项
function ConfigItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-gray-500">{label}</p>
      <p className="font-medium">{value}</p>
    </div>
  );
}
