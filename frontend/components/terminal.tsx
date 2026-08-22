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
    fitAddon.fit();

    term.writeln('\x1b[1;34mBerth Terminal\x1b[0m');
    term.writeln('Connecting to sandbox...');

    // Phase 3: WebSocket connection to backend
    // const ws = new WebSocket(`ws://localhost:8080/ws/sandbox/${envId}`);
    // ws.onmessage = (e) => term.write(e.data);
    // term.onData((data) => ws.send(data));

    xtermRef.current = term;

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      term.dispose();
    };
  }, [envId]);

  return <div ref={terminalRef} className="h-full w-full" />;
}
