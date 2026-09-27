"use client";

import { useState, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Loader2,
  Plus,
  ArrowUp,
  ArrowDown,
  GitBranch,
  GitCommitHorizontal,
  ArrowLeft,
  ArrowRight,
  RefreshCw,
  X,
  CheckCircle,
  AlertCircle,
  MessageSquare,
  Copy,
  Search,
  Filter,
  MoreVertical,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  FileText,
  Diff,
} from "lucide-react";
import { api } from "@/lib/api";
import toast from "react-hot-toast";
import { formatDistanceToNow } from "date-fns";
import { useAuthStore } from "@/stores/auth";

interface GitPanelProps {
  envId: string;
}

interface GitStatus {
  branch: string;
  dirty: boolean;
  ahead: number;
  behind: number;
}

interface CommitEntry {
  hash: string;
  shortHash: string;
  message: string;
  author: string;
  date: string;
}

interface Branch {
  name: string;
  current: boolean;
  remote?: boolean;
}

interface GitPanelState {
  status: GitStatus | null;
  branches: Branch[];
  commits: CommitEntry[];
  loading: boolean;
  error: string | null;
  activeTab: "status" | "branches" | "commits" | "changes";
  showCreateBranch: boolean;
  newBranchName: string;
  commitMessage: string;
  selectedFile: string | null;
  diff: string | null;
}

