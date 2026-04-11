import { statusColors } from "@/lib/api";

export function StatusBadge({ status }: { status: string }) {
  const colorClass = statusColors[status] || "bg-gray-100 text-gray-800";
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colorClass}`}>
      {status}
    </span>
  );
}
