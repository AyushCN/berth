'use client';

import { useEffect, useRef, useState } from 'react';
import { api } from '@/lib/api';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';

export function DockerLogs({ envId }: { envId: string }) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const [logs, setLogs] = useState<string>('');
  
  useEffect(() => {
    let isMounted = true;
    
    const fetchLogs = async () => {
      try {
        const data = await api.environments.logs(envId);
        if (isMounted && data && data.logs) {
          setLogs(data.logs);
        }
      } catch (err: any) {
        if (isMounted) {
          // The API says "environment is not running (no container id)" and
          // "unauthorized to get logs", not "sandbox is not running", so this
          // branch never matched and the raw Go error was written into the
          // terminal buffer. Treat any not-yet-ready state as "keep waiting".
          const msg = err.message || "";
          const waiting =
            msg.includes("not running") ||
            msg.includes("no container id") ||
            msg.includes("unauthorized to get logs");
          if (waiting) {
             // do nothing, let it show "Waiting for logs..."
          } else {
             setLogs(`Error: ${msg}`);
          }
        }
      }
    };

    fetchLogs();
    // 5s rather than 3s: this panel alone accounted for 20 requests a minute,
    // which on its own exhausted the shared per-user rate limit.
    const interval = setInterval(fetchLogs, 5000);
    
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [envId]);

  useEffect(() => {
    if (!terminalRef.current) return;

    const term = new XTerm({
      cursorBlink: false,
      fontSize: 13,
      fontFamily: 'monospace',
      disableStdin: true,
      theme: {
        background: '#1f2937',
        foreground: '#e5e7eb',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    
    setTimeout(() => {
      try {
        if (terminalRef.current && terminalRef.current.clientHeight > 0) {
          fitAddon.fit();
        }
      } catch (e) {}
    }, 10);

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
      term.dispose();
    };
  }, []);

  useEffect(() => {
    if (xtermRef.current && logs) {
      xtermRef.current.clear();
      // Write the logs properly preserving newlines
      xtermRef.current.write(logs.replace(/\n/g, '\r\n'));
      xtermRef.current.scrollToBottom();
    } else if (xtermRef.current && !logs) {
      xtermRef.current.clear();
      xtermRef.current.writeln('\x1b[36mWaiting for logs...\x1b[0m');
    }
  }, [logs]);

  return <div ref={terminalRef} className="h-full w-full overflow-hidden" />;
}
