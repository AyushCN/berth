"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import useSWR from "swr";
import { motion, AnimatePresence } from "framer-motion";
import {
  Box,
  TerminalSquare,
  GitBranch,
  Share2,
  Settings,
  ChevronDown,
  X,
  Loader2,
  AlertCircle,
  Code,
  GitCommitHorizontal,
  Server,
  ExternalLink,
  Minimize2,
  Maximize2,
  Power,
  Play,
  Square,
  Search,
  Bell,
  MoreVertical,
  Menu,
  ArrowLeft,
  ArrowRight,
  FlipHorizontal,
  FlipVertical,
  Maximize,
  Minimize,
  Plus,
  Copy,
} from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import { formatDistanceToNow } from "date-fns";
import dynamic from "next/dynamic";
import { useEditorStore } from "@/stores/editor";
import { useEnvStore } from "@/stores/env";
import { useAuthStore } from "@/stores/auth";

// Dynamic imports for heavy components
const Terminal = dynamic(
  () => import("@/components/terminal").then((mod) => mod.Terminal),
  { ssr: false, loading: () => <TerminalSkeleton /> }
);

const CodeEditor = dynamic(
  () => import("@/components/code-editor").then((mod) => mod.CodeEditor),
  { ssr: false, loading: () => <EditorSkeleton /> }
);

const FileTree = dynamic(
  () => import("@/components/file-tree").then((mod) => mod.FileTree),
  { ssr: false, loading: () => <FileTreeSkeleton /> }
);

const GitPanel = dynamic(
  () => import("@/components/git-panel").then((mod) => mod.GitPanel),
  { ssr: false, loading: () => <PanelSkeleton /> }
);

const PresenceBar = dynamic(
  () => import("@/components/presence-bar").then((mod) => mod.PresenceBar),
  { ssr: false, loading: () => null }
);

function TerminalSkeleton() {
  return <div className="h-full bg-slate-950 animate-pulse flex items-center justify-center text-slate-500">Loading terminal...</div>;
}

function EditorSkeleton() {
  return <div className="h-full bg-slate-950 animate-pulse flex items-center justify-center text-slate-500">Loading editor...</div>;
}

function FileTreeSkeleton() {
  return <div className="h-full bg-slate-900/50 animate-pulse p-4">Loading files...</div>;
}

function PanelSkeleton() {
  return <div className="h-full bg-slate-900/50 animate-pulse p-4">Loading...</div>;
}

interface Sandbox {
  id: string;
  name: string;
  state: string;
  git_url?: string;
  git_branch?: string;
  created_at: string;
  public_url?: string;
  port?: number;
  owner_id: string;
  project_id?: string;
  has_uncommitted_changes?: boolean;
  last_activity_at?: string;
}

interface FileNode {
  path: string;
  name: string;
  is_directory: boolean;
  children?: FileNode[];
  git_status?: "modified" | "added" | "deleted" | "untracked" | "clean";
}

const statusConfig: Record<string, { color: string; dot: string; label: string }> = {
  IDLE: { color: "text-gray-400 bg-gray-400/10 border-gray-400/20", dot: "bg-gray-400", label: "Idle" },
  PENDING: { color: "text-yellow-400 bg-yellow-400/10 border-yellow-400/20", dot: "bg-yellow-400 animate-bounce", label: "Pending" },
  BUILDING: { color: "text-blue-400 bg-blue-400/10 border-blue-400/20", dot: "bg-blue-400 animate-bounce", label: "Building" },
  RUNNING: { color: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20", dot: "bg-emerald-400 animate-pulse", label: "Running" },
  STOPPED: { color: "text-orange-400 bg-orange-400/10 border-orange-400/20", dot: "bg-orange-400", label: "Stopped" },
  FAILED: { color: "text-red-400 bg-red-400/10 border-red-400/20", dot: "bg-red-400", label: "Failed" },
};

function StatusBadge({ status }: { status: string }) {
  const cfg = statusConfig[status] ?? statusConfig.IDLE;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold tracking-widest uppercase border ${cfg.color}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
      {cfg.label}
    </span>
  );
}

