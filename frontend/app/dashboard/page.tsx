'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { useEnvStore } from '@/stores/env';
import { EnvironmentList } from '@/components/environment-list';
import { CreateEnvironmentModal } from '@/components/create-env-modal';
import { LogOut, Plus } from 'lucide-react';

export default function DashboardPage() {
  const router = useRouter();
  const { user, isLoading, logout } = useAuthStore();
  const { environments, setEnvironments, isLoading: envLoading, setLoading } = useEnvStore();
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    if (!isLoading && !user) {
      router.push('/login');
    }
  }, [isLoading, user, router]);

  useEffect(() => {
    if (!user) return;
    setLoading(true);
    api.environments.list()
      .then((data) => setEnvironments(data.sandboxes || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [user, setEnvironments, setLoading]);

  if (isLoading || !user) {
    return <div className="flex min-h-screen items-center justify-center">Loading...</div>;
  }

  return (
    <div className="min-h-screen p-8">
      <header className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-gray-400">Welcome, {user.username || user.email}</p>
        </div>
        <div className="flex gap-4">
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 bg-berth-600 hover:bg-berth-700 px-4 py-2 rounded-lg transition"
          >
            <Plus size={18} /> New Environment
          </button>
          <button
            onClick={logout}
            className="flex items-center gap-2 bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg transition"
          >
            <LogOut size={18} /> Logout
          </button>
        </div>
      </header>

      {envLoading ? (
        <p className="text-gray-400">Loading environments...</p>
      ) : (
        <EnvironmentList environments={environments} />
      )}

      {showCreate && <CreateEnvironmentModal onClose={() => setShowCreate(false)} />}
    </div>
  );
}
