'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Folder, File, ChevronRight, ChevronDown, Trash2, FolderPlus, FilePlus } from 'lucide-react';
import toast from 'react-hot-toast';

interface FileNode {
  name: string;
  path: string;
  is_dir: boolean;
}

function FileTreeNode({
  envId,
  path,
  name,
  isDir,
  level = 0,
  onSelectFile,
  selectedPath,
  onRefreshParent,
}: {
  envId: string;
  path: string;
  name: string;
  isDir: boolean;
  level?: number;
  onSelectFile: (path: string) => void;
  selectedPath: string;
  onRefreshParent?: () => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [children, setChildren] = useState<FileNode[]>([]);
  const [loaded, setLoaded] = useState(false);

  const loadDirectory = async () => {
    try {
      const data = await api.files.list(envId, path);
      const sorted = (data.files || []).sort((a: FileNode, b: FileNode) => {
        if (a.is_dir && !b.is_dir) return -1;
        if (!a.is_dir && b.is_dir) return 1;
        return a.name.localeCompare(b.name);
      });
      const filtered = sorted.filter((f: FileNode) => !['.git', 'node_modules', '.next', 'dist', '.cache'].includes(f.name));
      setChildren(filtered);
      setLoaded(true);
    } catch (err: any) {
      // Ignore "no rows in result set" or "sandbox is not running" during pending states
      if (err.message && (err.message.includes("no rows") || err.message.includes("not running"))) return;
      console.warn("Failed to load directory", err.message);
    }
  };

  const handleToggle = () => {
    if (!isDir) {
      onSelectFile(path);
      return;
    }
    
    if (!loaded && !isOpen) {
      loadDirectory();
    }
    setIsOpen(!isOpen);
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Are you sure you want to delete "${path}"?`)) return;
    try {
      await api.files.delete(envId, path);
      toast.success('Deleted successfully');
      if (onRefreshParent) onRefreshParent();
      if (selectedPath === path || selectedPath.startsWith(path + '/')) {
        onSelectFile(''); // Clear selection if deleted
      }
    } catch (err: any) {
      toast.error(err.message || 'Failed to delete');
    }
  };

  const handleCreate = async (e: React.MouseEvent, createDir: boolean) => {
    e.stopPropagation();
    const typeStr = createDir ? 'Folder' : 'File';
    const newName = prompt(`Enter name for new ${typeStr} in ${path}:`);
    if (!newName || !newName.trim()) return;
    
    const newPath = path === '.' ? newName.trim() : `${path}/${newName.trim()}`;
    try {
      await api.files.create(envId, newPath, createDir);
      toast.success(`${typeStr} created`);
      if (!isOpen) {
        setIsOpen(true);
      }
      loadDirectory();
    } catch (err: any) {
      toast.error(err.message || `Failed to create ${typeStr}`);
    }
  };

  const isSelected = selectedPath === path;
  
  return (
    <div className="select-none">
      <div
        onClick={handleToggle}
        style={{ paddingLeft: `${level * 12 + 8}px` }}
        className={`group flex items-center justify-between py-1 text-sm cursor-pointer hover:bg-white/5 transition-colors ${
          isSelected && !isDir ? "bg-primary-fixed/10 text-primary-fixed font-medium" : "text-gray-300"
        }`}
      >
        <div className="flex items-center gap-1.5 truncate">
          {isDir ? (
            <>
              <span className="text-gray-500 opacity-60 shrink-0">
                {isOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              </span>
              <Folder size={14} className="text-sky-400 shrink-0" />
            </>
          ) : (
            <>
              <span className="w-3.5 inline-block shrink-0" />
              <File size={14} className={isSelected ? "text-primary-fixed shrink-0" : "text-gray-400 shrink-0"} />
            </>
          )}
          <span className="truncate">{name}</span>
        </div>

        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 pr-2 shrink-0">
          {isDir && (
            <>
              <button onClick={(e) => handleCreate(e, false)} className="p-0.5 text-gray-400 hover:text-white transition-colors" title="New File">
                <FilePlus size={13} />
              </button>
              <button onClick={(e) => handleCreate(e, true)} className="p-0.5 text-gray-400 hover:text-white transition-colors" title="New Folder">
                <FolderPlus size={13} />
              </button>
            </>
          )}
          {path !== '.' && (
            <button onClick={handleDelete} className="p-0.5 text-gray-400 hover:text-red-400 transition-colors" title="Delete">
              <Trash2 size={13} />
            </button>
          )}
        </div>
      </div>
      
      {isDir && isOpen && (
        <div>
          {children.map((child) => (
            <FileTreeNode
              key={child.path}
              envId={envId}
              path={child.path}
              name={child.name}
              isDir={child.is_dir}
              level={level + 1}
              onSelectFile={onSelectFile}
              selectedPath={selectedPath}
              onRefreshParent={loadDirectory}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function FileTree({ envId, selectedPath, onSelectFile }: { envId: string; selectedPath: string; onSelectFile: (path: string) => void }) {
  const [rootFiles, setRootFiles] = useState<FileNode[]>([]);

  const loadRoot = () => {
    api.files.list(envId, '.').then((data) => {
      const sorted = (data.files || []).sort((a: FileNode, b: FileNode) => {
        if (a.is_dir && !b.is_dir) return -1;
        if (!a.is_dir && b.is_dir) return 1;
        return a.name.localeCompare(b.name);
      });
      const filtered = sorted.filter((f: FileNode) => !['.git', 'node_modules', '.next', 'dist', '.cache'].includes(f.name));
      setRootFiles(filtered);
    }).catch((err: any) => {
      if (err.message && (err.message.includes("no rows") || err.message.includes("not running"))) return;
      console.warn("Failed to load root", err.message);
    });
  };

  useEffect(() => {
    loadRoot();
  }, [envId]);

  return (
    <div className="h-full overflow-y-auto py-2 flex flex-col w-full bg-[#18181b] border-r border-white/10 group">
      <div className="flex items-center justify-between px-3 py-2">
        <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Explorer</span>
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button onClick={() => {
            const name = prompt("Enter name for new file:");
            if (name) api.files.create(envId, name, false).then(loadRoot).catch(e => toast.error(e.message));
          }} className="p-1 text-gray-400 hover:text-white" title="New File at Root">
            <FilePlus size={14} />
          </button>
          <button onClick={() => {
            const name = prompt("Enter name for new folder:");
            if (name) api.files.create(envId, name, true).then(loadRoot).catch(e => toast.error(e.message));
          }} className="p-1 text-gray-400 hover:text-white" title="New Folder at Root">
            <FolderPlus size={14} />
          </button>
        </div>
      </div>
      <div className="flex-1 mt-1">
        {rootFiles.map((file) => (
          <FileTreeNode
            key={file.path}
            envId={envId}
            path={file.path}
            name={file.name}
            isDir={file.is_dir}
            level={0}
            onSelectFile={onSelectFile}
            selectedPath={selectedPath}
            onRefreshParent={loadRoot}
          />
        ))}
      </div>
    </div>
  );
}
