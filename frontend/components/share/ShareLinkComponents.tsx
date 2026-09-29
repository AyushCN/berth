"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  Copy,
  Check,
  Calendar,
  Users,
  X,
  Link as LinkIcon,
  Loader2,
  Trash2,
  AlertTriangle,
} from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import { formatDistanceToNow } from "date-fns";

export interface ShareLink {
  id: string;
  code: string;
  role: "VIEWER" | "EDITOR";
  created_by: string;
  expires_at: string | null;
  max_uses: number | null;
  // The API emits `uses_count` (backend/internal/domain/share_link.go).
  // This was declared as `current_uses`, so the usage pill rendered
  // "undefined / 3" on every link.
  uses_count: number;
  created_at: string;
}

interface CreateShareLinkModalProps {
  projectId: string;
  onClose: () => void;
  onCreated?: () => void;
}

export function CreateShareLinkModal({ projectId, onClose, onCreated }: CreateShareLinkModalProps) {
  const [role, setRole] = useState<"VIEWER" | "EDITOR">("EDITOR");
  const [expiresAt, setExpiresAt] = useState("");
  const [maxUses, setMaxUses] = useState("");
  const [creating, setCreating] = useState(false);
  const [isOpen, setIsOpen] = useState(true);

  useEffect(() => {
    setIsOpen(true);
    return () => setIsOpen(false);
  }, []);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const data: any = { role };
      if (expiresAt) data.expires_at = expiresAt;
      if (maxUses) data.max_uses = parseInt(maxUses, 10);

      await api.projects.shareLinks.create(projectId, data);
      toast.success("Share link created!");
      onCreated?.();
      onClose();
    } catch (err: any) {
      toast.error(err.message || "Failed to create share link");
    } finally {
      setCreating(false);
    }
  };

  if (!isOpen) return null;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        className="w-full max-w-md bg-slate-900 border border-slate-700 rounded-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <h2 className="text-lg font-bold text-white">Create Share Link</h2>
          <button
            onClick={() => { setIsOpen(false); setTimeout(onClose, 200); }}
            className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-5">
          {/* Role Selection */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-3">Access Level</label>
            <div className="grid grid-cols-2 gap-3">
              {(["EDITOR", "VIEWER"] as const).map((r) => (
                <button
                  key={r}
                  onClick={() => setRole(r)}
                  className={`p-4 rounded-xl border-2 text-sm font-medium transition-all ${
                    role === r
                      ? "border-primary-fixed bg-primary-fixed/10 text-primary-fixed"
                      : "border-slate-700 bg-slate-800/50 text-slate-400 hover:border-slate-600 hover:text-slate-300"
                  }`}
                >
                  <div className="font-bold capitalize">{r.toLowerCase()}</div>
                  <div className="text-xs mt-1 opacity-80">
                    {r === "EDITOR" ? "Can edit code & push" : "Read-only access"}
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Expiration */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Expires At (optional)</label>
            <div className="relative">
              <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
              <input
                type="datetime-local"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed transition-all"
              />
            </div>
            <p className="text-xs text-slate-500 mt-1">Leave empty for no expiration</p>
          </div>

          {/* Max Uses */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Max Uses (optional)</label>
            <div className="relative">
              <Users className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
              <input
                type="number"
                min="1"
                value={maxUses}
                onChange={(e) => setMaxUses(e.target.value)}
                placeholder="Unlimited"
                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed transition-all"
              />
            </div>
            <p className="text-xs text-slate-500 mt-1">Leave empty for unlimited uses</p>
          </div>

          {/* Actions */}
          <div className="flex gap-3 pt-4">
            <button
              onClick={() => { setIsOpen(false); setTimeout(onClose, 200); }}
              className="flex-1 px-4 py-3 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-xl font-medium hover:bg-slate-800 hover:text-white transition-all"
            >
              Cancel
            </button>
            <button
              onClick={handleCreate}
              disabled={creating}
              className="flex-1 px-4 py-3 bg-primary-container text-on-primary-fixed-variant rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-[0.98] transition-all disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {creating ? (
                <>
                  <svg className="w-5 h-5 animate-spin" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Creating...
                </>
              ) : (
                "Create Link"
              )}
            </button>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}
interface ShareLinkListProps {
  projectId: string;
  links: ShareLink[];
  onRefresh: () => void;
}

export function ShareLinkList({ projectId, links, onRefresh }: ShareLinkListProps) {
  const [revoking, setRevoking] = useState<string | null>(null);

  const copyToClipboard = async (code: string) => {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/join/${code}`);
      toast.success("Link copied to clipboard!");
    } catch {
      toast.error("Failed to copy");
    }
  };

  const handleRevoke = async (linkId: string) => {
    if (!confirm("Are you sure you want to revoke this share link?")) return;
    setRevoking(linkId);
    try {
      await api.projects.shareLinks.revoke(projectId, linkId);
      toast.success("Share link revoked");
      onRefresh();
    } catch (err: any) {
      toast.error(err.message || "Failed to revoke link");
    } finally {
      setRevoking(null);
    }
  };

  const formatExpiry = (dateStr: string | null) => {
    if (!dateStr) return "Never";
    const date = new Date(dateStr);
    const now = new Date();
    if (date < now) return "Expired";
    return formatDistanceToNow(date, { addSuffix: true });
  };

  const getRoleBadge = (role: string) => (
    <span className={`px-2 py-0.5 rounded-full text-xs font-bold ${
      role === "EDITOR"
        ? "bg-primary-fixed/10 text-primary-fixed"
        : "bg-slate-700/50 text-slate-400"
    }`}>
      {role}
    </span>
  );

  const getUsageText = (current: number, max: number | null) => {
    if (max === null) return `${current} / ∞`;
    return `${current} / ${max}`;
  };

  if (links.length === 0) {
    return (
      <div className="bg-slate-800/50 border border-slate-700/50 rounded-xl p-8 text-center">
        <LinkIcon className="w-12 h-12 text-slate-600 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-white mb-2">No share links yet</h3>
        <p className="text-slate-500 text-sm">Create a share link to invite collaborators to this project</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {links.map((link) => (
        <motion.div
          key={link.id}
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
        >
          <div className="flex items-center justify-between gap-4">
            <div className="flex-1 min-w-0 flex items-center gap-4">
              <div className="relative">
                <input
                  type="text"
                  readOnly
                  value={`${window.location.origin}/join/${link.code}`}
                  className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2 font-mono text-sm text-slate-300 w-64 truncate"
                />
                <button
                  onClick={() => copyToClipboard(link.code)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-slate-400 hover:text-white transition-colors"
                  title="Copy link"
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
              <div className="flex items-center gap-2 flex-wrap">
                {getRoleBadge(link.role)}
                <span className="px-2 py-0.5 rounded-full bg-slate-700/50 text-slate-400 text-xs font-mono">
                  {getUsageText(link.uses_count, link.max_uses)}
                </span>
                <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                  link.expires_at && new Date(link.expires_at) < new Date()
                    ? "bg-red-500/10 text-red-400"
                    : "bg-slate-700/50 text-slate-400"
                }`}>
                  {link.expires_at ? formatDistanceToNow(new Date(link.expires_at), { addSuffix: true }) : "Never"}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                onClick={() => copyToClipboard(link.code)}
                className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
                title="Copy link"
              >
                <LinkIcon className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleRevoke(link.id)}
                disabled={revoking === link.id}
                className="p-2 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-colors disabled:opacity-50"
                title="Revoke link"
              >
                {revoking === link.id ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Trash2 className="w-4 h-4" />
                )}
              </button>
            </div>
          </div>
        </motion.div>
      ))}
    </div>
  );
}
