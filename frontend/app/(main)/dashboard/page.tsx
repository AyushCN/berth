"use client";
import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  Plus,
  GitBranch,
  Clock,
  Box,
  XCircle,
  Code,
  ArrowRight,
  MoreVertical,
  GitFork,
} from "lucide-react";
import toast from "react-hot-toast";
import { formatDistanceToNow } from "date-fns";
import { motion } from "framer-motion";
import { useEnvStore } from "@/stores/env";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";
import { CreateEnvironmentModal } from "@/components/create-env-modal";

const statusConfig: Record<
  string,
  { color: string; dot: string; label: string }
> = {
  IDLE: {
    color: "text-gray-400 bg-gray-400/10 border-gray-400/20",
    dot: "bg-gray-400",
    label: "Idle",
  },
  BUILDING: {
    color: "text-blue-400 bg-blue-400/10 border-blue-400/20",
    dot: "bg-blue-400 animate-bounce",
    label: "Building",
  },
  RUNNING: {
    color: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20",
    dot: "bg-emerald-400 animate-pulse shadow-[0_0_6px_#34d399]",
    label: "Running",
  },
  STOPPED: {
    color: "text-orange-400 bg-orange-400/10 border-orange-400/20",
    dot: "bg-orange-400",
    label: "Stopped",
  },
  FAILED: {
    color: "text-red-400 bg-red-400/10 border-red-400/20",
    dot: "bg-red-400",
    label: "Failed",
  },
};

function StatusBadge({ status }: { status: string }) {
  const cfg = statusConfig[status] ?? statusConfig.IDLE;
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold tracking-widest uppercase border ${cfg.color}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
      {cfg.label}
    </span>
  );
}

import { Suspense } from "react";

