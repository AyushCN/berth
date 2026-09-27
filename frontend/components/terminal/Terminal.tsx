"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { X, Loader2, Minimize2, Maximize2, Copy, Trash2, TerminalSquare } from "lucide-react";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";

interface TerminalProps {
  envId: string;
  onClose?: () => void;
}

export function Terminal({ envId, onClose }: TerminalProps) {
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [buffer, setBuffer] = useState<string>("");
  const [input, setInput] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [cursorPosition, setCursorPosition] = useState({ row: 0, col: 0 });

  const terminalRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const websocketRef = useRef<WebSocket | null>(null);
  const bufferRef = useRef<string>("");

  const { user } = useAuthStore();

  const appendToBuffer = useCallback((data: string) => {
    bufferRef.current += data;
    setBuffer(bufferRef.current);
    // Auto-scroll to bottom
    setTimeout(() => {
      if (terminalRef.current) {
        terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
      }
    }, 0);
  }, []);

  const sendCommand = useCallback((cmd: string) => {
    if (websocketRef.current?.readyState === WebSocket.OPEN) {
      websocketRef.current.send(
        JSON.stringify({ type: "input", data: cmd + "\r" })
      );
      setHistory((prev) => [...prev.slice(-99), cmd]);
      setHistoryIndex(-1);
      setInput("");
    }
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case "Enter":
        if (input.trim()) {
          sendCommand(input);
        } else {
          sendCommand("");
        }
        break;
      case "ArrowUp":
        e.preventDefault();
        if (history.length > 0 && historyIndex < history.length - 1) {
          const newIndex = Math.min(historyIndex + 1, history.length - 1);
          setHistoryIndex(newIndex);
          setInput(history[history.length - 1 - newIndex]);
        }
        break;
      case "ArrowDown":
        e.preventDefault();
        if (historyIndex > 0) {
          const newIndex = historyIndex - 1;
          setHistoryIndex(newIndex);
          setInput(history[history.length - 1 - newIndex]);
        } else if (historyIndex === 0) {
          setHistoryIndex(-1);
          setInput("");
        }
        break;
      case "Tab":
        e.preventDefault();
        // Could implement tab completion here
        break;
      case "c":
        if (e.ctrlKey) {
          sendCommand("\x03"); // Ctrl+C
        }
        break;
      case "d":
        if (e.ctrlKey && input === "") {
          sendCommand("\x04"); // Ctrl+D
        }
        break;
      case "l":
        if (e.ctrlKey) {
          e.preventDefault();
          sendCommand("clear\r");
        }
        break;
    }
  };

  const connect = useCallback(async () => {
    if (!user) return;

    setConnecting(true);
    setError(null);

    try {
      // Get auth token
      const token = localStorage.getItem("access_token");
      if (!token) throw new Error("Not authenticated");

      // Connect to WebSocket
      const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080"}/ws/sandboxes/${envId}?token=${token}`;
      const ws = new WebSocket(wsUrl);
      websocketRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        setConnecting(false);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          switch (msg.type) {
            case "output":
              appendToBuffer(msg.data);
              break;
            case "resize":
              // Handle resize acknowledgment
              break;
            case "error":
              setError(msg.message);
              break;
            case "close":
              setConnected(false);
              break;
          }
        } catch (err) {
          // If not JSON, treat as raw output
          appendToBuffer(event.data);
        }
      };

      ws.onerror = (err) => {
        console.error("WebSocket error:", err);
        setError("Connection error");
        setConnecting(false);
      };

      ws.onclose = () => {
        setConnected(false);
        if (!connecting) {
          setError("Connection closed");
        }
      };
    } catch (err: any) {
      setError(err.message || "Failed to connect");
      setConnecting(false);
    }
  }, [envId, user, appendToBuffer]);

  useEffect(() => {
    connect();
    return () => {
      websocketRef.current?.close();
    };
  }, [connect]);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Handle resize
  useEffect(() => {
    if (!connected || !websocketRef.current) return;

    const resize = () => {
      if (websocketRef.current?.readyState === WebSocket.OPEN) {
        const rows = Math.floor((window.innerHeight - 200) / 20);
        const cols = Math.floor(window.innerWidth / 8);
        websocketRef.current.send(
          JSON.stringify({ type: "resize", rows, cols })
        );
      }
    };

    window.addEventListener("resize", resize);
    resize();
    return () => window.removeEventListener("resize", resize);
  }, [connected]);

  if (connecting) {
    return (
      <div className="h-full flex items-center justify-center bg-slate-950">
        <Loader2 className="w-8 h-8 text-primary-fixed animate-spin" />
      </div>
    );
  }

  if (error && !connected) {
    return (
      <div className="h-full flex flex-col items-center justify-center bg-slate-950 p-8">
        <AlertCircle className="w-12 h-12 text-error mb-4" />
        <h3 className="text-lg font-semibold text-white mb-2">Connection Failed</h3>
        <p className="text-slate-400 text-center mb-4 max-w-md">{error}</p>
        <button
          onClick={connect}
          className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)]"
        >
          Retry Connection
        </button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col bg-slate-950">
      {/* Terminal Header */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-slate-800 bg-slate-900/50">
        <div className="flex items-center gap-2">
          <TerminalSquare className="w-4 h-4 text-slate-400" />
          <span className="text-xs font-medium text-slate-300">bash</span>
          {connected && (
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" title="Connected" />
          )}
          {!connected && (
            <span className="w-1.5 h-1.5 rounded-full bg-red-400" title="Disconnected" />
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => {
              bufferRef.current = "";
              setBuffer("");
            }}
            className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800"
            title="Clear terminal"
          >
            <Trash2 className="w-4 h-4" />
          </button>
          <button
            onClick={() => {
              navigator.clipboard.writeText(buffer);
              toast.success("Copied to clipboard");
            }}
            className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800"
            title="Copy output"
          >
            <Copy className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Terminal Output */}
      <div
        ref={terminalRef}
        className="flex-1 overflow-y-auto p-4 font-mono text-sm text-slate-200 bg-slate-950 select-text"
        style={{
          fontFamily: '"JetBrains Mono", "Fira Code", "Monaco", "Consolas", monospace',
          lineHeight: 1.5,
          letterSpacing: "0.3px",
        }}
      >
        <pre className="m-0 whitespace-pre-wrap word-break-break-word">{buffer || <span className="text-slate-600">Terminal ready. Type commands to begin...</span>}</pre>
        <div className="flex items-baseline gap-1">
          <span className="text-green-400">user@berth</span>
          <span className="text-blue-400">:</span>
          <span className="text-yellow-400">~</span>
          <span className="text-green-400">$</span>
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            className="flex-1 bg-transparent border-none outline-none text-slate-100 font-mono text-sm"
            style={{ minWidth: "100px" }}
            autoComplete="off"
            spellCheck={false}
          />
          <span className="w-2 h-5 bg-slate-200 animate-blink" />
        </div>
      </div>
    </div>
  );
}

// Need to import AlertCircle and toast
import { AlertCircle } from "lucide-react";
import toast from "react-hot-toast";