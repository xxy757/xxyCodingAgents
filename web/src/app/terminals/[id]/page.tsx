"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";

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

export default function TerminalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState<string>("");
  const [session, setSession] = useState<TerminalSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const router = useRouter();

  useEffect(() => {
    params.then((p) => setId(p.id));
  }, [params]);

  useEffect(() => {
    if (!id) return;
    apiFetch<TerminalSession>(`/api/terminals/${id}`)
      .then(setSession)
      .catch((e) => setError(e.message));
  }, [id]);

  useEffect(() => {
    if (!session || !terminalRef.current) return;

    let terminal: any;
    let fitAddon: any;

    const initTerminal = async () => {
      const { Terminal } = await import("@xterm/xterm");
      const { FitAddon } = await import("@xterm/addon-fit");

      await import("@xterm/xterm/css/xterm.css");

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

      const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = `${wsProtocol}//${window.location.host}/api/terminals/${session.id}/ws`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

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

      terminal.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data }));
        }
      });

      const handleResize = () => {
        if (fitAddon) fitAddon.fit();
      };
      window.addEventListener("resize", handleResize);

      return () => {
        window.removeEventListener("resize", handleResize);
      };
    };

    initTerminal();

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

      <div
        ref={terminalRef}
        className="flex-1 rounded-lg overflow-hidden border border-gray-700"
        style={{ minHeight: "500px" }}
      />
    </div>
  );
}
