// page.tsx - 系统仪表盘首页
// 展示系统资源指标（Agent 数量、内存、CPU、磁盘、压力等级）和服务状态。
// 每 5 秒自动刷新数据。顶部集成 Prompt Composer 快速入口。
"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, type ResourceSnapshot } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";

// HealthStatus 健康状态接口
interface HealthStatus {
  status: string;
}

// 快捷任务类型配置
const QUICK_TASKS = [
  { type: "bugfix", label: "修复Bug", emoji: "🐛" },
  { type: "build", label: "创建API", emoji: "⚡" },
  { type: "qa", label: "添加测试", emoji: "🧪" },
  { type: "build", label: "代码重构", emoji: "🔧" },
  { type: "docs", label: "写文档", emoji: "📝" },
  { type: "review", label: "代码审查", emoji: "👀" },
];

// DashboardPage 仪表盘首页组件
export default function DashboardPage() {
  const router = useRouter();
  const [metrics, setMetrics] = useState<ResourceSnapshot | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [ready, setReady] = useState<HealthStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [quickInput, setQuickInput] = useState("");

  // 并行获取指标和健康状态，每 5 秒刷新一次
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [m, h, r] = await Promise.allSettled([
          apiFetch<ResourceSnapshot>("/api/system/metrics"),
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
      color: "bg-primary-50 text-primary-700",
    },
    {
      label: "内存使用",
      value: metrics ? `${metrics.memory_percent.toFixed(1)}%` : "-",
      color: "bg-info-50 text-info-600",
    },
    {
      label: "CPU 使用",
      value: metrics ? `${metrics.cpu_percent.toFixed(1)}%` : "-",
      color: "bg-warning-50 text-warning-600",
    },
    {
      label: "磁盘使用",
      value: metrics ? `${metrics.disk_percent.toFixed(1)}%` : "-",
      color: "bg-error-50 text-error-600",
    },
    {
      label: "压力等级",
      value: metrics?.pressure_level ?? "-",
      color:
        metrics?.pressure_level === "normal"
          ? "bg-success-50 text-success-700"
          : metrics?.pressure_level === "warn"
          ? "bg-warning-50 text-warning-600"
          : metrics?.pressure_level
          ? "bg-error-50 text-error-700"
          : "bg-neutral-50 text-neutral-700",
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

  // handleQuickSubmit 快速输入跳转到草稿页面
  const handleQuickSubmit = (type?: string) => {
    const params = new URLSearchParams();
    if (quickInput.trim()) params.set("input", quickInput.trim());
    if (type) params.set("type", type);
    router.push(`/prompt-drafts?${params.toString()}`);
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">系统仪表盘</h1>

      {/* Prompt Composer 快速入口 */}
      <div className="bg-gradient-to-r from-primary-50 to-primary-100 rounded-lg p-5 mb-6 border border-primary-200">
        <div className="flex gap-3">
          <input
            type="text"
            value={quickInput}
            onChange={(e) => setQuickInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleQuickSubmit()}
            placeholder="今天想让我帮你做什么？"
            aria-label="快速任务输入"
            className="flex-1 border border-neutral-300 rounded-lg px-4 py-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400"
          />
          <button
            onClick={() => handleQuickSubmit()}
            className="px-5 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 text-sm font-medium focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
          >
            优化提示词
          </button>
        </div>
        <div className="mt-3 flex gap-2 flex-wrap" role="group" aria-label="快捷任务类型">
          {QUICK_TASKS.map((t, i) => (
            <button
              key={i}
              onClick={() => handleQuickSubmit(t.type)}
              aria-label={t.label}
              className="px-4 py-2 text-sm bg-white border border-neutral-200 rounded-full hover:bg-primary-50 hover:border-primary-300 transition-colors focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none"
            >
              <span aria-hidden="true">{t.emoji}</span> {t.label}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div role="alert" className="mb-4 p-3 bg-error-50 border border-error-500 rounded text-error-700 text-sm">
          加载失败：{error}。请检查后端服务是否运行。
        </div>
      )}

      {/* 指标卡片网格 */}
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
        {metrics === null ? (
          Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="p-4 rounded-lg bg-neutral-50">
              <div className="skeleton h-4 w-16 mb-2" />
              <div className="skeleton h-8 w-20" />
            </div>
          ))
        ) : (
          cards.map((card) => (
            <div key={card.label} className={`p-4 rounded-lg ${card.color}`}>
              <div className="text-sm opacity-75">{card.label}</div>
              <div className="text-2xl font-bold mt-1">{card.value}</div>
            </div>
          ))
        )}
      </div>

      {/* 服务状态 */}
      <h2 className="text-lg font-semibold mb-4">服务状态</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {serviceCards.map((card) => (
          <div key={card.label} className="p-4 bg-white border border-neutral-200 rounded-lg flex items-center justify-between">
            <span className="font-medium">{card.label}</span>
            <StatusBadge status={card.status} />
          </div>
        ))}
      </div>
    </div>
  );
}
