// Sidebar.tsx - 侧边栏导航
// Phosphor 图标 + 高亮当前路由 + 紧凑专业风格。
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  House,
  FolderNotchOpenIcon as FolderNotchOpen,
  PlayCircle,
  Robot,
  Terminal,
  GearSix,
  Lightning,
} from "@phosphor-icons/react";

const navItems = [
  { href: "/", label: "仪表盘", icon: House },
  { href: "/projects", label: "项目", icon: FolderNotchOpen },
  { href: "/runs", label: "运行", icon: PlayCircle },
  { href: "/agents", label: "Agent", icon: Robot },
  { href: "/terminals", label: "终端", icon: Terminal },
  { href: "/system", label: "系统", icon: GearSix },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-60 shrink-0 bg-zinc-950 text-zinc-300 flex flex-col border-r border-zinc-800/50">
      {/* Brand */}
      <div className="px-5 py-5 border-b border-zinc-800/50">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-accent-600 flex items-center justify-center">
            <Lightning weight="fill" className="w-4 h-4 text-white" />
          </div>
          <div>
            <h1 className="text-sm font-semibold text-zinc-100 tracking-tight">
              AI Dev Platform
            </h1>
            <p className="text-[11px] text-zinc-500 leading-none mt-0.5">
              v0.1.0
            </p>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-4 space-y-0.5" role="navigation" aria-label="主导航">
        {navItems.map((item) => {
          const active = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href));
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={`
                flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] font-medium
                transition-all duration-200 ease-[cubic-bezier(0.16,1,0.3,1)]
                pressable
                ${active
                  ? "bg-zinc-800/80 text-zinc-100 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]"
                  : "text-zinc-400 hover:bg-zinc-800/40 hover:text-zinc-200"
                }
              `}
            >
              <Icon
                weight={active ? "fill" : "regular"}
                className={`w-[18px] h-[18px] ${active ? "text-accent-400" : ""}`}
              />
              <span>{item.label}</span>
              {active && (
                <div className="ml-auto w-1.5 h-1.5 rounded-full bg-accent-400" />
              )}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="px-5 py-4 border-t border-zinc-800/50">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-accent-500 animate-pulse" />
          <span className="text-[11px] text-zinc-500">服务运行中</span>
        </div>
      </div>
    </aside>
  );
}
