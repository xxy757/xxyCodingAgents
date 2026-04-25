// terminals/[id]/page.tsx - 终端详情页面
// 通过 WebSocket 和 xterm.js 实现交互式终端，支持双向数据传输。
"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";

// TerminalSession 终端会话数据接口
interface TerminalSession {
  id: string;
  task_id: string;
  tmux_session: string;
  tmux_pane: string;
  status: string;
  agent_id?: string;
  log_file_path: string;
  created_at: string;
}

// TerminalDetailPage 终端详情页面组件
export default function TerminalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState<string>("");
  const [session, setSession] = useState<TerminalSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const terminalRef = useRef<HTMLDivElement>(null);    // xterm.js 挂载点
  const xtermRef = useRef<any>(null);                   // xterm 实例引用
  const wsRef = useRef<WebSocket | null>(null);          // WebSocket 连接引用
  const router = useRouter();

  // 解析异步参数获取终端 ID
  useEffect(() => {
    params.then((p) => setId(p.id));
  }, [params]);

  // 加载终端会话信息
  useEffect(() => {
    if (!id) return;
    apiFetch<TerminalSession>(`/api/terminals/${id}`)
      .then(setSession)
      .catch((e) => setError(e.message));
  }, [id]);

  // 初始化 xterm.js 终端和 WebSocket 连接
  useEffect(() => {
    if (!session || !terminalRef.current) return;

    let terminal: any;
    let fitAddon: any;

    const initTerminal = async () => {
      // 动态导入 xterm.js 和适配插件
      const { Terminal } = await import("@xterm/xterm");
      const { FitAddon } = await import("@xterm/addon-fit");

      await import("@xterm/xterm/css/xterm.css");

      // 创建终端实例
      terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "Menlo, Monaco, 'Courier New', monospace",
        theme: {
          background: "#1e1e1e",
          foreground: "#d4d4d4",
          cursor: "#d4d4d4",
        },
      });

      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalRef.current!);
      fitAddon.fit();

      xtermRef.current = terminal;

      // 建立 WebSocket 连接
      const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = `${wsProtocol}//${window.location.host}/api/terminals/${session.id}/ws`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      // 处理服务端推送的输出数据
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === "output" && data.data) {
            terminal.write(data.data);
          } else if (data.type === "error") {
            terminal.write(`\r\n\x1b[31m[Error] ${data.message}\x1b[0m\r\n`);
          }
        } catch {
          terminal.write(event.data);
        }
      };

      ws.onopen = () => {
        terminal.write("\x1b[32m[Connected]\x1b[0m\r\n");
      };

      ws.onclose = () => {
        terminal.write("\r\n\x1b[33m[Disconnected]\x1b[0m\r\n");
      };

      ws.onerror = () => {
        terminal.write("\r\n\x1b[31m[Connection Error]\x1b[0m\r\n");
      };

      // 将用户输入转发到 WebSocket
      terminal.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data }));
        }
      });

      // 窗口大小变化时自动调整终端尺寸
      const handleResize = () => {
        if (fitAddon) fitAddon.fit();
      };
      window.addEventListener("resize", handleResize);

      return () => {
        window.removeEventListener("resize", handleResize);
      };
    };

    initTerminal();

    // 组件卸载时关闭连接和终端
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (terminal) {
        terminal.dispose();
      }
    };
  }, [session]);

  if (!id) return <div className="p-6 text-gray-500">加载中...</div>;

  if (error) {
    return (
      <div className="p-6">
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">{error}</div>
        <button onClick={() => router.back()} className="text-blue-600 hover:underline">
          返回
        </button>
      </div>
    );
  }

  return (
    <div className="p-6 h-full flex flex-col">
      {/* 页头：返回按钮、标题和状态 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4">
          <button onClick={() => router.back()} className="text-blue-600 hover:underline text-sm">
            ← 返回
          </button>
          <h1 className="text-xl font-bold">终端 {id.slice(0, 8)}</h1>
          {session && (
            <span
              className={`px-2 py-0.5 rounded text-xs font-medium ${
                session.status === "active"
                  ? "bg-green-100 text-green-700"
                  : "bg-gray-100 text-gray-700"
              }`}
            >
              {session.status}
            </span>
          )}
        </div>
        {session && (
          <div className="text-sm text-gray-500">
            tmux: <span className="font-mono">{session.tmux_session}</span>
          </div>
        )}
      </div>

      {/* xterm.js 终端容器 */}
      <div
        ref={terminalRef}
        className="flex-1 rounded-lg overflow-hidden border border-gray-700"
        style={{ minHeight: "500px" }}
      />
    </div>
  );
}
