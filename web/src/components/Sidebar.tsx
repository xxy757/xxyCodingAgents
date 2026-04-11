"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { href: "/", label: "仪表盘", icon: "▦" },
  { href: "/projects", label: "项目", icon: "▣" },
  { href: "/runs", label: "运行", icon: "▶" },
  { href: "/agents", label: "Agent", icon: "◈" },
  { href: "/terminals", label: "终端", icon: "□" },
  { href: "/system", label: "系统", icon: "◉" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-56 bg-slate-900 text-white flex flex-col">
      <div className="p-4 border-b border-slate-700">
        <h1 className="text-lg font-bold">AI Dev Platform</h1>
        <p className="text-xs text-slate-400 mt-1">开发调度控制台</p>
      </div>
      <nav className="flex-1 p-2">
        {navItems.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors ${
              pathname === item.href
                ? "bg-slate-700 text-white"
                : "text-slate-300 hover:bg-slate-800 hover:text-white"
            }`}
          >
            <span className="w-5 h-5 flex items-center justify-center text-base">{item.icon}</span>
            <span>{item.label}</span>
          </Link>
        ))}
      </nav>
      <div className="p-4 border-t border-slate-700 text-xs text-slate-500">
        v0.1.0 MVP
      </div>
    </aside>
  );
}
