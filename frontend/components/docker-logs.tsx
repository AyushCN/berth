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
          // If it's the "not running" error, just show waiting
          if (err.message && err.message.includes("sandbox is not running")) {
             // do nothing, let it show "Waiting for logs..."
          } else {
             // Optionally write error to logs
             setLogs(`Error: ${err.message}`);
          }
        }
      }
    };
    
    fetchLogs();
    const interval = setInterval(fetchLogs, 3000);
    
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
