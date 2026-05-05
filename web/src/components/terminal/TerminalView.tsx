'use client';

import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { cn } from '@/lib/utils';

interface TerminalViewProps {
  sessionId: string;
  className?: string;
}

export function TerminalView({ sessionId, className }: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace",
      theme: {
        background: '#141414',
        foreground: '#e8e8e8',
        cursor: '#e8e8e8',
        selectionBackground: 'rgba(22, 119, 255, 0.3)',
        black: '#141414',
        red: '#ff4d4f',
        green: '#52c41a',
        yellow: '#faad14',
        blue: '#1677ff',
        magenta: '#9254de',
        cyan: '#13c2c2',
        white: '#e8e8e8',
      },
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(containerRef.current);
    fitAddon.fit();
    terminalRef.current = terminal;

    // WebSocket 连接
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/terminals/${sessionId}/ws`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output' && msg.data) {
          terminal.write(msg.data);
        } else if (msg.type === 'error' && msg.message) {
          terminal.write(`\r\n\x1b[31mError: ${msg.message}\x1b[0m\r\n`);
        }
      } catch {
        terminal.write(event.data);
      }
    };

    ws.onopen = () => terminal.write('\r\n\x1b[32m已连接到终端\x1b[0m\r\n');
    ws.onclose = () => terminal.write('\r\n\x1b[33m终端连接已断开\x1b[0m\r\n');
    ws.onerror = () => terminal.write('\r\n\x1b[31m终端连接错误\x1b[0m\r\n');

    // 用户输入发送到 WebSocket
    terminal.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // 窗口大小调整
    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      terminal.dispose();
    };
  }, [sessionId]);

  return (
    <div
      ref={containerRef}
      className={cn('rounded-lg overflow-hidden bg-[#141414] p-2', className)}
    />
  );
}
