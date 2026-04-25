// layout.tsx - 应用根布局
// 提供全局 HTML 结构、侧边栏和主内容区域布局。
import type { Metadata } from "next";
import "./globals.css";
import { Sidebar } from "@/components/Sidebar";

export const metadata: Metadata = {
  title: "AI Dev Platform",
  description: "AI Development Orchestration Platform",
};

// RootLayout 是应用的根布局组件，包含侧边栏和主内容区
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className="flex h-screen overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-y-auto bg-gray-50 p-6">
          {children}
        </main>
      </body>
    </html>
  );
}
