"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import useSWR from "swr";
import { motion } from "framer-motion";
import {
  Folder,
  Box,
  Users,
  Share2,
  Settings,
  Plus,
  Trash2,
  Edit,
  ExternalLink,
  Loader2,
  XCircle,
  AlertCircle,
  CheckCircle,
} from "lucide-react";
import toast from "react-hot-toast";
import { api } from "@/lib/api";
import { presentState } from "@/lib/environment-state";
import { formatDistanceToNow } from "date-fns";
import { CreateShareLinkModal, ShareLinkList, ShareLink } from "@/components/share/ShareLinkComponents";

interface Project {
  git_url?: string;
  git_branch?: string;
  id: string;
  name: string;
  description?: string;
  is_public?: boolean;
  owner_organization_id: string;
  created_at: string;
  updated_at: string;
}

interface Sandbox {
  id: string;
  name: string;
  state: string;
  git_url?: string;
  git_branch?: string;
  created_at: string;
  public_url?: string;
}

function Tab({ active, label, icon, onClick }: { active: boolean; label: string; icon: React.ReactNode; onClick?: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-3 rounded-xl text-sm font-medium transition-all ${
        active
          ? "bg-primary-container text-on-primary-fixed-variant shadow-[0_0_20px_rgba(0,240,255,0.15)]"
          : "text-on-surface-variant hover:bg-surface-container hover:text-on-surface"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<"overview" | "sandboxes" | "share" | "members" | "settings">("overview");
  const [showCreateSandbox, setShowCreateSandbox] = useState(false);
  const [showCreateShareLink, setShowCreateShareLink] = useState(false);
  const [shareLinks, setShareLinks] = useState<ShareLink[]>([]);
  const [newSandboxName, setNewSandboxName] = useState("");
  const [isCreating, setIsCreating] = useState(false);

  // Fetch project
  const fetcher = (url: string) => fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}${url}`, { credentials: "include" }).then((r) => r.json());

  const { data: projectData, error: projectError, isLoading: projectLoading, mutate: mutateProject } = useSWR<{ project: Project }>(
    id ? `/api/projects/${id}` : null,
    fetcher
  );

  // Fetch sandboxes
  const { data: sandboxesData, error: sandboxesError, isLoading: sandboxesLoading, mutate: mutateSandboxes } = useSWR<{ sandboxes: Sandbox[] }>(
    id ? `/api/projects/${id}/sandboxes` : null,
    fetcher
  );

  // Fetch share links
  const { data: linksData, mutate: mutateLinks } = useSWR<{ links: ShareLink[] }>(
    id ? `/api/projects/${id}/share-links` : null,
    fetcher
  );

  const project = projectData?.project;
  const sandboxes = sandboxesData?.sandboxes || [];
  const links = linksData?.links || [];

  const handleDeleteProject = async () => {
    if (!confirm("Are you sure you want to delete this project? This cannot be undone.")) return;
    toast("Project deletion coming soon");
  };

  const handleCreateSandbox = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSandboxName.trim() || !project) return;
    setIsCreating(true);
    try {
      await api.environments.create({
        name: newSandboxName,
        git_url: project.git_url || "",
        git_branch: project.git_branch || "main",
        project_id: project.id,
      });
      toast.success("Sandbox created successfully!");
      setShowCreateSandbox(false);
      setNewSandboxName("");
      mutateSandboxes();
    } catch (err: any) {
      toast.error(err.message || "Failed to create sandbox");
    } finally {
      setIsCreating(false);
    }
  };

  const handleRefreshLinks = () => {
    mutateLinks();
  };

  const handleLinkCreated = () => {
    setShowCreateShareLink(false);
    mutateLinks();
  };

  if (projectLoading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 text-primary-fixed animate-spin" />
      </div>
    );
  }

  if (projectError || !project) {
    return (
      <div className="flex flex-col items-center justify-center h-[60vh] text-center">
        <AlertCircle className="w-12 h-12 text-error mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Project Not Found</h2>
        <p className="text-slate-400">This project doesn't exist or you don't have access.</p>
        <Link href="/projects" className="mt-4 text-primary-fixed hover:underline">
          Back to Projects
        </Link>
      </div>
    );
  }

  const tabs = [
    { key: "overview", label: "Overview", icon: <Folder className="w-4 h-4" /> },
    { key: "sandboxes", label: "Sandboxes", icon: <Box className="w-4 h-4" /> },
    { key: "share", label: "Share Links", icon: <Share2 className="w-4 h-4" /> },
    { key: "members", label: "Members", icon: <Users className="w-4 h-4" /> },
    { key: "settings", label: "Settings", icon: <Settings className="w-4 h-4" /> },
  ];

  return (
    <div className="space-y-6 pb-12">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          <Link
            href="/projects"
            className="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
          </Link>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">{project.name}</h1>
            <p className="text-sm text-slate-400 font-mono">ID: {project.id.slice(0, 8)}...</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Link
            href={`/dashboard?project_id=${project.id}`}
            className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] transition-all"
          >
            <Plus className="w-4 h-4 mr-2" />
            New Sandbox
          </Link>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-1">
        <div className="flex gap-1">
          {tabs.map((tab) => (
            <Tab
              key={tab.key}
              active={activeTab === tab.key}
              onClick={() => setActiveTab(tab.key as typeof activeTab)}
              label={tab.label}
              icon={tab.icon}
            />
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div className="bg-slate-800/30 border border-slate-700/50 rounded-xl overflow-hidden">
        {activeTab === "overview" && (
          <div className="p-6 space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-white">Project Overview</h2>
                <p className="text-slate-400 mt-1">General information and settings</p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5">
                <h3 className="font-semibold text-white mb-3">Description</h3>
                <p className="text-slate-400 whitespace-pre-wrap">{project.description || "No description provided."}</p>
              </div>
              <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5">
                <h3 className="font-semibold text-white mb-3">Details</h3>
                <dl className="space-y-3 text-sm">
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Visibility</dt>
                    <dd className="font-medium text-white">{project.is_public ? "Public" : "Private"}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Created</dt>
                    <dd className="font-medium text-white">{formatDistanceToNow(new Date(project.created_at), { addSuffix: true })}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Last Updated</dt>
                    <dd className="font-medium text-white">{formatDistanceToNow(new Date(project.updated_at), { addSuffix: true })}</dd>
                  </div>
                </dl>
              </div>
            </div>

            <div className="flex gap-3">
              {/* There is no app/(main)/projects/[id]/settings route, so this
                  link was a guaranteed 404. Share links are the only settings
                  surface that exists today. */}
              <a
                href="#share-links"
                className="px-4 py-2 bg-slate-800/50 border border-slate-700 text-slate-300 rounded-xl font-medium hover:bg-slate-800 hover:text-white transition-all"
              >
                <Settings className="w-4 h-4 mr-2 inline" />
                Manage Settings
              </a>
              <button
                onClick={handleDeleteProject}
                className="px-4 py-2 bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl font-medium hover:bg-red-500/20 transition-all"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete Project
              </button>
            </div>
          </div>
        )}

        {activeTab === "sandboxes" && (
          <div className="p-6">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-white">Sandboxes</h2>
                <p className="text-slate-400 mt-1">Development environments in this project</p>
              </div>
              <button
                onClick={() => setShowCreateSandbox(true)}
                className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] transition-all"
              >
                <Plus className="w-4 h-4 mr-2" />
                New Sandbox
              </button>
            </div>

            {sandboxesLoading ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="bg-slate-800/50 border border-slate-700 rounded-xl h-40 animate-pulse" />
                ))}
              </div>
            ) : sandboxes.length === 0 ? (
              <div className="bg-slate-800/30 border border-slate-700/50 rounded-xl py-16 text-center">
                <Box className="w-12 h-12 text-slate-600 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-white mb-2">No sandboxes yet</h3>
                <p className="text-slate-500 mb-4">Create your first development environment</p>
                <button
                  onClick={() => setShowCreateSandbox(true)}
                  className="bg-primary-container text-on-primary-fixed-variant px-5 py-2.5 rounded-xl font-bold"
                >
                  Create Sandbox
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {sandboxes.map((sandbox, idx) => (
                  <motion.div
                    key={sandbox.id}
                    initial={{ opacity: 0, y: 16 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: idx * 0.04, duration: 0.35 }}
                  >
                    <Link
                      href={`/${project.name}/${sandbox.id}`}
                      className="block group h-full"
                    >
                      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5 h-full flex flex-col gap-4 transition-all hover:border-primary-fixed/40">
                        <div className="flex items-start justify-between gap-3">
                          <div className="w-9 h-9 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center shrink-0">
                            <Box className="w-4 h-4 text-primary-fixed" />
                          </div>
                          <div className="flex items-center gap-2">
                            <span className={`px-2 py-0.5 rounded-full text-xs font-bold ${presentState(sandbox.state).color}`}>
                              {presentState(sandbox.state).label}
                            </span>
                          </div>
                        </div>
                        <div className="relative z-10 flex-1 min-w-0">
                          <h3 className="font-bold text-base text-white group-hover:text-primary-fixed transition-colors truncate mb-1.5">
                            {sandbox.name || 'Untitled Sandbox'}
                          </h3>
                          <div className="flex items-center gap-1.5 text-xs text-slate-400 font-mono">
                            <Folder className="w-3.5 h-3.5 shrink-0 text-slate-500" />
                            <span className="truncate">
                              {sandbox.git_url ? sandbox.git_url.replace("https://github.com/", "") : 'No repository'}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center justify-between text-[10px] font-bold text-slate-500 tracking-wider uppercase pt-3 border-t border-slate-700/50">
                          <div className="flex items-center gap-1">
                            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            {sandbox.created_at ? formatDistanceToNow(new Date(sandbox.created_at), { addSuffix: true }) : 'Just now'}
                          </div>
                          <svg className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 text-primary-fixed transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </div>
                      </div>
                    </Link>
                  </motion.div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === "share" && (
          <div className="p-6">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-white">Share Links</h2>
                <p className="text-slate-400 mt-1">Invite collaborators to this project</p>
              </div>
              <button
                onClick={() => setShowCreateShareLink(true)}
                className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] transition-all"
              >
                <Plus className="w-4 h-4 mr-2" />
                Create Share Link
              </button>
            </div>

            <ShareLinkList
              projectId={project.id}
              links={links}
              onRefresh={handleRefreshLinks}
            />
          </div>
        )}

        {activeTab === "members" && (
          <div className="p-6">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-white">Members</h2>
                <p className="text-slate-400 mt-1">Manage project collaborators</p>
              </div>
              <button className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium">
                <Plus className="w-4 h-4 mr-2" />
                Invite Member
              </button>
            </div>
            <div className="bg-slate-800/30 border border-slate-700/50 rounded-xl py-12 text-center">
              <Users className="w-12 h-12 text-slate-600 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-white mb-2">Member management coming soon</h3>
              <p className="text-slate-500">Invite and manage project collaborators with roles</p>
            </div>
          </div>
        )}

        {activeTab === "settings" && (
          <div className="p-6 max-w-2xl">
            <h2 className="text-xl font-bold text-white mb-6">Project Settings</h2>

            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-5 mb-6">
              <h3 className="font-semibold text-white mb-4">General</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">Project Name</label>
                  <input
                    defaultValue={project.name}
                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed transition-all"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">Description</label>
                  <textarea
                    defaultValue={project.description || ""}
                    rows={3}
                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed transition-all resize-none"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <span className="font-medium text-white">Public Project</span>
                    <p className="text-sm text-slate-400">Visible to everyone</p>
                  </div>
                  <input
                    type="checkbox"
                    defaultChecked={project.is_public}
                    className="w-5 h-5 accent-primary-fixed rounded border-slate-600"
                  />
                </div>
              </div>
              <button className="mt-4 px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium">Save Changes</button>
            </div>

            <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-5">
              <h3 className="font-semibold text-red-400 mb-3">Danger Zone</h3>
              <p className="text-slate-400 mb-4">Once deleted, this project and all its sandboxes cannot be recovered.</p>
              <button
                onClick={handleDeleteProject}
                className="px-4 py-2 bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl font-medium hover:bg-red-500/20 transition-all"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Delete Project
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Create Share Link Modal */}
      {showCreateShareLink && (
        <CreateShareLinkModal
          projectId={project.id}
          onClose={() => setShowCreateShareLink(false)}
          onCreated={handleLinkCreated}
        />
      )}

      {/* Create Sandbox Modal */}
      {showCreateSandbox && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
          style={{ backgroundColor: "rgba(0,0,0,0.75)" }}
        >
          <div className="w-full max-w-md bg-slate-900 rounded-2xl border border-slate-700 shadow-2xl overflow-hidden">
            <div className="flex items-center justify-between px-5 py-4 border-b border-slate-700">
              <h3 className="font-semibold text-white">Create Sandbox</h3>
              <button onClick={() => setShowCreateSandbox(false)} className="text-slate-400 hover:text-white">
                <XCircle className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreateSandbox} className="p-5 space-y-4">
              <div>
                <label className="text-sm font-bold tracking-wide text-slate-400 uppercase mb-1.5 block">Sandbox Name</label>
                <input type="text" required minLength={3} placeholder="e.g. Feature Branch" className="w-full bg-slate-800 px-4 py-3 rounded-lg border border-slate-700 text-white focus:border-primary-fixed focus:ring-primary-fixed transition-all outline-none" />
              </div>
              <div className="pt-2 flex justify-end gap-3">
                <button type="button" onClick={() => setShowCreateSandbox(false)} className="px-4 py-2 text-sm font-medium text-slate-400 hover:text-white">Cancel</button>
                <button type="submit" className="px-5 py-2 bg-primary-container text-on-primary-fixed-variant hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] flex items-center gap-2 transition-all rounded-xl">
                  <Plus className="w-4 h-4" />
                  Create Sandbox
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}