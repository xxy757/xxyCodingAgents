// page.tsx - 系统仪表盘
// Bento 布局：资源指标 + 压力状态 + 服务健康 + 快速入口。
"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, type ResourceSnapshot } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import {
  Cpu,
  HardDrive,
  Memory,
  Gauge,
  ActivityIcon as Activity,
  Lightning,
  ArrowRight,
  Pulse,
} from "@phosphor-icons/react/dist/ssr";

interface HealthStatus {
  status: string;
}

const QUICK_TASKS = [
  { type: "bugfix", label: "修复 Bug" },
  { type: "build", label: "创建 API" },
  { type: "qa", label: "添加测试" },
  { type: "review", label: "代码审查" },
  { type: "docs", label: "写文档" },
];

export default function DashboardPage() {
  const router = useRouter();
  const [metrics, setMetrics] = useState<ResourceSnapshot | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [ready, setReady] = useState<HealthStatus | null>(null);
  const [quickInput, setQuickInput] = useState("");

  useEffect(() => {
    const fetch_ = async () => {
      const [m, h, r] = await Promise.allSettled([
        apiFetch<ResourceSnapshot>("/api/system/metrics"),
        apiFetch<HealthStatus>("/healthz"),
        apiFetch<HealthStatus>("/readyz"),
      ]);
      if (m.status === "fulfilled") setMetrics(m.value);
      if (h.status === "fulfilled") setHealth(h.value);
      if (r.status === "fulfilled") setReady(r.value);
    };
    fetch_();
    const t = setInterval(fetch_, 5000);
    return () => clearInterval(t);
  }, []);

  const handleSubmit = (type?: string) => {
    const p = new URLSearchParams();
    if (quickInput.trim()) p.set("input", quickInput.trim());
    if (type) p.set("type", type);
    router.push(`/prompt-drafts?${p.toString()}`);
  };

  return (
    <div className="space-y-8 animate-fade-up">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-zinc-900">
          仪表盘
        </h1>
        <p className="text-sm text-zinc-500 mt-1">
          系统资源与服务状态概览
        </p>
      </div>

      {/* Quick Input */}
      <div className="card-bezel p-6">
        <div className="flex items-center gap-2 mb-4">
          <Lightning weight="fill" className="w-4 h-4 text-accent-600" />
          <span className="text-sm font-medium text-zinc-700">快速任务</span>
        </div>
        <div className="flex gap-3">
          <input
            type="text"
            value={quickInput}
            onChange={(e) => setQuickInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
            placeholder="描述你想完成的任务..."
            className="flex-1 bg-zinc-50 border border-zinc-200 rounded-xl px-4 py-3 text-sm
                       placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-accent-500/20
                       focus:border-accent-500 transition-all duration-200"
          />
          <button
            onClick={() => handleSubmit()}
            className="px-5 py-3 bg-zinc-900 text-white rounded-xl text-sm font-medium
                       hover:bg-zinc-800 pressable transition-colors duration-200
                       flex items-center gap-2"
          >
            优化提示词
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>
        <div className="mt-3 flex gap-2 flex-wrap">
          {QUICK_TASKS.map((t) => (
            <button
              key={t.type}
              onClick={() => handleSubmit(t.type)}
              className="px-3.5 py-1.5 text-xs font-medium bg-zinc-100 text-zinc-600
                         rounded-full hover:bg-zinc-200 hover:text-zinc-800
                         transition-colors duration-200 pressable border border-zinc-200/60"
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* Metrics Bento Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 stagger">
        <MetricCard
          icon={Activity}
          label="活跃 Agent"
          value={metrics?.active_agents ?? 0}
          unit="个"
          loading={!metrics}
        />
        <MetricCard
          icon={Memory}
          label="内存使用"
          value={metrics?.memory_percent ?? 0}
          unit="%"
          loading={!metrics}
          bar={metrics?.memory_percent}
        />
        <MetricCard
          icon={Cpu}
          label="CPU 使用"
          value={metrics?.cpu_percent ?? 0}
          unit="%"
          loading={!metrics}
          bar={metrics?.cpu_percent}
        />
        <MetricCard
          icon={HardDrive}
          label="磁盘使用"
          value={metrics?.disk_percent ?? 0}
          unit="%"
          loading={!metrics}
          bar={metrics?.disk_percent}
        />
      </div>

      {/* Bottom Row: Pressure + Services */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Pressure */}
        <div className="card-bezel p-6">
          <div className="flex items-center gap-2 mb-4">
            <Gauge className="w-4 h-4 text-zinc-500" />
            <span className="text-sm font-medium text-zinc-700">压力等级</span>
          </div>
          <div className="flex items-baseline gap-2">
            <span className={`text-3xl font-semibold tracking-tight ${
              metrics?.pressure_level === "normal" ? "text-emerald-600" :
              metrics?.pressure_level === "warn" ? "text-amber-600" : "text-red-600"
            }`}>
              {(metrics?.pressure_level ?? "normal").toUpperCase()}
            </span>
          </div>
          <div className="mt-3 h-1.5 rounded-full bg-zinc-100 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-700 ease-[cubic-bezier(0.16,1,0.3,1)] ${
                metrics?.pressure_level === "normal" ? "bg-emerald-500" :
                metrics?.pressure_level === "warn" ? "bg-amber-500" : "bg-red-500"
              }`}
              style={{ width: `${Math.min((metrics?.memory_percent ?? 0), 100)}%` }}
            />
          </div>
        </div>

        {/* Services */}
        <div className="card-bezel p-6 lg:col-span-2">
          <div className="flex items-center gap-2 mb-4">
            <Pulse className="w-4 h-4 text-zinc-500" />
            <span className="text-sm font-medium text-zinc-700">服务状态</span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <ServiceItem
              label="后端 API"
              status={health?.status === "ok" ? "running" : "unknown"}
            />
            <ServiceItem
              label="数据库"
              status={ready?.status === "ready" ? "running" : "unknown"}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  icon: Icon,
  label,
  value,
  unit,
  loading,
  bar,
}: {
  icon: React.ElementType;
  label: string;
  value: number;
  unit: string;
  loading: boolean;
  bar?: number;
}) {
  return (
    <div className="card-bezel p-5">
      {loading ? (
        <>
          <div className="skeleton h-3 w-16 mb-3" />
          <div className="skeleton h-7 w-20 mb-2" />
          <div className="skeleton h-1 w-full" />
        </>
      ) : (
        <>
          <div className="flex items-center gap-2 mb-2">
            <Icon className="w-3.5 h-3.5 text-zinc-400" />
            <span className="text-xs text-zinc-500 font-medium">{label}</span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-semibold tracking-tight text-zinc-900">
              {typeof value === "number" ? value.toFixed(bar !== undefined ? 1 : 0) : value}
            </span>
            <span className="text-sm text-zinc-400">{unit}</span>
          </div>
          {bar !== undefined && (
            <div className="mt-3 h-1 rounded-full bg-zinc-100 overflow-hidden">
              <div
                className="h-full rounded-full bg-accent-500 transition-all duration-700 ease-[cubic-bezier(0.16,1,0.3,1)]"
                style={{ width: `${Math.min(bar, 100)}%` }}
              />
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ServiceItem({ label, status }: { label: string; status: string }) {
  return (
    <div className="flex items-center justify-between py-2.5 px-3 rounded-lg bg-zinc-50 border border-zinc-100">
      <span className="text-sm text-zinc-700">{label}</span>
      <StatusBadge status={status} />
    </div>
  );
}
