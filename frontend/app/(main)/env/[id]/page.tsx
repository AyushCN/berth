"use client";

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { FileTree } from '@/components/file-tree';
import { CodeEditor } from '@/components/code-editor';
import { GitStatusPanel, CommitHistoryPanel, BranchPicker } from '@/components/GitUI';
import { useEnvStore } from '@/stores/env';
import { TerminalSquare, Box, GitBranch, Clock, RefreshCw, Trash2, ExternalLink, Loader2, Code } from 'lucide-react';
import dynamic from 'next/dynamic';

const Terminal = dynamic(() => import('@/components/terminal').then(mod => mod.Terminal), { 
  ssr: false,
  loading: () => <div className="p-4 text-gray-500 font-mono text-sm border border-gray-800 rounded bg-gray-900 flex-1 flex items-center justify-center">Loading terminal...</div>
});
import { formatDistanceToNow } from "date-fns";
import toast from "react-hot-toast";

const statusColors: Record<string, string> = {
  IDLE: "text-gray-400 bg-gray-400/10 border-gray-400/20",
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
  const [activeFile, setActiveFile] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchEnv = () => {
    api.environments.get(id).then(setEnv).catch(console.error);
  };

  useEffect(() => {
    selectEnvironment(id);
    fetchEnv();
    const interval = setInterval(fetchEnv, 3000);
    return () => clearInterval(interval);
  }, [id, selectEnvironment]);

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

  if (!env) {
    return (
      <div className="flex h-64 items-center justify-center text-white/50 animate-pulse font-medium">
        Loading workspace...
      </div>
    );
  }

  return (
    <div className="space-y-4 h-full flex flex-col pb-4">
      {/* Header Card */}
      <div className="bg-surface-container-lowest border border-outline-variant rounded-xl px-6 py-4 shrink-0">
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
            {env.public_url && (
              <a
                href={env.public_url}
                target="_blank"
                rel="noreferrer"
                className="px-4 py-1.5 rounded-lg border border-berth-500/30 text-berth-400 hover:text-berth-300 hover:bg-berth-500/10 flex items-center gap-1.5 text-xs font-semibold transition-colors"
              >
                Open Preview <ExternalLink className="w-3.5 h-3.5" />
              </a>
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
                <FileTree envId={id} onSelectFile={setActiveFile} />
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
              {activeFile ? (
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
                    Live Editor Workspace
                  </h3>
                  <p className="text-xs max-w-sm text-white/30">
                    Select a file from the sidebar to view or modify its contents.
                  </p>
                </div>
              )}
            </div>

            {/* Terminal Panel */}
            <div className="h-64 border-t border-outline-variant flex flex-col shrink-0 bg-slate-950">
              <div className="px-4 py-1.5 bg-slate-900 border-b border-white/5 text-[10px] uppercase tracking-widest font-bold text-white/50 flex items-center gap-2">
                <TerminalSquare className="w-3.5 h-3.5" />
                Terminal
              </div>
              <div className="flex-1 relative overflow-hidden">
                <Terminal envId={id} />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
