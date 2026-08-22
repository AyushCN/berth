'use client';

import { useEffect, useState } from 'react';
import Editor from '@monaco-editor/react';
import { api } from '@/lib/api';

export function CodeEditor({ envId, filePath }: { envId: string; filePath: string }) {
  const [content, setContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setIsLoading(true);
    api.files.getContent(envId, filePath)
      .then((data) => {
        // Response is raw bytes, need to handle properly
        // For now, assume text
        setContent(typeof data === 'string' ? data : JSON.stringify(data));
      })
      .catch(() => setContent('// Failed to load file'))
      .finally(() => setIsLoading(false));
  }, [envId, filePath]);

  const handleSave = (value: string | undefined) => {
    if (!value) return;
    api.files.updateContent(envId, filePath, value).catch(console.error);
  };

  if (isLoading) return <div className="flex items-center justify-center h-full">Loading...</div>;

  const language = filePath.split('.').pop() || 'text';

  return (
    <div className="h-full flex flex-col">
      <div className="bg-gray-800 px-4 py-2 text-sm border-b border-gray-700 flex items-center justify-between">
        <span className="text-gray-300">{filePath}</span>
        <span className="text-xs text-gray-500">Auto-save on Ctrl+S</span>
      </div>
      <div className="flex-1">
        <Editor
          height="100%"
          language={language}
          value={content}
          theme="vs-dark"
          onChange={(value) => setContent(value || '')}
          onMount={(editor) => {
            editor.addCommand(
              // Ctrl+S
              2097,
              () => handleSave(editor.getValue())
            );
          }}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            automaticLayout: true,
          }}
        />
      </div>
    </div>
  );
}
