"use client";

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { FileTree } from '@/components/file-tree';
import { CodeEditor } from '@/components/code-editor';
import { GitStatusPanel, CommitHistoryPanel, BranchPicker } from '@/components/GitUI';
import { useEnvStore } from '@/stores/env';
import { TerminalSquare, Box, GitBranch, Clock, RefreshCw, Trash2, ExternalLink, Loader2, Code, Power, Play } from 'lucide-react';
import dynamic from 'next/dynamic';

const Terminal = dynamic(() => import('@/components/terminal').then(mod => mod.Terminal), { 
  ssr: false,
  loading: () => <div className="p-4 text-gray-500 font-mono text-sm border border-gray-800 rounded bg-gray-900 flex-1 flex items-center justify-center">Loading terminal...</div>
});
const DockerLogs = dynamic(() => import('@/components/docker-logs').then(mod => mod.DockerLogs), { 
  ssr: false,
  loading: () => <div className="p-4 text-gray-500 font-mono text-sm border border-gray-800 rounded bg-gray-900 flex-1 flex items-center justify-center">Loading logs...</div>
});
import { formatDistanceToNow } from "date-fns";
import toast from "react-hot-toast";

const statusColors: Record<string, string> = {
  IDLE: "text-gray-400 bg-gray-400/10 border-gray-400/20",
  PENDING: "text-yellow-400 bg-yellow-400/10 border-yellow-400/20",
  BUILDING: "text-blue-400 bg-blue-400/10 border-blue-400/20",
  RUNNING: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20",
  STOPPED: "text-orange-400 bg-orange-400/10 border-orange-400/20",
  FAILED: "text-red-400 bg-red-400/10 border-red-400/20",
};

