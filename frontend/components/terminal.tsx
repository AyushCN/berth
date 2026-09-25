'use client';

import { useEffect, useRef } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';

export function Terminal({ envId }: { envId: string }) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);

  useEffect(() => {
    if (!terminalRef.current) return;

    const term = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'monospace',
      theme: {
        background: '#1f2937',
        foreground: '#e5e7eb',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    
    // Defer initial fit to ensure layout is complete
    setTimeout(() => {
      try {
        if (terminalRef.current && terminalRef.current.clientHeight > 0) {
          fitAddon.fit();
        }
      } catch (e) {}
    }, 10);

    term.writeln('\x1b[1;34mBerth Terminal\x1b[0m');
    term.writeln('Connecting to sandbox...');

    // Phase 3: WebSocket connection to backend
    const getCookie = (name: string) => {
      const value = `; ${document.cookie}`;
      const parts = value.split(`; ${name}=`);
      if (parts.length === 2) return parts.pop()?.split(';').shift();
      return null;
    };
    const token = getCookie('berth_token');
    
    const apiUrl = new URL(process.env.NEXT_PUBLIC_API_URL || window.location.origin, window.location.origin);
    apiUrl.protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${apiUrl.origin}/ws/sandbox/${envId}?token=${encodeURIComponent(token || '')}`);
    ws.onopen = () => {
      term.writeln('\x1b[32mConnected!\x1b[0m');
    };
    ws.onclose = () => {
      term.writeln('\x1b[31mDisconnected from sandbox.\x1b[0m');
    };
    ws.onerror = () => {
      term.writeln('\x1b[31mConnection error.\x1b[0m');
    };

    ws.onmessage = (e) => term.write(e.data);
    const input = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(data);
    });

    xtermRef.current = term;

    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(() => {
        try {
          if (terminalRef.current && terminalRef.current.clientHeight > 0) {
            fitAddon.fit();
          }
        } catch (e) {}
      });
    });
    
    resizeObserver.observe(terminalRef.current);

    return () => {
      resizeObserver.disconnect();
      input.dispose();
      ws.close();
      term.dispose();
    };
  }, [envId]);

  return <div ref={terminalRef} className="h-full w-full overflow-hidden" />;
}
