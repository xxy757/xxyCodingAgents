// terminals/[id]/page.tsx - 终端详情
// xterm.js + WebSocket 实现交互式终端。
"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, type TerminalSession } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import { ArrowLeft } from "@phosphor-icons/react/dist/ssr";

export default function TerminalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const [id, setId] = useState<string>("");
  const [session, setSession] = useState<TerminalSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const router = useRouter();

  useEffect(() => { params.then((p) => setId(p.id)); }, [params]);

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

    const init = async () => {
      const { Terminal } = await import("@xterm/xterm");
      const { FitAddon } = await import("@xterm/addon-fit");
      await import("@xterm/xterm/css/xterm.css");

      terminal = new Terminal({
        cursorBlink: true,
        fontSize: 13,
        fontFamily: "var(--font-mono), Menlo, Monaco, 'Courier New', monospace",
        theme: {
          background: "#09090b",
          foreground: "#d4d4d8",
          cursor: "#a1a1aa",
          selectionBackground: "rgba(161,161,170,0.2)",
        },
      });

      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalRef.current!);
      fitAddon.fit();
      xtermRef.current = terminal;

      const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${wsProtocol}//${window.location.host}/api/terminals/${session.id}/ws`);
      wsRef.current = ws;

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === "output" && data.data) terminal.write(data.data);
          else if (data.type === "error") terminal.write(`\r\n\x1b[31m[Error] ${data.message}\x1b[0m\r\n`);
        } catch {
          terminal.write(event.data);
        }
      };

      ws.onopen = () => terminal.write("\x1b[32m[Connected]\x1b[0m\r\n");
      ws.onclose = () => terminal.write("\r\n\x1b[33m[Disconnected]\x1b[0m\r\n");
      ws.onerror = () => terminal.write("\r\n\x1b[31m[Connection Error]\x1b[0m\r\n");

      terminal.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: "input", data }));
      });

      const onResize = () => fitAddon?.fit();
      window.addEventListener("resize", onResize);
      return () => window.removeEventListener("resize", onResize);
    };

    init();

    return () => {
      wsRef.current?.close();
      terminal?.dispose();
    };
  }, [session]);

  if (!id) return <div className="text-sm text-zinc-400 py-12 text-center">加载中...</div>;

  if (error) {
    return (
      <div className="space-y-4">
        <div className="p-3 bg-red-50 border border-red-200/60 rounded-xl text-red-700 text-sm">{error}</div>
        <button onClick={() => router.back()} className="text-sm text-accent-600 hover:underline">返回</button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col animate-fade-up">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4">
          <button
            onClick={() => router.back()}
            className="w-9 h-9 rounded-lg bg-zinc-100 flex items-center justify-center hover:bg-zinc-200 pressable transition-colors"
          >
            <ArrowLeft className="w-4 h-4 text-zinc-600" />
          </button>
          <h1 className="text-lg font-semibold tracking-tight text-zinc-900">
            终端 <span className="font-mono text-zinc-400">{id.slice(0, 8)}</span>
          </h1>
          {session && <StatusBadge status={session.status} />}
        </div>
        {session && (
          <div className="text-xs text-zinc-400 font-mono">
            tmux: {session.tmux_session}
          </div>
        )}
      </div>

      <div
        ref={terminalRef}
        className="flex-1 rounded-xl overflow-hidden border border-zinc-800 p-3"
        style={{ minHeight: "500px" }}
      />
    </div>
  );
}
