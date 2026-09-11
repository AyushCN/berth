'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { useEnvStore } from '@/stores/env';
import { X, GitBranch, Link as LinkIcon, Loader2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

export function CreateEnvironmentModal({ onClose, projectId }: { onClose: () => void, projectId?: string | null }) {
  const [name, setName] = useState('');
  const [gitUrl, setGitUrl] = useState('');
  const [gitBranch, setGitBranch] = useState('main');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isFetchingBranches, setIsFetchingBranches] = useState(false);
  const [availableBranches, setAvailableBranches] = useState<string[]>([]);
  const { addEnvironment } = useEnvStore();

  useEffect(() => {
    // Attempt to extract owner and repo from standard GitHub URLs
    const githubRegex = /github\.com\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_-]+)(\.git)?/;
    const match = gitUrl.match(githubRegex);

    if (match) {
      const owner = match[1];
      const repo = match[2];
      setIsFetchingBranches(true);
      setAvailableBranches([]);

      fetch(`https://api.github.com/repos/${owner}/${repo}/branches`)
        .then(res => {
          if (!res.ok) throw new Error('Failed to fetch');
          return res.json();
        })
        .then(data => {
          if (Array.isArray(data)) {
            const branches = data.map((b: any) => b.name);
            setAvailableBranches(branches);
            if (branches.length > 0 && !branches.includes(gitBranch)) {
              // If default 'main' isn't there, pick the first one
              setGitBranch(branches.includes('master') ? 'master' : branches[0]);
            }
          }
        })
        .catch(() => {
          // Silent catch - fallback to text input
          setAvailableBranches([]);
        })
        .finally(() => {
          setIsFetchingBranches(false);
        });
    } else {
      setAvailableBranches([]);
      setIsFetchingBranches(false);
    }
  }, [gitUrl]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const payload: any = { name, git_url: gitUrl, git_branch: gitBranch };
      if (projectId) {
        payload.project_id = projectId;
      }
      const env = await api.environments.create(payload);
      addEnvironment(env);
      onClose();
    } catch (err) {
      console.error(err);
      alert('Failed to create environment');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AnimatePresence>
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 20 }}
          className="bg-surface-container-low border border-outline-variant rounded-2xl w-full max-w-md shadow-2xl overflow-hidden"
        >
          <div className="flex items-center justify-between p-6 border-b border-outline-variant bg-surface-container/50">
            <h2 className="text-xl font-bold tracking-tight text-on-surface">New Environment</h2>
            <button 
              onClick={onClose} 
              className="text-on-surface-variant hover:text-on-surface hover:bg-surface-variant/50 p-2 rounded-full transition-colors"
            >
              <X size={20} />
            </button>
          </div>
          <form onSubmit={handleSubmit} className="p-6 space-y-5">
            <div>
              <label className="block text-sm font-medium text-on-surface-variant mb-1.5">Environment Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-surface-container border border-outline-variant rounded-xl px-4 py-3 text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                placeholder="e.g. my-awesome-project"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-on-surface-variant mb-1.5">Git Repository URL</label>
              <div className="relative">
                <LinkIcon className="absolute left-3.5 top-1/2 -translate-y-1/2 text-on-surface-variant w-4 h-4" />
                <input
                  type="url"
                  value={gitUrl}
                  onChange={(e) => setGitUrl(e.target.value)}
                  className="w-full bg-surface-container border border-outline-variant rounded-xl pl-10 pr-4 py-3 text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                  placeholder="https://github.com/user/repo"
                  required
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-on-surface-variant mb-1.5">Branch</label>
              <div className="relative">
                <GitBranch className="absolute left-3.5 top-1/2 -translate-y-1/2 text-on-surface-variant w-4 h-4" />
                
                {isFetchingBranches ? (
                  <div className="w-full bg-surface-container border border-outline-variant rounded-xl pl-10 pr-4 py-3 flex items-center gap-2 text-on-surface-variant">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Fetching branches...</span>
                  </div>
                ) : availableBranches.length > 0 ? (
                  <select
                    value={gitBranch}
                    onChange={(e) => setGitBranch(e.target.value)}
                    className="w-full bg-surface-container border border-outline-variant rounded-xl pl-10 pr-4 py-3 text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all appearance-none"
                  >
                    {availableBranches.map((branch) => (
                      <option key={branch} value={branch}>
                        {branch}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    value={gitBranch}
                    onChange={(e) => setGitBranch(e.target.value)}
                    className="w-full bg-surface-container border border-outline-variant rounded-xl pl-10 pr-4 py-3 text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                    placeholder="main"
                  />
                )}
              </div>
              <p className="text-xs text-on-surface-variant/70 mt-2">
                {availableBranches.length > 0 
                  ? `Successfully found ${availableBranches.length} branch(es).` 
                  : "If this is a private repo, type the branch name manually."}
              </p>
            </div>
            <div className="pt-2">
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full bg-primary text-on-primary font-medium hover:bg-primary/90 disabled:opacity-50 py-3 rounded-xl transition-colors flex items-center justify-center gap-2"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Creating...
                  </>
                ) : (
                  'Create Environment'
                )}
              </button>
            </div>
          </form>
        </motion.div>
      </div>
    </AnimatePresence>
  );
}
