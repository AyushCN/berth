"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import {
  Github,
  Loader2,
  AlertCircle,
  CheckCircle,
  Lock,
  ExternalLink,
  ArrowRight,
} from "lucide-react";
import { motion } from "framer-motion";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";

interface ShareLinkInfo {
  valid: boolean;
  project_name: string;
  role: string;
  expires_at?: string;
}

export default function JoinPage() {
  const { code } = useParams<{ code: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user, setUser, isLoading: authLoading } = useAuthStore();

  const [linkInfo, setLinkInfo] = useState<ShareLinkInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [joining, setJoining] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [redirectUrl, setRedirectUrl] = useState<string | null>(null);

  // Validate the share link code
  useEffect(() => {
    if (!code) return;
    validateLink(code);
  }, [code]);

  const validateLink = async (shareCode: string) => {
    setLoading(true);
    setError(null);
    try {
      const info = await api.projects.shareLinks.validate(shareCode);
      setLinkInfo(info);
    } catch (err: any) {
      setError(err.message || "Invalid or expired share link");
    } finally {
      setLoading(false);
    }
  };

  const handleJoin = async () => {
    if (!code || joining) return;
    setJoining(true);
    setError(null);

    try {
      // Check if already authenticated
      if (!user) {
        // Store the join URL to redirect after login
        const joinUrl = `/join/${code}?redirect=${encodeURIComponent(window.location.href)}`;
        setRedirectUrl(joinUrl);
        router.push(`/login?redirect=${encodeURIComponent(joinUrl)}`);
        return;
      }

      const result = await api.projects.shareLinks.join(code);
      toast.success("Successfully joined the project!");
      
      // Redirect to the workspace
      if (result.workspace?.id) {
        router.push(`/${user?.username || 'user'}/${result.workspace.id}`);
      } else {
        router.push("/dashboard");
      }
    } catch (err: any) {
      setError(err.message || "Failed to join project");
    } finally {
      setJoining(false);
    }
  };

  const handleGitHubLogin = () => {
    if (!code) return;
    const joinUrl = `/join/${code}?redirect=${encodeURIComponent(window.location.href)}`;
    router.push(`/auth/github?redirect=${encodeURIComponent(joinUrl)}`);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 px-4">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center"
        >
          <Loader2 className="w-12 h-12 text-primary-fixed mx-auto mb-6 animate-spin" />
          <p className="text-white/60 text-lg">Validating share link...</p>
        </motion.div>
      </div>
    );
  }

  if (error || !linkInfo?.valid) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 px-4">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="w-full max-w-md bg-slate-900/50 border border-slate-700 rounded-2xl p-8 text-center"
        >
          <div className="w-16 h-16 rounded-full bg-red-500/10 border border-red-500/20 flex items-center justify-center mx-auto mb-6">
            <AlertCircle className="w-8 h-8 text-red-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-3">Link Invalid</h1>
          <p className="text-slate-400 mb-6">
            {error || "This share link is invalid, expired, or has reached its maximum uses."}
          </p>
          <Link
            href="/dashboard"
            className="inline-flex items-center gap-2 px-6 py-3 bg-primary-container text-on-primary-fixed-variant rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] transition-all"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Go to Dashboard
          </Link>
        </motion.div>
      </div>
    );
  }

  const formatExpiry = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24));
    
    if (diffDays <= 0) return "Expired";
    if (diffDays === 1) return "Expires tomorrow";
    return `Expires in ${diffDays} days`;
  };

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4 py-12">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-2xl"
      >
        {/* Header */}
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center mx-auto mb-5">
            <svg className="w-8 h-8 text-primary-fixed" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">Join Project</h1>
          <p className="text-slate-400">You've been invited to collaborate</p>
        </div>

        {/* Project Info Card */}
        <div className="bg-slate-900/50 border border-slate-700 rounded-2xl p-6 mb-6">
          <div className="flex items-center gap-4 mb-4">
            <div className="w-12 h-12 rounded-xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
              <svg className="w-6 h-6 text-primary-fixed" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <h2 className="text-xl font-bold text-white truncate">{linkInfo.project_name}</h2>
              <div className="flex items-center gap-3 mt-1 text-sm text-slate-400">
                <span className="px-2 py-0.5 rounded-full bg-primary-fixed/10 text-primary-fixed font-medium">
                  {linkInfo.role}
                </span>
                {linkInfo.expires_at && (
                  <span className="px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400 font-medium">
                    {formatExpiry(linkInfo.expires_at)}
                  </span>
                )}
              </div>
            </div>
          </div>
          
          <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4 text-sm">
            <div className="flex items-center gap-2 text-slate-400 mb-2">
              <Lock className="w-4 h-4 shrink-0" />
              <span className="font-medium text-slate-300">What you'll get:</span>
            </div>
            <ul className="space-y-1.5 text-slate-400 pl-6">
              <li className="flex items-center gap-2">{"•"} Your own fork workspace with full IDE access</li>
              <li className="flex items-center gap-2">{"•"} Real-time code editing with hot reload</li>
              <li className="flex items-center gap-2">{"•"} Terminal access and preview links</li>
              <li className="flex items-center gap-2">{"•"} Git integration - commit, push, create PRs</li>
            </ul>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="space-y-4">
          {user ? (
            <button
              onClick={handleJoin}
              disabled={joining}
              className="w-full bg-primary-container text-on-primary-fixed-variant px-6 py-4 rounded-xl font-bold text-lg hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-[0.98] transition-all disabled:opacity-50 flex items-center justify-center gap-3"
            >
              {joining ? (
                <>
                  <Loader2 className="w-5 h-5 animate-spin" />
                  Joining...
                </>
              ) : (
                <>
                  <CheckCircle className="w-5 h-5" />
                  Join as {linkInfo.role}
                </>
              )}
            </button>
          ) : (
            <button
              onClick={handleGitHubLogin}
              disabled={joining}
              className="w-full flex items-center justify-center gap-3 bg-github bg-opacity-90 hover:bg-github text-white px-6 py-4 rounded-xl font-bold text-lg transition-all disabled:opacity-50"
              style={{ backgroundColor: '#24292e' }}
            >
              <Github className="w-5 h-5" />
              <span>Sign in with GitHub to Join</span>
            </button>
          )}

          <p className="text-center text-slate-500 text-sm">
            By joining, you agree to the project's collaboration terms.
          </p>
        </div>

        {/* Footer */}
        <div className="mt-8 text-center">
          <Link
            href="/login"
            className="text-slate-500 hover:text-slate-300 text-sm underline"
          >
            Already have an account? Sign in
          </Link>
        </div>
      </motion.div>
    </div>
  );
}