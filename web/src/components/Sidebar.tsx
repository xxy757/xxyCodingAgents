// Sidebar.tsx - 应用侧边栏导航组件
// 提供主导航菜单，高亮当前路由对应的导航项。
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

// navItems 定义导航菜单项，包含路径、显示标签和图标
const navItems = [
  { href: "/", label: "仪表盘", icon: "▦" },
  { href: "/projects", label: "项目", icon: "▣" },
  { href: "/runs", label: "运行", icon: "▶" },
  { href: "/agents", label: "Agent", icon: "◈" },
  { href: "/terminals", label: "终端", icon: "□" },
  { href: "/system", label: "系统", icon: "◉" },
];

// Sidebar 是应用的侧边栏导航组件
export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-56 bg-neutral-900 text-white flex flex-col" role="navigation" aria-label="主导航">
      {/* 品牌区域 */}
      <div className="p-4 border-b border-neutral-700">
        <h1 className="text-lg font-bold">AI Dev Platform</h1>
        <p className="text-xs text-neutral-400 mt-1">开发调度控制台</p>
      </div>
      {/* 导航链接列表 */}
      <nav className="flex-1 p-2">
        {navItems.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            aria-current={pathname === item.href ? "page" : undefined}
            className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors focus-visible:ring-2 focus-visible:ring-primary-400 focus-visible:outline-none ${
              pathname === item.href
                ? "bg-neutral-700 text-white"
                : "text-neutral-300 hover:bg-neutral-800 hover:text-white"
            }`}
          >
            <span className="w-5 h-5 flex items-center justify-center text-base" aria-hidden="true">{item.icon}</span>
            <span>{item.label}</span>
          </Link>
        ))}
      </nav>
      {/* 版本信息 */}
      <div className="p-4 border-t border-neutral-700 text-xs text-neutral-500">
        v0.1.0 MVP
      </div>
    </aside>
  );
}