function SandboxesDashboardContent() {
  const { user } = useAuthStore();
  const { environments, setEnvironments, isLoading, setLoading } = useEnvStore();
  const [error, setError] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [retryKey, setRetryKey] = useState(0);
  const [showCreate, setShowCreate] = useState(false);
  const searchParams = useSearchParams();
  const projectId = searchParams.get("project_id");
  const [forkingId, setForkingId] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    setLoading(true);
    setError(false);
    
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;
    let retryDelay = 15_000;

    const fetchEnvs = async () => {
      const fetchPromise = projectId 
        ? api.projects.sandboxes(projectId)
        : api.environments.list();

      try {
        const data = await fetchPromise;
        setEnvironments(data.sandboxes || []);
        setError(false);
        setErrorMessage("");
        retryDelay = 15_000;
      } catch (err) {
        const status = (err as { status?: number })?.status;
        setError(true);
        setErrorMessage(status === 429
          ? "The API is rate limiting requests. We’ll retry automatically in a minute."
          : "Couldn’t refresh your sandboxes. We’ll try again shortly.");
        // The user bucket resets every minute. Back off immediately on 429s.
        retryDelay = status === 429 ? 60_000 : Math.min(retryDelay * 2, 60_000);
      } finally {
        setLoading(false);
        if (!stopped) timer = setTimeout(fetchEnvs, retryDelay);
      }
    };

    void fetchEnvs();
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [user, setEnvironments, setLoading, projectId, retryKey]);

  const handleFork = async (e: React.MouseEvent, envId: string, envName: string) => {
    e.preventDefault();
    if (forkingId) return;
    setForkingId(envId);
    try {
      await api.environments.fork(envId, { name: `${envName} (Fork)` });
      toast.success("Sandbox forked successfully!");
      // Refetch
      const fetchPromise = projectId 
        ? api.projects.sandboxes(projectId)
        : api.environments.list();
      const data = await fetchPromise;
      setEnvironments(data.sandboxes || []);
    } catch (err: any) {
      toast.error(err.message || "Failed to fork sandbox");
    } finally {
      setForkingId(null);
    }
  };

  return (
    <div className="space-y-8 pb-12">
      {/* Page Header */}
      <div className="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center">
            <Code className="w-5 h-5 text-primary-fixed" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-on-surface">
              Development Sandboxes
            </h1>
            <p className="text-on-surface-variant text-sm">
              Manage and collaborate on your interactive code workspaces
            </p>
          </div>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 bg-primary-container text-on-primary-fixed-variant px-5 py-2.5 rounded-xl font-bold text-sm hover:shadow-[0_0_20px_rgba(0,240,255,0.25)] active:scale-95 transition-all cursor-pointer border-none"
        >
          <Plus className="w-4 h-4" />
          New Sandbox
        </button>
      </div>

      {/* Educational Banner */}
      <div className="bg-primary-fixed/5 border border-primary-fixed/20 rounded-xl p-5 mb-8 flex items-start gap-4">
        <div className="w-10 h-10 rounded-full bg-primary-fixed/10 flex items-center justify-center shrink-0 mt-0.5">
          <Code className="w-5 h-5 text-primary-fixed" />
        </div>
        <div>
          <h3 className="font-bold text-on-surface mb-1 text-primary-fixed">
            Interactive Code Environments
          </h3>
          <p className="text-sm text-on-surface-variant leading-relaxed">
            Sandboxes give you a full Web IDE to view and edit code in
            real-time. Any file changes you save will automatically hot-reload
            the container.
          </p>
        </div>
      </div>

      {/* Sandboxes Grid */}
      <div className="mt-8">
        {isLoading && environments.length === 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
            {[1, 2, 3, 4].map((i) => (
              <div
                key={i}
                className="bg-surface-container-lowest border border-outline-variant rounded-xl h-52 animate-pulse"
              />
            ))}
          </div>
        ) : error && environments.length === 0 ? (
          <div className="bg-error-container/10 border border-error/20 p-8 rounded-2xl text-center">
            <XCircle className="w-10 h-10 text-error mx-auto mb-3" />
            <p className="text-error font-semibold">{errorMessage || "Failed to load sandboxes"}</p>
            <button onClick={() => { setError(false); setLoading(true); setRetryKey((key) => key + 1); }} className="mt-4 rounded-lg border border-error/30 px-4 py-2 text-sm font-semibold text-error hover:bg-error/10 transition-colors">
              Try again
            </button>
          </div>
        ) : environments?.length === 0 ? (
          <div className="bg-surface-container-lowest border border-outline-variant border-dashed rounded-xl py-24 text-center flex flex-col items-center">
            <div className="w-16 h-16 rounded-2xl bg-primary-fixed/5 border border-primary-fixed/10 flex items-center justify-center mb-5">
              <Code className="w-8 h-8 text-on-surface-variant/30" />
            </div>
            <h3 className="text-xl font-bold text-on-surface mb-2">
              No sandboxes yet
            </h3>
            <p className="text-on-surface-variant mb-8 max-w-xs">
              Create your first interactive workspace to write and edit code.
            </p>
            <button
              onClick={() => setShowCreate(true)}
              className="bg-primary-container text-on-primary-fixed-variant px-8 py-3 rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-95 transition-all cursor-pointer border-none"
            >
              New Sandbox
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
            {environments?.map((env, idx) => (
              <motion.div
                key={env.id}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: idx * 0.04, duration: 0.35 }}
              >
                <Link
                  href={`/${user?.username || 'user'}/${env.id}`}
                  className="block group h-full"
                >
                  <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5 h-full flex flex-col gap-4 transition-all duration-300 hover:border-primary-fixed/40 hover:shadow-[0_0_24px_rgba(0,240,255,0.08)] relative overflow-hidden">
                    <div className="absolute inset-0 bg-gradient-to-br from-primary-fixed/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none" />
                    <div className="flex items-start justify-between gap-3 relative z-10">
                      <div className="w-9 h-9 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
                        <Box className="w-4 h-4 text-primary-fixed" />
                      </div>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={env.state} />
                        <button
                          onClick={(e) => handleFork(e, env.id, env.name)}
                          disabled={forkingId === env.id}
                          className="p-1 rounded bg-surface-container hover:bg-surface-container-high transition-colors text-on-surface-variant z-20 disabled:opacity-50"
                          title="Fork Sandbox"
                        >
                          <GitFork className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    <div className="relative z-10 flex-1 min-w-0">
                      <h3 className="font-bold text-base text-on-surface group-hover:text-primary-fixed transition-colors truncate mb-1.5">
                        {env.name || 'Untitled Sandbox'}
                      </h3>
                      <div className="flex items-center gap-1.5 text-xs text-on-surface-variant font-mono">
                        <GitBranch className="w-3.5 h-3.5 shrink-0 text-on-surface-variant/50" />
                        <span className="truncate">
                          {env.git_url ? env.git_url.replace("https://github.com/", "") : 'No repository'}
                        </span>
                      </div>
                    </div>
                    <div className="relative z-10 flex items-center justify-between text-[10px] font-bold text-on-surface-variant tracking-wider uppercase pt-3 border-t border-outline-variant/50">
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {env.created_at ? formatDistanceToNow(new Date(env.created_at), {
                          addSuffix: true,
                        }) : 'Just now'}
                      </div>
                      <ArrowRight className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 text-primary-fixed transition-all group-hover:translate-x-0.5 duration-200" />
                    </div>
                  </div>
                </Link>
              </motion.div>
            ))}
          </div>
        )}
      </div>

      {error && environments.length > 0 && (
        <div role="status" className="flex items-center justify-between gap-4 rounded-xl border border-amber-300/20 bg-amber-300/5 px-4 py-3 text-sm text-amber-100/80">
          <span>{errorMessage || "Couldn’t refresh your sandboxes. Your last results are still shown."}</span>
          <button onClick={() => setRetryKey((key) => key + 1)} className="shrink-0 font-semibold text-amber-100 underline underline-offset-4">Retry now</button>
        </div>
      )}

      {showCreate && <CreateEnvironmentModal onClose={() => setShowCreate(false)} projectId={projectId} />}
    </div>
  );
}

export default function SandboxesDashboard() {
  return (
    <Suspense fallback={<div className="p-8 text-center text-white/50 animate-pulse">Loading dashboard...</div>}>
      <SandboxesDashboardContent />
    </Suspense>
  );
}
