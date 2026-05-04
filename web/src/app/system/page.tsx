// system/page.tsx - 系统监控
// 资源指标卡片 + 压力状态 + tmux 会话 + 配置参数。
"use client";

import { useState, useEffect } from "react";
import { apiFetch, type ResourceSnapshot } from "@/lib/api";
import {
  Cpu,
  HardDrive,
  Memory,
  Gauge,
  Terminal,
  GearSix,
  ActivityIcon as Activity,
} from "@phosphor-icons/react/dist/ssr";

export default function SystemPage() {
  const [metrics, setMetrics] = useState<ResourceSnapshot | null>(null);
  const [diagnostics, setDiagnostics] = useState<any>(null);

  useEffect(() => {
    apiFetch<ResourceSnapshot>("/api/system/metrics").then(setMetrics).catch(() => {});
    apiFetch<any>("/api/system/diagnostics").then(setDiagnostics).catch(() => {});
    const t = setInterval(() => {
      apiFetch<ResourceSnapshot>("/api/system/metrics").then(setMetrics).catch(() => {});
    }, 3000);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="space-y-6 animate-fade-up">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">系统监控</h1>
        <p className="text-sm text-zinc-500 mt-1">资源使用与服务状态</p>
      </div>

      {/* 资源指标 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 stagger">
        <GaugeCard
          icon={Memory}
          label="内存使用率"
          value={metrics?.memory_percent ?? 0}
          loading={!metrics}
        />
        <GaugeCard
          icon={Cpu}
          label="CPU 使用率"
          value={metrics?.cpu_percent ?? 0}
          loading={!metrics}
        />
        <GaugeCard
          icon={HardDrive}
          label="磁盘使用率"
          value={metrics?.disk_percent ?? 0}
          loading={!metrics}
        />
      </div>

      {/* 压力 + tmux + 配置 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* 压力状态 */}
        <div className="card-bezel p-6">
          <div className="flex items-center gap-2 mb-5">
            <Gauge className="w-4 h-4 text-zinc-500" />
            <span className="text-sm font-medium text-zinc-700">压力状态</span>
          </div>
          <div className="space-y-4">
            <div>
              <p className="text-xs text-zinc-400 mb-1">当前等级</p>
              <span className={`text-2xl font-semibold tracking-tight ${
                metrics?.pressure_level === "normal" ? "text-emerald-600" :
                metrics?.pressure_level === "warn" ? "text-amber-600" : "text-red-600"
              }`}>
                {(metrics?.pressure_level ?? "normal").toUpperCase()}
              </span>
            </div>
            <div>
              <p className="text-xs text-zinc-400 mb-1">活跃 Agent</p>
              <span className="text-2xl font-semibold tracking-tight text-zinc-900">
                {metrics?.active_agents ?? 0}
              </span>
            </div>
          </div>
        </div>

        {/* tmux 会话 */}
        <div className="card-bezel p-6">
          <div className="flex items-center gap-2 mb-5">
            <Terminal className="w-4 h-4 text-zinc-500" />
            <span className="text-sm font-medium text-zinc-700">tmux 会话</span>
          </div>
          <div className="bg-zinc-50 rounded-xl p-4 border border-zinc-100">
            <pre className="text-xs text-zinc-600 font-mono whitespace-pre-wrap break-all leading-relaxed">
              {diagnostics?.tmux_sessions || "无活跃会话"}
            </pre>
          </div>
        </div>

        {/* 配置参数 */}
        <div className="card-bezel p-6">
          <div className="flex items-center gap-2 mb-5">
            <GearSix className="w-4 h-4 text-zinc-500" />
            <span className="text-sm font-medium text-zinc-700">调度配置</span>
          </div>
          <div className="space-y-3">
            <ConfigRow label="最大并发 Agent" value={diagnostics?.config?.max_concurrent_agents ?? "-"} />
            <ConfigRow label="最大重型任务" value={diagnostics?.config?.max_heavy_agents ?? "-"} />
          </div>
        </div>
      </div>
    </div>
  );
}

function GaugeCard({
  icon: Icon,
  label,
  value,
  loading,
}: {
  icon: React.ElementType;
  label: string;
  value: number;
  loading: boolean;
}) {
  const pct = Math.min(value, 100);
  const color = pct > 88 ? "bg-red-500" : pct > 70 ? "bg-amber-500" : "bg-accent-500";

  return (
    <div className="card-bezel p-6">
      {loading ? (
        <>
          <div className="skeleton h-3 w-20 mb-3" />
          <div className="skeleton h-8 w-16 mb-4" />
          <div className="skeleton h-1.5 w-full rounded-full" />
        </>
      ) : (
        <>
          <div className="flex items-center gap-2 mb-2">
            <Icon className="w-3.5 h-3.5 text-zinc-400" />
            <span className="text-xs text-zinc-500 font-medium">{label}</span>
          </div>
          <div className="flex items-baseline gap-1 mb-4">
            <span className="text-3xl font-semibold tracking-tight text-zinc-900">
              {value.toFixed(1)}
            </span>
            <span className="text-sm text-zinc-400">%</span>
          </div>
          <div className="h-1.5 rounded-full bg-zinc-100 overflow-hidden">
            <div
              className={`h-full rounded-full ${color} transition-all duration-700 ease-[cubic-bezier(0.16,1,0.3,1)]`}
              style={{ width: `${pct}%` }}
            />
          </div>
        </>
      )}
    </div>
  );
}

function ConfigRow({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-zinc-100 last:border-0">
      <span className="text-sm text-zinc-500">{label}</span>
      <span className="text-sm font-semibold text-zinc-900">{value}</span>
    </div>
  );
}