export default function EnvironmentPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { selectEnvironment } = useEnvStore();
  const [env, setEnv] = useState<any>(null);
  const [loadError, setLoadError] = useState('');
  const [activeFile, setActiveFile] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [activeTab, setActiveTab] = useState<'terminal' | 'logs'>('terminal');

  const fetchEnv = useCallback(async () => {
    try {
      setEnv(await api.environments.get(id));
      setLoadError('');
      return 0;
    } catch (err: any) {
      if (err.status === 404) router.push("/dashboard");
      else setLoadError(err.message || 'Could not load this sandbox.');
      return err.status || 500;
    }
  }, [id, router]);

  useEffect(() => {
    selectEnvironment(id);
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;
    const poll = async () => {
      const status = await fetchEnv();
      if (stopped) return;
      const retryAfter = status === 0 ? 15_000 : status === 429 ? 60_000 : 30_000;
      timer = setTimeout(poll, retryAfter);
    };
    void poll();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
    };
  }, [id, selectEnvironment, fetchEnv]);

  // Auto-switch to logs tab when environment is BUILDING or FAILED
  useEffect(() => {
    if (env?.state === "BUILDING" || env?.state === "FAILED") {
      setActiveTab("logs");
    }
  }, [env?.state]);

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this sandbox? This cannot be undone.")) return;
    setIsDeleting(true);
    try {
      await api.environments.delete(id);
      toast.success("Sandbox deleted successfully");
      router.push("/dashboard");
    } catch (e: any) {
      toast.error(e.message || "Failed to delete sandbox");
      setIsDeleting(false);
    }
  };

  const handleStop = async () => {
    setIsStopping(true);
    try {
      await api.environments.stop(id);
      toast.success("Sandbox stopping...");
      void fetchEnv();
    } catch (e: any) {
      toast.error(e.message || "Failed to stop sandbox");
    } finally {
      setIsStopping(false);
    }
  };

  const handleRestart = async () => {
    setIsRestarting(true);
    try {
      await api.environments.restart(id);
      toast.success("Sandbox restarting...");
      void fetchEnv();
    } catch (e: any) {
      toast.error(e.message || "Failed to restart sandbox");
    } finally {
      setIsRestarting(false);
    }
  };

  if (!env) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-3 text-white/50 font-medium">
        {loadError ? <p className="text-red-300">{loadError}</p> : <p>Loading workspace...</p>}
        {loadError && <button onClick={() => void fetchEnv()} className="rounded border border-white/15 px-3 py-1.5 text-sm hover:bg-white/5">Retry</button>}
      </div>
    );
  }

  return (
    <div className="space-y-4 h-full flex flex-col pb-4">
      {/* Header Card */}
      <div className="bg-surface-container-lowest border border-outline-variant rounded-xl px-6 py-4 shrink-0">
        {loadError && <div className="mb-3 flex items-center justify-between gap-3 rounded-lg border border-red-400/20 bg-red-400/5 px-3 py-2 text-xs text-red-300"><span>{loadError}</span><button onClick={() => void fetchEnv()} className="underline underline-offset-2">Retry</button></div>}
        <div className="flex items-center justify-between gap-4 flex-wrap">
          {/* Left: icon + name + meta */}
          <div className="flex items-center gap-4 min-w-0">
            <div className="w-10 h-10 rounded-xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
              <Box className="w-5 h-5 text-primary-fixed" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-on-surface tracking-tight truncate">
                  {env.name || 'Untitled Sandbox'}
                </h1>
                <span className="px-2 py-0.5 rounded text-[10px] uppercase font-bold bg-primary-fixed/10 text-primary-fixed tracking-wider">
                  Live Workspace
                </span>
              </div>
              <div className="flex items-center gap-3 mt-1 flex-wrap">
                <div className="flex items-center gap-1.5 text-xs text-on-surface-variant/70 font-mono">
                  <GitBranch className="w-3.5 h-3.5" />
                  <a
                    href={env.git_url}
                    target="_blank"
                    rel="noreferrer"
                    className="hover:text-primary-fixed transition-colors truncate max-w-[200px]"
                  >
                    {env.git_url ? env.git_url.replace("https://github.com/", "") : 'No repository'}
                  </a>
                </div>
                {env.git_url && (
                  <BranchPicker 
                    envId={id} 
                    currentBranch={env.git_branch || "main"} 
                    onBranchChanged={fetchEnv} 
                  />
                )}
                <div className="flex items-center gap-1 text-[10px] text-on-surface-variant/50 font-mono">
                  <Clock className="w-3 h-3" />
                  <span>{env.created_at ? formatDistanceToNow(new Date(env.created_at), { addSuffix: true }) : 'Just now'}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Right: status + actions */}
          <div className="flex items-center gap-3 shrink-0 flex-wrap">
            <div
              className={`px-3 py-1.5 rounded-full border text-xs font-bold tracking-wider uppercase flex items-center gap-2 ${statusColors[env.state] || statusColors.IDLE}`}
            >
              {env.state === "BUILDING" && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              {env.state === "RUNNING" && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_6px_#34d399]" />}
              {env.state}
            </div>
            {env.expires_at && (
              <span className="text-xs text-on-surface-variant flex items-center gap-1" title={`Expires ${new Date(env.expires_at).toLocaleString()}`}>
                <Clock className="w-3.5 h-3.5" /> Expires {formatDistanceToNow(new Date(env.expires_at), { addSuffix: true })}
              </span>
            )}
            {env.state === "RUNNING" && (
              <a
                href={env.public_url || `${process.env.NEXT_PUBLIC_API_URL || ''}/p/${id}/`}
                target="_blank"
                rel="noreferrer"
                className="px-4 py-1.5 rounded-lg border border-berth-500/30 text-berth-400 hover:text-berth-300 hover:bg-berth-500/10 flex items-center gap-1.5 text-xs font-semibold transition-colors"
              >
                Open Preview <ExternalLink className="w-3.5 h-3.5" />
              </a>
            )}
            
            {env.state === "RUNNING" && (
              <button
                onClick={handleStop}
                disabled={isStopping}
                className="px-4 py-1.5 rounded-lg border border-orange-500/30 bg-orange-500/5 text-orange-400 hover:bg-orange-500/15 flex items-center gap-1.5 transition-colors disabled:opacity-50 text-xs font-semibold"
              >
                {isStopping ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Power className="w-3.5 h-3.5" />}
                Stop
              </button>
            )}

            {(env.state === "STOPPED" || env.state === "FAILED" || env.state === "RUNNING") && (
              <button
                onClick={handleRestart}
                disabled={isRestarting}
                className="px-4 py-1.5 rounded-lg border border-blue-500/30 bg-blue-500/5 text-blue-400 hover:bg-blue-500/15 flex items-center gap-1.5 transition-colors disabled:opacity-50 text-xs font-semibold"
              >
                {isRestarting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
                Restart
              </button>
            )}

            <button
              onClick={handleDelete}
              disabled={isDeleting}
              className="px-4 py-1.5 rounded-lg border border-red-500/30 bg-red-500/5 text-red-400 hover:bg-red-500/15 flex items-center gap-1.5 transition-colors disabled:opacity-50 text-xs font-semibold"
            >
              {isDeleting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
              Delete
            </button>
          </div>
        </div>
      </div>

      {/* Code Workspace View */}
      <div className="bg-surface-container-lowest border border-outline-variant rounded-xl overflow-hidden flex-1 flex flex-col min-h-[600px]">
        <div className="flex-1 flex overflow-hidden">
          {/* Sidebar */}
          <div className="w-80 border-r border-outline-variant bg-surface-container/40 flex flex-col min-h-0 shrink-0">
            {/* Files Explorer */}
            <div className="flex-1 flex flex-col min-h-0 border-b border-outline-variant">
              <div className="px-4 py-2.5 border-b border-outline-variant font-medium text-on-surface-variant text-xs tracking-wider uppercase">
                Files
              </div>
              <div className="flex-1 overflow-y-auto p-2">
                <FileTree envId={id} selectedPath={activeFile || ''} onSelectFile={setActiveFile} enabled={env.state === 'RUNNING' || env.state === 'STOPPED'} />
              </div>
            </div>

            {/* Git Panels */}
            {env.git_url && (
              <div className="flex-1 flex flex-col min-h-0 overflow-y-auto p-3 gap-3">
                <GitStatusPanel envId={id} />
                <CommitHistoryPanel envId={id} />
              </div>
            )}
          </div>

          {/* Editor and Terminal */}
          <div className="flex-1 flex flex-col bg-slate-950/20 min-h-0 min-w-0">
            {/* Editor Workspace */}
            <div className="flex-1 relative flex flex-col overflow-hidden bg-slate-950/90 font-mono min-h-0">
              {activeFile && (env.state === 'RUNNING' || env.state === 'STOPPED') ? (
                <>
                  <div className="px-4 py-2 bg-slate-950/50 border-b border-white/10 text-xs font-mono text-primary-fixed flex items-center gap-2 shrink-0">
                     <Code className="w-3.5 h-3.5" />
                     {activeFile}
                  </div>
                  <div className="flex-1 relative min-h-0">
                    <CodeEditor envId={id} filePath={activeFile} />
                  </div>
                </>
              ) : (
                <div className="flex-1 flex flex-col items-center justify-center text-center text-white/40">
                  <Code className="w-12 h-12 mb-4 text-white/10" />
                  <h3 className="text-base font-semibold text-white/60 mb-1">
                    {env.state === 'RUNNING' || env.state === 'STOPPED' ? 'Workspace Ready' : `Sandbox ${env.state.toLowerCase()}`}
                  </h3>
                  <p className="text-xs max-w-sm text-white/30">
                    {env.state === 'RUNNING' || env.state === 'STOPPED' ? 'Select a file from the sidebar to view or modify its contents.' : 'Files and terminal will be available when setup finishes.'}
                  </p>
                </div>
              )}
            </div>

            {/* Terminal Panel */}
            <div className="h-64 border-t border-outline-variant flex flex-col shrink-0 bg-slate-950">
              <div className="bg-slate-900 border-b border-white/5 flex items-center">
                <button
                  onClick={() => setActiveTab('terminal')}
                  className={`px-4 py-1.5 text-[10px] uppercase tracking-widest font-bold flex items-center gap-2 border-b-2 transition-colors ${
                    activeTab === 'terminal' 
                      ? 'text-primary-fixed border-primary-fixed bg-white/5' 
                      : 'text-white/50 border-transparent hover:bg-white/5'
                  }`}
                >
                  <TerminalSquare className="w-3.5 h-3.5" />
                  Terminal
                </button>
                <button
                  onClick={() => setActiveTab('logs')}
                  className={`px-4 py-1.5 text-[10px] uppercase tracking-widest font-bold flex items-center gap-2 border-b-2 transition-colors ${
                    activeTab === 'logs' 
                      ? 'text-primary-fixed border-primary-fixed bg-white/5' 
                      : 'text-white/50 border-transparent hover:bg-white/5'
                  }`}
                >
                  <Box className="w-3.5 h-3.5" />
                  Build Logs
                </button>
              </div>
              <div className="flex-1 relative overflow-hidden">
                {activeTab === 'terminal' && env.state === 'RUNNING' ? (
                  <Terminal envId={id} />
                ) : activeTab === 'logs' ? (
                  <DockerLogs envId={id} />
                ) : <div className="flex h-full items-center justify-center text-sm text-white/40">Terminal is available when the sandbox is running.</div>}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