export default function EnvironmentPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user } = useAuthStore();
  const { selectEnvironment, activeEnvironmentId, environments, setEnvironments, updateEnvironment } = useEnvStore();
  const { openFiles, activeFileId, setActiveFile, closeFile } = useEditorStore();

  const [sandbox, setSandbox] = useState<Sandbox | null>(null);
  const [loadError, setLoadError] = useState("");
  const [activeTab, setActiveTab] = useState<"terminal" | "git" | "problems" | "output">("terminal");
  const [bottomPanelOpen, setBottomPanelOpen] = useState(true);
  const [bottomPanelHeight, setBottomPanelHeight] = useState(200);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [sidebarWidth, setSidebarWidth] = useState(280);
  const [showSettings, setShowSettings] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);

  const bottomPanelRef = useRef<HTMLDivElement>(null);
  const isResizingRef = useRef(false);

  // Fetch sandbox data
  const { data: sandboxData, error: sandboxError, isLoading: sandboxLoading, mutate: mutateSandbox } = useSWR<Sandbox>(
    id ? `/api/environments/${id}` : null,
    (url) => fetch(`${process.env.NEXT_PUBLIC_API_URL || ""}${url}`, { credentials: "include" }).then((r) => r.json()),
    { refreshInterval: 30000, revalidateOnFocus: true }
  );

  useEffect(() => {
    if (sandboxData) {
      setSandbox(sandboxData);
      selectEnvironment(id);
      // Update env store
      updateEnvironment(id, sandboxData);
    }
    if (sandboxError) {
      setLoadError(sandboxError.message || "Could not load sandbox");
    }
  }, [sandboxData, sandboxError, id, selectEnvironment, updateEnvironment]);

  // Handle sandbox actions
  const handleStart = async () => {
    try {
      await api.environments.start(id);
      toast.success("Starting sandbox...");
      mutateSandbox();
    } catch (err: any) {
      toast.error(err.message || "Failed to start");
    }
  };

  const handleStop = async () => {
    try {
      await api.environments.stop(id);
      toast.success("Stopping sandbox...");
      mutateSandbox();
    } catch (err: any) {
      toast.error(err.message || "Failed to stop");
    }
  };

  const handleRestart = async () => {
    try {
      await api.environments.restart(id);
      toast.success("Restarting sandbox...");
      mutateSandbox();
    } catch (err: any) {
      toast.error(err.message || "Failed to restart");
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this sandbox? This cannot be undone.")) return;
    try {
      await api.environments.delete(id);
      toast.success("Sandbox deleted");
      router.push("/dashboard");
    } catch (err: any) {
      toast.error(err.message || "Failed to delete");
    }
  };

  // File tree handlers
  const handleFileSelect = useCallback((path: string) => {
    if (!sandbox) return;
    setActiveFile(path);
    // Fetch file content
    api.files.getContent(id, path)
      .then((content) => {
        // Update editor store with file content
        // This would be handled by the editor store
      })
      .catch(() => toast.error("Failed to load file"));
  }, [id, setActiveFile]);

  // Bottom panel resize
  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRef.current = true;
    const startY = e.clientY;
    const startHeight = bottomPanelHeight;

    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizingRef.current) return;
      const delta = startY - e.clientY;
      const newHeight = Math.max(100, Math.min(600, startHeight + delta));
      setBottomPanelHeight(newHeight);
    };

    const handleMouseUp = () => {
      isResizingRef.current = false;
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  // Sidebar resize
  const handleSidebarMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRef.current = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizingRef.current) return;
      const delta = e.clientX - startX;
      const newWidth = Math.max(200, Math.min(500, startWidth + delta));
      setSidebarWidth(newWidth);
    };

    const handleMouseUp = () => {
      isResizingRef.current = false;
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  if (sandboxLoading && !sandbox) {
    return (
      <div className="flex h-screen items-center justify-center bg-slate-950">
        <div className="text-center">
          <Loader2 className="w-12 h-12 text-primary-fixed mx-auto mb-6 animate-spin" />
          <p className="text-white/60 text-lg">Loading sandbox...</p>
        </div>
      </div>
    );
  }

  if (loadError || !sandbox) {
    return (
      <div className="flex h-screen items-center justify-center bg-slate-950">
        <div className="text-center">
          <AlertCircle className="w-12 h-12 text-error mx-auto mb-4" />
          <h2 className="text-xl font-bold text-white mb-2">Sandbox Not Found</h2>
          <p className="text-slate-400 mb-4">{loadError || "This sandbox doesn't exist or you don't have access."}</p>
          <button onClick={() => router.push("/dashboard")} className="px-6 py-3 bg-primary-container text-on-primary-fixed-variant rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] transition-all">
            Back to Dashboard
          </button>
        </div>
      </div>
    );
  }

  const cfg = statusConfig[sandbox.state] ?? statusConfig.IDLE;

  return (
    <div className="h-screen bg-slate-950 flex flex-col">
      {/* Top Bar */}
      <div className="bg-slate-900/80 backdrop-blur-sm border-b border-slate-800 px-4 py-3 shrink-0">
        <div className="flex items-center justify-between h-12">
          {/* Left: Project/Sandbox Info */}
          <div className="flex items-center gap-4">
            <button
              onClick={() => router.push("/dashboard")}
              className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors lg:hidden"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
                <Box className="w-5 h-5 text-primary-fixed" />
              </div>
              <div className="min-w-0">
                <h1 className="text-lg font-bold text-white truncate">{sandbox.name}</h1>
                <div className="flex items-center gap-2 text-xs text-slate-400 font-mono">
                  <span className="truncate max-w-[200px]">
                    {sandbox.git_url ? sandbox.git_url.replace("https://github.com/", "") : "No repository"}
                  </span>
                  {sandbox.git_branch && (
                    <>
                      <GitBranch className="w-3 h-3 shrink-0" />
                      <span>{sandbox.git_branch}</span>
                    </>
                  )}
                </div>
              </div>
            </div>
            <StatusBadge status={sandbox.state} />
          </div>

          {/* Center: Preview URL */}
          <div className="flex items-center gap-2 lg:block hidden">
            {sandbox.state === "RUNNING" && sandbox.port && (
              <a
                href={sandbox.public_url || `${process.env.NEXT_PUBLIC_API_URL || ""}/p/${id}/`}
                target="_blank"
                rel="noreferrer"
                className="px-3 py-1.5 rounded-lg border border-emerald-500/30 bg-emerald-500/5 text-emerald-400 hover:bg-emerald-500/10 flex items-center gap-1.5 text-xs font-medium transition-colors group"
              >
                <ExternalLink className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
                <span>Open Preview</span>
                <span className="px-1.5 py-0.5 bg-emerald-500/20 text-emerald-300 rounded text-[10px] font-mono">
                  :{sandbox.port}
                </span>
              </a>
            )}
            {sandbox.state !== "RUNNING" && (
              <span className="px-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-slate-500 text-xs flex items-center gap-2">
                <Server className="w-3.5 h-3.5" />
                <span>{sandbox.state === "BUILDING" ? "Building..." : "Start sandbox to preview"}</span>
              </span>
            )}
          </div>

          {/* Right: Actions */}
          <div className="flex items-center gap-2">
            <div className="relative">
              <button
                onClick={() => setShowShareModal(true)}
                className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
                title="Share"
              >
                <Share2 className="w-5 h-5" />
              </button>
            </div>

            <div className="relative">
              <button
                onClick={() => setShowSettings(true)}
                className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
                title="Settings"
              >
                <Settings className="w-5 h-5" />
              </button>
            </div>

            {/* User Menu */}
            <div className="relative">
              <button
                className="flex items-center gap-2 p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              >
                <div className="w-7 h-7 rounded-full bg-primary-fixed/20 border border-primary-fixed/30 flex items-center justify-center">
                  {user?.username?.charAt(0).toUpperCase()}
                </div>
                <ChevronDown className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Workspace */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar - File Tree */}
        <AnimatePresence>
          {sidebarOpen && (
            <motion.div
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: sidebarWidth, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              className="bg-slate-900 border-r border-slate-800 flex flex-col shrink-0 overflow-hidden relative"
              style={{ width: sidebarWidth, minWidth: 200, maxWidth: 500 }}
            >
              {/* Sidebar Header */}
              <div className="flex items-center justify-between px-3 py-2.5 border-b border-slate-800 bg-slate-900/50">
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
                    <Code className="w-4 h-4 text-primary-fixed" />
                  </div>
                  <h3 className="font-semibold text-white text-sm">Files</h3>
                </div>
                <div className="flex items-center gap-1">
                  <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="Search files">
                    <Search className="w-4 h-4" />
                  </button>
                  <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="Collapse all">
                    <Minimize2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Resize Handle */}
              <div
                onMouseDown={handleSidebarMouseDown}
                className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary-fixed/20 transition-colors"
                aria-label="Resize sidebar"
              />

              {/* File Tree */}
              <div className="flex-1 overflow-hidden">
                <FileTree
                  envId={id}
                  selectedPath={activeFileId || ""}
                  onSelectFile={handleFileSelect}
                  enabled={sandbox.state === "RUNNING" || sandbox.state === "STOPPED"}
                />
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Sidebar Resize Handle (when closed) */}
        {!sidebarOpen && (
          <div
            onClick={() => setSidebarOpen(true)}
            className="w-1 bg-slate-800 border-r border-slate-700 flex items-center justify-center cursor-pointer hover:bg-primary-fixed/20 transition-colors"
            style={{ height: "100%" }}
          >
            <ChevronDown className="w-5 h-5 text-slate-500 rotate-90" />
          </div>
        )}

        {/* Main Editor Area */}
        <div className="flex-1 flex flex-col bg-slate-950/50 min-w-0">
          {/* Editor Tabs Bar */}
          <div className="bg-slate-900 border-b border-slate-800 px-3 py-2 flex items-center gap-2 shrink-0">
            <div className="flex-1 flex items-center gap-1 overflow-x-auto pb-1">
              {openFiles.map((file) => (
                <motion.button
                  key={file.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 20 }}
                  onClick={() => setActiveFile(file.path)}
                  className={`flex items-center gap-2 px-3 py-1.5 rounded-t-lg text-sm font-medium transition-all shrink-0 ${
                    activeFileId === file.path
                      ? "bg-slate-800 text-white border-b-2 border-primary-fixed"
                      : "bg-slate-900/50 text-slate-400 hover:bg-slate-800 hover:text-white"
                  }`}
                >
                  <Code className="w-3.5 h-3.5 shrink-0" />
                  <span className="truncate max-w-[150px]">{file.name}</span>
                  {file.hasChanges && (
                    <span className="w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
                  )}
                  <button
                    onClick={(e) => { e.stopPropagation(); closeFile(file.path); }}
                    className="ml-1 p-0.5 rounded text-slate-500 hover:text-red-400 hover:bg-red-500/10"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </motion.button>
              ))}
              {openFiles.length === 0 && (
                <div className="flex-1 flex items-center justify-center text-slate-500 text-sm py-4">
                  No files open. Select a file from the sidebar.
                </div>
              )}
            </div>
            <div className="flex items-center gap-1 ml-auto">
              <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="Split editor">
                <FlipHorizontal className="w-4 h-4" />
              </button>
              <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="New file">
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Editor Area */}
          <div className="flex-1 relative overflow-hidden">
            <AnimatePresence mode="wait">
              {activeFileId && sandbox ? (
                <CodeEditor
                  key={activeFileId}
                  envId={id}
                  filePath={activeFileId}
                />
              ) : (
                <div className="flex h-full items-center justify-center bg-slate-950">
                  <div className="text-center text-slate-500">
                    <Code className="w-16 h-16 mx-auto mb-4 text-slate-700" />
                    <h3 className="text-lg font-semibold text-slate-400 mb-2">No file open</h3>
                    <p className="text-slate-500 max-w-sm">Select a file from the sidebar to start editing</p>
                  </div>
                </div>
              )}
            </AnimatePresence>
          </div>

          {/* Bottom Panel Resizer */}
          {bottomPanelOpen && (
            <div
              onMouseDown={handleMouseDown}
              className="h-1.5 bg-slate-800 border-t border-slate-700 cursor-ns-resize flex items-center justify-center hover:bg-primary-fixed/20 transition-colors"
              style={{ userSelect: "none" }}
              aria-label="Resize bottom panel"
            >
              <div className="w-12 h-0.5 bg-slate-600 rounded-full" />
            </div>
          )}

          {/* Bottom Panel - Terminal / Git / Problems / Output */}
          <AnimatePresence>
            {bottomPanelOpen && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: bottomPanelHeight, opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                className="bg-slate-900 border-t border-slate-800 flex flex-col overflow-hidden relative"
                style={{ height: bottomPanelHeight, minHeight: 100, maxHeight: 600 }}
              >
                {/* Panel Header */}
                <div className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/50 shrink-0">
                  <div className="flex items-center gap-1">
                    {[
                      { key: "terminal", label: "Terminal", icon: <TerminalSquare className="w-3.5 h-3.5" /> },
                      { key: "git", label: "Git", icon: <GitBranch className="w-3.5 h-3.5" /> },
                      { key: "problems", label: "Problems", icon: <AlertCircle className="w-3.5 h-3.5" /> },
                      { key: "output", label: "Output", icon: <Code className="w-3.5 h-3.5" /> },
                    ].map((tab) => (
                      <button
                        key={tab.key}
                        onClick={() => setActiveTab(tab.key as typeof activeTab)}
                        className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                          activeTab === tab.key
                            ? "bg-primary-container text-on-primary-fixed-variant"
                            : "text-slate-400 hover:text-white hover:bg-slate-800"
                        }`}
                      >
                        <span className="flex items-center gap-1.5">{tab.icon}{tab.label}</span>
                      </button>
                    ))}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setBottomPanelOpen(false)}
                      className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800"
                      title="Close panel"
                    >
                      <X className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setBottomPanelHeight(bottomPanelHeight > 300 ? 200 : 400)}
                      className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800"
                      title={bottomPanelHeight > 300 ? "Minimize" : "Maximize"}
                    >
                      {bottomPanelHeight > 300 ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
                    </button>
                  </div>
                </div>

                {/* Panel Content */}
                <div className="flex-1 overflow-hidden relative">
                  <AnimatePresence mode="wait">
                    {activeTab === "terminal" && (
                      <Terminal key="terminal" envId={id} />
                    )}
                    {activeTab === "git" && (
                      <GitPanel key="git" envId={id} />
                    )}
                    {activeTab === "problems" && (
                      <div key="problems" className="h-full flex items-center justify-center text-slate-500">
                        <div className="text-center">
                          <Code className="w-12 h-12 mx-auto mb-3 text-slate-600" />
                          <p className="text-slate-500">No problems detected</p>
                          <p className="text-slate-500/50 text-sm mt-1">LSP diagnostics will appear here</p>
                        </div>
                      </div>
                    )}
                    {activeTab === "output" && (
                      <div key="output" className="h-full flex items-center justify-center text-slate-500">
                        <div className="text-center">
                          <Code className="w-12 h-12 mx-auto mb-3 text-slate-600" />
                          <p className="text-slate-500">Task output will appear here</p>
                        </div>
                      </div>
                    )}
                  </AnimatePresence>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Right Sidebar - Git Panel / Presence */}
        <AnimatePresence>
          {sidebarOpen && sandbox?.git_url && (
            <motion.div
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 320, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              className="bg-slate-900 border-l border-slate-800 flex flex-col shrink-0 overflow-hidden w-80"
            >
              <div className="flex items-center justify-between px-3 py-2.5 border-b border-slate-800 bg-slate-900/50">
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center shrink-0">
                    <GitBranch className="w-4 h-4 text-emerald-400" />
                  </div>
                  <h3 className="font-semibold text-white text-sm">Git</h3>
                </div>
              </div>
              <div className="flex-1 overflow-hidden">
                <GitPanel envId={id} />
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Status Bar */}
      <div className="bg-slate-900 border-t border-slate-800 px-4 py-2 shrink-0">
        <div className="flex items-center justify-between h-8">
          <div className="flex items-center gap-4 text-xs text-slate-400 font-mono">
            <span className="flex items-center gap-1.5">
              <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
              <span className={cfg.color.replace("bg-", "text-")}>{cfg.label}</span>
            </span>
            {sandbox.port && (
              <span className="flex items-center gap-1.5 text-emerald-400">
                <ExternalLink className="w-3 h-3" />
                <span>:{sandbox.port}</span>
              </span>
            )}
            {sandbox.git_branch && (
              <span className="flex items-center gap-1.5">
                <GitBranch className="w-3 h-3" />
                <span>{sandbox.git_branch}</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-400">
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              LSP: Connected
            </span>
            <span>UTF-8</span>
            <span>LF</span>
            <span>Spaces: 2</span>
          </div>
        </div>
      </div>

      {/* Modals */}
      {showSettings && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onClick={() => setShowSettings(false)}>
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            className="w-full max-w-2xl bg-slate-900 border border-slate-700 rounded-2xl overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <h2 className="text-lg font-bold text-white">Sandbox Settings</h2>
              <button onClick={() => setShowSettings(false)} className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
              {/* General */}
              <div className="space-y-4">
                <h3 className="font-semibold text-white">General</h3>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">Name</label>
                  <input defaultValue={sandbox.name} className="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed" />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-white">Auto-stop on inactivity</span>
                    <p className="text-sm text-slate-500">Stop sandbox after 30 minutes of inactivity</p>
                  </div>
                  <input type="checkbox" defaultChecked className="w-5 h-5 accent-primary-fixed rounded border-slate-600" />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-white">Auto-save files</span>
                    <p className="text-sm text-slate-500">Automatically save file changes</p>
                  </div>
                  <input type="checkbox" defaultChecked className="w-5 h-5 accent-primary-fixed rounded border-slate-600" />
                </div>
              </div>

              {/* Resources */}
              <div className="space-y-4 border-t border-slate-700 pt-6">
                <h3 className="font-semibold text-white">Resources</h3>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">Memory Limit</label>
                    <select className="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed">
                      <option value="256">256 MB</option>
                      <option value="512" selected>512 MB</option>
                      <option value="1024">1 GB</option>
                      <option value="2048">2 GB</option>
                      <option value="4096">4 GB</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">CPU Limit</label>
                    <select className="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed">
                      <option value="500">0.5 vCPU</option>
                      <option value="1000" selected>1 vCPU</option>
                      <option value="2000">2 vCPU</option>
                      <option value="4000">4 vCPU</option>
                    </select>
                  </div>
                </div>
              </div>

              {/* Expiration */}
              <div className="space-y-4 border-t border-slate-700 pt-6">
                <h3 className="font-semibold text-white">Expiration</h3>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">Expires At</label>
                  <input type="datetime-local" className="w-full bg-slate-800/50 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed" />
                </div>
              </div>

              {/* Actions */}
              <div className="flex justify-end gap-3 pt-6 border-t border-slate-700">
                <button onClick={() => setShowSettings(false)} className="px-4 py-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-xl font-medium">
                  Cancel
                </button>
                <button className="px-6 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)]">
                  Save Settings
                </button>
              </div>
            </div>
          </motion.div>
        </div>
      )}

      {showShareModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" onClick={() => setShowShareModal(false)}>
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            className="w-full max-w-md bg-slate-900 border border-slate-700 rounded-2xl overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <h2 className="text-lg font-bold text-white">Share Sandbox</h2>
              <button onClick={() => setShowShareModal(false)} className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">Share Link</label>
                <div className="relative">
                  <input
                    readOnly
                    value={`${window.location.origin}/join/${sandbox.id}`}
                    className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-slate-700 rounded-xl text-white font-mono text-sm placeholder-slate-500"
                  />
                  <button
                    onClick={() => { navigator.clipboard.writeText(`${window.location.origin}/join/${sandbox.id}`); toast.success("Copied!"); }}
                    className="absolute right-3 top-1/2 -translate-y-1/2 p-2 text-slate-400 hover:text-white"
                  >
                    <Copy className="w-5 h-5" />
                  </button>
                </div>
              </div>
              <div className="flex gap-3 pt-4">
                <button onClick={() => setShowShareModal(false)} className="flex-1 px-4 py-3 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-xl font-medium hover:bg-slate-800 hover:text-white">
                  Close
                </button>
              </div>
            </div>
          </motion.div>
        </div>
      )}

      {/* Presence Bar */}
      <PresenceBar envId={id} />
    </div>
  );
}