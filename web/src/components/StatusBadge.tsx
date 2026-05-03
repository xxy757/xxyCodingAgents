// StatusBadge.tsx - 状态徽章组件
// 根据状态值显示不同颜色的状态标签，附带图标辅助色盲用户识别。
import { statusColors } from "@/lib/api";

// 状态图标映射（辅助色盲用户，不依赖颜色区分）
const statusIcons: Record<string, string> = {
  pending: "○",
  queued: "◌",
  admitted: "◉",
  running: "▶",
  completed: "✓",
  failed: "✗",
  cancelled: "⊘",
  evicted: "▽",
  paused: "❚❚",
  stopped: "■",
  starting: "↻",
  recoverable: "⚠",
  orphaned: "⚡",
  active: "▶",
  detached: "◇",
  closed: "⊗",
  unknown: "?",
};

// StatusBadge 根据状态字符串渲染对应的彩色徽章
export function StatusBadge({ status }: { status: string }) {
  const colorClass = statusColors[status] || "bg-neutral-100 text-neutral-800";
  const icon = statusIcons[status] || "●";
  return (
    <span
      className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium ${colorClass}`}
      role="status"
      aria-label={`状态: ${status}`}
    >
      <span aria-hidden="true">{icon}</span>
      {status}
    </span>
  );
}
