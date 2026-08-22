'use client';

import { useState } from 'react';
import { api } from '@/lib/api';
import { useEnvStore } from '@/stores/env';
import { X } from 'lucide-react';

export function CreateEnvironmentModal({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState('');
  const [gitUrl, setGitUrl] = useState('');
  const [gitBranch, setGitBranch] = useState('main');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { addEnvironment } = useEnvStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const env = await api.environments.create({ name, git_url: gitUrl, git_branch: gitBranch });
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
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-gray-800 border border-gray-700 rounded-lg w-full max-w-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold">New Environment</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X size={20} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 focus:border-berth-500 outline-none"
              placeholder="my-project"
              required
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Git URL</label>
            <input
              type="url"
              value={gitUrl}
              onChange={(e) => setGitUrl(e.target.value)}
              className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 focus:border-berth-500 outline-none"
              placeholder="https://github.com/user/repo.git"
              required
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Branch</label>
            <input
              type="text"
              value={gitBranch}
              onChange={(e) => setGitBranch(e.target.value)}
              className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 focus:border-berth-500 outline-none"
              placeholder="main"
            />
          </div>
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full bg-berth-600 hover:bg-berth-700 disabled:opacity-50 py-2 rounded-lg transition"
          >
            {isSubmitting ? 'Creating...' : 'Create Environment'}
          </button>
        </form>
      </div>
    </div>
  );
}