export function GitPanel({ envId }: GitPanelProps) {
  const [state, setState] = useState<GitPanelState>({
    status: null,
    branches: [],
    commits: [],
    loading: true,
    error: null,
    activeTab: "status",
    showCreateBranch: false,
    newBranchName: "",
    commitMessage: "",
    selectedFile: null,
    diff: null,
  });

  const fetchAll = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const [status, branches, commits] = await Promise.all([
        api.git.status(envId),
        api.git.branches(envId),
        api.git.log(envId),
      ]);
      setState((s) => ({
        ...s,
        status,
        branches: branches.map((b: string) => ({
          name: b,
          current: b === status.branch,
          remote: b.startsWith("origin/"),
        })),
        commits,
        loading: false,
      }));
    } catch (err: any) {
      setState((s) => ({ ...s, error: err.message, loading: false }));
    }
  }, [envId]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const handleFetch = async (fn: () => Promise<any>) => {
    setState((s) => ({ ...s, loading: true }));
    try {
      await fn();
      await fetchAll();
    } catch (err: any) {
      setState((s) => ({ ...s, error: err.message, loading: false }));
    }
  };

  const handleCreateBranch = async () => {
    if (!state.newBranchName.trim()) return;
    await fetchFetch(async () => {
      await api.git.createBranch(envId, state.newBranchName);
      setState((s) => ({ ...s, showCreateBranch: false, newBranchName: "" }));
    });
  };

  const handleCheckout = async (branch: string) => {
    await fetchFetch(async () => {
      await api.git.checkout(envId, branch, false);
    });
  };

  const handlePull = async () => {
    await fetchFetch(async () => {
      await api.git.pull(envId);
    });
  };

  const handleCommit = async () => {
    if (!state.commitMessage.trim()) return;
    await fetchFetch(async () => {
      await api.git.commit(envId, state.commitMessage);
      setState((s) => ({ ...s, commitMessage: "" }));
    });
  };

  const handlePush = async () => {
    await fetchFetch(async () => {
      await api.git.push(envId);
    });
  };

  const handleFetchDiff = async (filePath: string) => {
    
    setState((s) => ({ ...s, selectedFile: filePath, diff: "Diff coming soon..." }));
  };

  const { user } = useAuthStore();
  const userId = user?.id;

  const fetchFetch = async (fn: () => Promise<void>) => {
    setState((s) => ({ ...s, loading: true }));
    try {
      await fn();
      await fetchAll();
    } catch (err: any) {
      setState((s) => ({ ...s, error: err.message, loading: false }));
    }
  };

  if (state.loading && !state.status) {
    return (
      <div className="h-full flex items-center justify-center bg-slate-900/50">
        <Loader2 className="w-8 h-8 text-primary-fixed animate-spin" />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col bg-slate-900/50">
      {/* Tab Bar */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/50">
        <div className="flex items-center gap-1">
          {[
            { key: "status", label: "Status", icon: <GitBranch className="w-3.5 h-3.5" /> },
            { key: "branches", label: "Branches", icon: <GitBranch className="w-3.5 h-3.5" /> },
            { key: "commits", label: "Commits", icon: <GitCommitHorizontal className="w-3.5 h-3.5" /> },
            { key: "changes", label: "Changes", icon: <Diff className="w-3.5 h-3.5" /> },
          ].map((tab) => (
            <button
              key={tab.key}
              onClick={() => setState((s) => ({ ...s, activeTab: tab.key as any }))}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                state.activeTab === tab.key
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
            onClick={() => fetchAll()}
            disabled={state.loading}
            className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800 disabled:opacity-50"
            title="Refresh"
          >
            <RefreshCw className={`w-4 h-4 ${state.loading ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        <AnimatePresence mode="wait">
          {state.activeTab === "status" && (
            <div key="status" className="h-full p-4 overflow-y-auto">
              <div className="space-y-6">
                {/* Current Branch */}
                <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5">
                  <h3 className="font-semibold text-white mb-4 flex items-center gap-2">
                    <GitBranch className="w-5 h-5" />
                    Current Branch
                  </h3>
                  <div className="flex items-center gap-3">
                    <code className="bg-slate-900 px-3 py-2 rounded-lg font-mono text-primary-fixed text-lg">
                      {state.status?.branch || "unknown"}
                    </code>
                    {state.status?.dirty && (
                      <span className="px-2 py-1 rounded-full bg-amber-500/10 text-amber-400 text-xs font-medium border border-amber-500/20">
                        Uncommitted changes
                      </span>
                    )}
                    {!state.status?.dirty && (
                      <span className="px-2 py-1 rounded-full bg-emerald-500/10 text-emerald-400 text-xs font-medium border border-emerald-500/20">
                        Working tree clean
                      </span>
                    )}
                  </div>
                </div>

                {/* Sync Status */}
                <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5">
                  <h3 className="font-semibold text-white mb-4 flex items-center gap-2">
                    <RefreshCw className="w-5 h-5" />
                    Sync Status
                  </h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="bg-slate-900/50 rounded-lg p-4">
                      <div className="flex items-center gap-2 text-slate-400 text-sm mb-2">
                        <ArrowUp className="w-4 h-4 text-emerald-400" />
                        <span>Ahead</span>
                      </div>
                      <div className="text-3xl font-bold text-emerald-400">{state.status?.ahead || 0}</div>
                      <p className="text-xs text-slate-500 mt-1">commits ahead of origin</p>
                    </div>
                    <div className="bg-slate-900/50 rounded-lg p-4">
                      <div className="flex items-center gap-2 text-slate-400 text-sm mb-2">
                        <ArrowDown className="w-4 h-4 text-red-400" />
                        <span>Behind</span>
                      </div>
                      <div className="text-3xl font-bold text-red-400">{state.status?.behind || 0}</div>
                      <p className="text-xs text-slate-500 mt-1">commits behind origin</p>
                    </div>
                  </div>
                  <div className="flex gap-2 mt-4">
                    {state.status && state.status.behind > 0 && (
                      <button
                        onClick={() => handleFetch(async () => { await api.git.pull(envId); })}
                        className="flex-1 px-4 py-2 bg-blue-500/10 border border-blue-500/20 text-blue-400 rounded-lg font-medium hover:bg-blue-500/20 flex items-center justify-center gap-2"
                      >
                        <ArrowDown className="w-4 h-4" /> Pull
                      </button>
                    )}
                    {state.status && (state.status.ahead > 0 || state.status.dirty) && (
                      <button
                        onClick={() => handleFetch(async () => { await api.git.push(envId); })}
                        className="flex-1 px-4 py-2 bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 rounded-lg font-medium hover:bg-emerald-500/20 flex items-center justify-center gap-2"
                      >
                        <ArrowUp className="w-4 h-4" /> Push
                      </button>
                    )}
                    <button
                      onClick={() => handleFetch(async () => { await api.git.pull(envId); })}
                      className="px-4 py-2 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg font-medium hover:bg-slate-800 hover:text-white flex items-center justify-center gap-2"
                    >
                      <RefreshCw className="w-4 h-4" /> Fetch
                    </button>
                  </div>
                </div>

                {/* Working Tree Status */}
                {state.status?.dirty && (
                  <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5">
                    <h3 className="font-semibold text-white mb-4 flex items-center gap-2">
                      <AlertTriangle className="w-5 h-5 text-amber-400" />
                      Uncommitted Changes
                    </h3>
                    <p className="text-slate-400 text-sm mb-3">Working tree has uncommitted changes</p>
                    <div className="flex gap-2">
                      <button
                        onClick={() => setState((s) => ({ ...s, activeTab: "changes" }))}
                        className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-lg font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)]"
                      >
                        View Changes
                      </button>
                      <button
                        onClick={() => setState((s) => ({ ...s, activeTab: "changes" }))}
                        className="px-4 py-2 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg font-medium hover:bg-slate-800 hover:text-white"
                      >
                        Stage All & Commit
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {state.activeTab === "branches" && (
            <div key="branches" className="h-full p-4 overflow-y-auto">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-white">Branches</h3>
                <button
                  onClick={() => setState((s) => ({ ...s, showCreateBranch: true }))}
                  className="px-3 py-1.5 bg-primary-container text-on-primary-fixed-variant rounded-lg text-sm font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] flex items-center gap-1.5"
                >
                  <Plus className="w-3.5 h-3.5" />
                  New Branch
                </button>
              </div>

              {state.showCreateBranch && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="bg-slate-800/50 border border-slate-700 rounded-xl p-4 mb-4"
                >
                  <h4 className="font-semibold text-white mb-3">Create New Branch</h4>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={state.newBranchName}
                      onChange={(e) => setState((s) => ({ ...s, newBranchName: e.target.value }))}
                      placeholder="branch-name"
                      className="flex-1 bg-slate-800/50 border border-slate-700 rounded-lg px-4 py-2 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed"
                      onKeyDown={(e) => e.key === "Enter" && handleFetch(async () => {
                        await api.git.createBranch(envId, state.newBranchName);
                        setState((s) => ({ ...s, showCreateBranch: false, newBranchName: "" }));
                        await fetchAll();
                      })}
                    />
                    <button
                      onClick={() => handleFetch(async () => {
                        await api.git.createBranch(envId, state.newBranchName);
                        setState((s) => ({ ...s, showCreateBranch: false, newBranchName: "" }));
                        await fetchAll();
                      })}
                      className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-lg font-medium"
                    >
                      Create
                    </button>
                    <button
                      onClick={() => setState((s) => ({ ...s, showCreateBranch: false, newBranchName: "" }))}
                      className="px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg"
                    >
                      Cancel
                    </button>
                  </div>
                </motion.div>
              )}

              <div className="space-y-2">
                {state.branches.map((branch) => (
                  <motion.div
                    key={branch.name}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    className={`flex items-center justify-between p-3 rounded-lg ${
                      branch.current
                        ? "bg-primary-container/20 border border-primary-fixed/30"
                        : "bg-slate-800/50 border border-slate-700 hover:bg-slate-800/50"
                    }`}
                  >
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      {branch.current && (
                        <span className="px-1.5 py-0.5 rounded text-xs font-bold bg-primary-container text-on-primary-fixed-variant">
                          Current
                        </span>
                      )}
                      {branch.remote && (
                        <span className="px-1.5 py-0.5 rounded text-xs text-purple-400 bg-purple-500/10">
                          Remote
                        </span>
                      )}
                      <span className="font-mono text-sm truncate">{branch.name}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {!branch.current && (
                        <button
                          onClick={() => handleFetch(async () => {
                            await api.git.checkout(envId, branch.name);
                            await fetchAll();
                          })}
                          className="px-3 py-1.5 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg text-sm hover:bg-slate-800 hover:text-white"
                        >
                          Checkout
                        </button>
                      )}
                    </div>
                  </motion.div>
                ))}
              </div>
            </div>
          )}

          {state.activeTab === "commits" && (
            <div key="commits" className="h-full p-4 overflow-y-auto">
              <div className="space-y-3">
                {state.commits.map((commit) => (
                  <motion.div
                    key={commit.hash}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-3 mb-2">
                          <span className="font-mono text-sm text-slate-400">
                            {commit.shortHash}
                          </span>
                          <span className="px-2 py-0.5 rounded text-xs bg-slate-700 text-slate-300">
                            {formatDistanceToNow(new Date(commit.date), { addSuffix: true })}
                          </span>
                        </div>
                        <p className="text-white font-medium mb-1">{commit.message.split("\n")[0]}</p>
                        <p className="text-slate-400 text-sm flex items-center gap-2">
                          <span>{commit.author}</span>
                        </p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          onClick={() => navigator.clipboard.writeText(commit.hash)}
                          className="p-2 rounded text-slate-400 hover:text-white hover:bg-slate-800"
                          title="Copy hash"
                        >
                          <Copy className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleFetch(async () => {
                            await api.git.checkout(envId, commit.shortHash, true);
                            await fetchAll();
                          })}
                          className="px-2 py-1 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg text-xs hover:bg-slate-800 hover:text-white"
                        >
                          Checkout
                        </button>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            </div>
          )}

          {state.activeTab === "changes" && (
            <div key="changes" className="h-full p-4 overflow-y-auto">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-white">Changes</h3>
                  <div className="flex gap-2">
                    <button className="px-3 py-1.5 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-lg text-sm hover:bg-slate-800 hover:text-white">
                      Stage All
                    </button>
                    <button className="px-3 py-1.5 bg-primary-container text-on-primary-fixed-variant rounded-lg text-sm font-medium">
                      Commit
                    </button>
                  </div>
                </div>
                <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
                  <p className="text-slate-400 text-center py-8">
                    File changes will appear here when sandbox is running
                  </p>
                </div>
              </div>
            </div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}