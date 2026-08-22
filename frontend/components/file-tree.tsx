'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Folder, File, ChevronRight, ChevronDown } from 'lucide-react';

interface FileNode {
  name: string;
  path: string;
  is_dir: boolean;
}

export function FileTree({ envId, onSelectFile }: { envId: string; onSelectFile: (path: string) => void }) {
  const [files, setFiles] = useState<FileNode[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['.']));
  const [currentPath, setCurrentPath] = useState('.');

  useEffect(() => {
    api.files.list(envId, currentPath).then((data) => {
      setFiles(data.files || []);
    }).catch(console.error);
  }, [envId, currentPath]);

  const toggleDir = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  return (
    <div className="p-2">
      <h3 className="text-xs font-semibold text-gray-500 uppercase mb-2 px-2">Files</h3>
      {files.map((file) => (
        <div key={file.path} className="select-none">
          {file.is_dir ? (
            <button
              onClick={() => toggleDir(file.path)}
              className="flex items-center gap-1 w-full px-2 py-1 text-sm hover:bg-gray-700 rounded"
            >
              {expanded.has(file.path) ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              <Folder size={14} className="text-yellow-500" />
              {file.name}
            </button>
          ) : (
            <button
              onClick={() => onSelectFile(file.path)}
              className="flex items-center gap-1 w-full px-2 py-1 text-sm hover:bg-gray-700 rounded pl-7"
            >
              <File size={14} className="text-gray-400" />
              {file.name}
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
