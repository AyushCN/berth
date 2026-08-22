import { useRouter } from 'next/navigation';
import { Play, Trash2, Clock } from 'lucide-react';

interface Environment {
  id: string;
  name: string;
  git_url: string;
  state: string;
  created_at: string;
}

export function EnvironmentList({ environments }: { environments: Environment[] }) {
  const router = useRouter();

  if (environments.length === 0) {
    return (
      <div className="text-center py-16 text-gray-400">
        <p>No environments yet.</p>
        <p className="text-sm">Create one to get started.</p>
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {environments.map((env) => (
        <div
          key={env.id}
          onClick={() => router.push(`/env/${env.id}`)}
          className="bg-gray-800 border border-gray-700 rounded-lg p-4 cursor-pointer hover:border-berth-500 transition group"
        >
          <div className="flex items-start justify-between mb-2">
            <h3 className="font-semibold truncate">{env.name}</h3>
            <span className={`text-xs px-2 py-0.5 rounded ${
              env.state === 'RUNNING' ? 'bg-green-900 text-green-300' :
              env.state === 'BUILDING' ? 'bg-yellow-900 text-yellow-300' :
              'bg-gray-700 text-gray-300'
            }`}>
              {env.state}
            </span>
          </div>
          <p className="text-sm text-gray-400 truncate mb-3">{env.git_url}</p>
          <div className="flex items-center gap-4 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <Clock size={12} />
              {new Date(env.created_at).toLocaleDateString()}
            </span>
            <span className="flex items-center gap-1 group-hover:text-berth-400 transition">
              <Play size={12} /> Open
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}
