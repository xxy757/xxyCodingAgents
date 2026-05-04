// StatusBadge.tsx - 状态徽章
// Phosphor 图标 + 语义色 + 无障碍。
import {
  Circle,
  CheckCircle,
  XCircle,
  WarningCircle,
  Clock,
  Spinner,
  Prohibit,
  ArrowSquareOut,
} from "@phosphor-icons/react/dist/ssr";

const statusConfig: Record<string, { icon: React.ElementType; label: string; className: string }> = {
  pending:     { icon: Clock,          label: "等待中", className: "bg-zinc-100 text-zinc-600 border-zinc-200" },
  queued:      { icon: Clock,          label: "队列中", className: "bg-amber-50 text-amber-700 border-amber-200/60" },
  admitted:    { icon: Circle,         label: "已准入", className: "bg-blue-50 text-blue-700 border-blue-200/60" },
  running:     { icon: Spinner,        label: "运行中", className: "bg-emerald-50 text-emerald-700 border-emerald-200/60" },
  completed:   { icon: CheckCircle,    label: "已完成", className: "bg-emerald-50 text-emerald-700 border-emerald-200/60" },
  failed:      { icon: XCircle,        label: "失败",   className: "bg-red-50 text-red-700 border-red-200/60" },
  cancelled:   { icon: Prohibit,       label: "已取消", className: "bg-zinc-100 text-zinc-500 border-zinc-200" },
  evicted:     { icon: ArrowSquareOut, label: "已驱逐", className: "bg-amber-50 text-amber-700 border-amber-200/60" },
  paused:      { icon: Circle,         label: "已暂停", className: "bg-amber-50 text-amber-700 border-amber-200/60" },
  stopped:     { icon: Prohibit,       label: "已停止", className: "bg-zinc-100 text-zinc-500 border-zinc-200" },
  starting:    { icon: Spinner,        label: "启动中", className: "bg-blue-50 text-blue-700 border-blue-200/60" },
  recoverable: { icon: WarningCircle,  label: "可恢复", className: "bg-amber-50 text-amber-700 border-amber-200/60" },
  orphaned:    { icon: WarningCircle,  label: "孤立",   className: "bg-red-50 text-red-700 border-red-200/60" },
  active:      { icon: CheckCircle,    label: "活跃",   className: "bg-emerald-50 text-emerald-700 border-emerald-200/60" },
  detached:    { icon: Circle,         label: "已分离", className: "bg-zinc-100 text-zinc-500 border-zinc-200" },
  closed:      { icon: Prohibit,       label: "已关闭", className: "bg-zinc-100 text-zinc-500 border-zinc-200" },
  draft:       { icon: Circle,         label: "草稿",   className: "bg-amber-50 text-amber-700 border-amber-200/60" },
  sent:        { icon: CheckCircle,    label: "已发送", className: "bg-emerald-50 text-emerald-700 border-emerald-200/60" },
  passed:      { icon: CheckCircle,    label: "通过",   className: "bg-emerald-50 text-emerald-700 border-emerald-200/60" },
  skipped:     { icon: Circle,         label: "跳过",   className: "bg-zinc-100 text-zinc-500 border-zinc-200" },
};

export function StatusBadge({ status }: { status: string }) {
  const config = statusConfig[status] || {
    icon: Circle,
    label: status,
    className: "bg-zinc-100 text-zinc-600 border-zinc-200",
  };
  const Icon = config.icon;

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${config.className}`}
      role="status"
      aria-label={`状态: ${config.label}`}
    >
      <Icon
        weight="fill"
        className={`w-3 h-3 ${status === "running" || status === "starting" ? "animate-spin" : ""}`}
      />
      {config.label}
    </span>
  );
}
