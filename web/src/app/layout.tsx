// layout.tsx - 根布局
// 提供 Geist 字体、侧边栏导航和主内容区。
import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import "./globals.css";
import { Sidebar } from "@/components/Sidebar";

export const metadata: Metadata = {
  title: "AI Dev Platform",
  description: "AI Development Orchestration Platform",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className="flex h-screen overflow-hidden bg-zinc-50 font-sans">
        <a href="#main-content" className="skip-link">
          跳转到主内容
        </a>
        <Sidebar />
        <main
          id="main-content"
          className="flex-1 overflow-y-auto"
          style={{ scrollBehavior: "smooth" }}
        >
          <div className="page-container px-6 py-8 md:px-10 md:py-10">
            {children}
          </div>
        </main>
      </body>
    </html>
  );
}
