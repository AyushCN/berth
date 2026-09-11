'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { api } from '@/lib/api';
import { FileTree } from '@/components/file-tree';
import { CodeEditor } from '@/components/code-editor';
import { Terminal } from '@/components/terminal';
import { useEnvStore } from '@/stores/env';

export default function EnvironmentPage() {
  const { id } = useParams<{ id: string }>();
  const { selectEnvironment } = useEnvStore();
  const [env, setEnv] = useState<any>(null);
  const [activeFile, setActiveFile] = useState<string | null>(null);

  useEffect(() => {
    selectEnvironment(id);
    api.environments.get(id).then(setEnv).catch(console.error);
  }, [id, selectEnvironment]);

  if (!env) return <div className="flex min-h-screen items-center justify-center">Loading...</div>;

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-gray-800 border-b border-gray-700 px-4 py-3 flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">{env.name}</h1>
          <p className="text-xs text-gray-400">{env.git_url} • {env.state}</p>
        </div>
        <div className="flex gap-2">
          <span className={`px-2 py-1 rounded text-xs ${
            env.state === 'RUNNING' ? 'bg-green-900 text-green-300' :
            env.state === 'BUILDING' ? 'bg-yellow-900 text-yellow-300' :
            'bg-gray-700 text-gray-300'
          }`}>
            {env.state}
          </span>
        </div>
      </header>

      <div className="flex-1 flex overflow-hidden">
        <aside className="w-64 bg-gray-800 border-r border-gray-700 overflow-y-auto">
          <FileTree envId={id} onSelectFile={setActiveFile} />
        </aside>

        <main className="flex-1 flex flex-col min-w-0">
          <div className="flex-1 overflow-hidden">
            {activeFile ? (
              <CodeEditor envId={id} filePath={activeFile} />
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                Select a file to edit
              </div>
            )}
          </div>
          <div className="h-48 border-t border-gray-700">
            <Terminal envId={id} />
          </div>
        </main>
      </div>
    </div>
  );
}
