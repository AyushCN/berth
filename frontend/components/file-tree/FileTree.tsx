"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  ChevronRight,
  ChevronDown,
  File,
  Folder,
  FileCode,
  FileText,
  Image,
  Database,
  Lock,
  GitBranch,
  Loader2,
  MoreVertical,
  Plus,
  Trash2,
  Edit2,
  Copy,
  Eye,
  ExternalLink,
  Search,
  ChevronLeft,
  ChevronRight as ChevronRightIcon,
  AlertCircle,
} from "lucide-react";
import { api } from "@/lib/api";
import toast from "react-hot-toast";
import { formatDistanceToNow } from "date-fns";

interface FileNode {
  path: string;
  name: string;
  is_directory: boolean;
  children?: FileNode[];
  git_status?: "modified" | "added" | "deleted" | "untracked" | "clean";
}

interface FileTreeNodeProps {
  node: FileNode;
  envId: string;
  selectedPath: string;
  onSelectFile: (path: string) => void;
  depth: number;
  enabled: boolean;
}

interface ContextMenuProps {
  show: boolean;
  onClose: () => void;
  onCreateFile: () => void;
  onCreateFolder: () => void;
  onRename: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
}

function ContextMenu({ show, onClose, onCreateFile, onCreateFolder, onRename, onDuplicate, onDelete }: ContextMenuProps) {
  return (
    <AnimatePresence>
      {show && (
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: -4 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: -4 }}
          className="fixed z-50 bg-slate-900 border border-slate-700 rounded-lg shadow-xl py-1 min-w-[160px]"
          style={{
            top: window.innerHeight / 2,
            left: window.innerWidth / 2,
          }}
        >
          <div className="py-1">
            <button
              onClick={(e) => { e.stopPropagation(); onCreateFile(); onClose(); }}
              className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2"
            >
              <Plus className="w-4 h-4" /> New File
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onCreateFolder(); onClose(); }}
              className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2"
            >
              <Folder className="w-4 h-4" /> New Folder
            </button>
            <div className="border-t border-slate-700 my-1" />
            <button
              onClick={(e) => { e.stopPropagation(); onRename(); onClose(); }}
              className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2"
            >
              <Edit2 className="w-4 h-4" /> Rename
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); onDuplicate(); onClose(); }}
              className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2"
            >
              <Copy className="w-4 h-4" /> Duplicate
            </button>
            <div className="border-t border-slate-700 my-1" />
            <button
              onClick={(e) => { e.stopPropagation(); onDelete(); onClose(); }}
              className="w-full px-3 py-2 text-sm text-red-400 hover:bg-red-500/10 flex items-center gap-2"
            >
              <Trash2 className="w-4 h-4" /> Delete
            </button>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

function FileTreeNode({
  node,
  envId,
  selectedPath,
  onSelectFile,
  depth,
  enabled,
}: FileTreeNodeProps) {
  const [expanded, setExpanded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [children, setChildren] = useState<FileNode[] | null>(
    node.children ?? null
  );
  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const isSelected = selectedPath === node.path;
  const isFile = !node.is_directory;

  const loadChildren = async () => {
    if (!node.is_directory || children !== null) return;
    setLoading(true);
    try {
      const data = await api.files.list(envId, node.path);
      const nodes: FileNode[] = (data.files || []).map((f: any) => ({
        path: f.path,
        name: f.name,
        is_directory: f.is_directory,
        git_status: f.git_status,
        children: f.is_directory ? [] : undefined
      }));
      setChildren(nodes);
    } catch (err) {
      console.error("Failed to load children:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggle = () => {
    if (node.is_directory) {
      setExpanded((prev) => !prev);
      if (!expanded && children === null) {
        loadChildren();
      }
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    if (isFile) {
      onSelectFile(node.path);
    } else {
      handleToggle();
    }
  };

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setShowMenu(true);
  };

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleCreateFile = async () => {
    const name = prompt("Enter file name:");
    if (!name) return;
    try {
      await api.files.create(envId, `${node.path}/${name}`, false);
      setChildren((prev) =>
        prev
          ? [
              ...prev,
              { path: `${node.path}/${name}`, name, is_directory: false },
            ]
          : null
      );
      setExpanded(true);
    } catch (err: any) {
      toast.error(err.message || "Failed to create file");
    }
  };

  const handleCreateFolder = async () => {
    const name = prompt("Enter folder name:");
    if (!name) return;
    try {
      await api.files.create(envId, `${node.path}/${name}`, true);
      setChildren((prev) =>
        prev
          ? [
              ...prev,
              { path: `${node.path}/${name}`, name, is_directory: true, children: [] },
            ]
          : null
      );
      setExpanded(true);
    } catch (err: any) {
      toast.error(err.message || "Failed to create folder");
    }
  };

  const handleRename = async () => {
    const name = prompt("Enter new name:", node.name);
    if (!name || name === node.name) return;
    try {
      await api.files.move(envId, node.path, `${node.path.split("/").slice(0, -1).join("/")}/${name}`);
      setChildren((prev) =>
        prev
          ? prev.map((child) =>
              child.path === node.path ? { ...child, name, path: `${node.path.split("/").slice(0, -1).join("/")}/${name}` } : child
            )
          : null
      );
      toast.success("Renamed successfully");
    } catch (err: any) {
      toast.error(err.message || "Failed to rename");
    }
  };

  const handleDelete = async () => {
    if (!confirm(`Delete ${node.name}?`)) return;
    try {
      await api.files.delete(envId, node.path);
      toast.success("Deleted");
    } catch (err: any) {
      toast.error(err.message || "Failed to delete");
    }
  };

  const handleDuplicate = async () => {
    try {
      await api.files.duplicate(envId, node.path);
      setChildren((prev) =>
        prev
          ? [
              ...prev,
              { path: `${node.path}.copy`, name: `${node.name}.copy`, is_directory: node.is_directory, children: node.is_directory ? [] : undefined },
            ]
          : null
      );
      setExpanded(true);
      toast.success("Duplicated successfully");
    } catch (err: any) {
      toast.error(err.message || "Failed to duplicate");
    }
  };

  const icon = node.is_directory
    ? expanded
      ? fileIcons.folderOpen
      : fileIcons.folder
    : getFileIcon(node.name, false);

  const statusColor = node.git_status
    ? statusColors[node.git_status]
    : "text-slate-500";

  const folderContent = loading ? (
    <div className="px-6 py-2 text-center text-slate-500 text-sm">
      <Loader2 className="w-4 h-4 animate-spin mx-auto mb-1" />
      Loading...
    </div>
  ) : children ? (
    children.map((child) => (
      <FileTreeNode
        key={child.path}
        node={child}
        envId={envId}
        selectedPath={selectedPath}
        onSelectFile={onSelectFile}
        depth={depth + 1}
        enabled={enabled}
      />
    ))
  ) : (
    <div className="px-6 py-4 text-center text-slate-500 text-sm">
      <p>Empty folder</p>
    </div>
  );

  return (
    <div className="relative">
      <div
        ref={menuRef}
        onContextMenu={handleContextMenu}
        onClick={handleClick}
        className={`flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors ${
          isSelected
            ? "bg-primary-container/20 text-primary-fixed"
            : "text-slate-300 hover:bg-slate-800/50"
        } cursor-pointer select-none`}
        style={{ paddingLeft: depth * 16 + 8 }}
      >
        {node.is_directory && (
          <button
            onClick={(e) => { e.stopPropagation(); handleToggle(); }}
            className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800/50 flex-shrink-0"
            aria-label={expanded ? "Collapse" : "Expand"}
          >
            {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRightIcon className="w-4 h-4" />}
          </button>
        )}
        {!node.is_directory && <div className="w-5 flex-shrink-0" />}
        {icon}
        <span className="truncate text-sm font-mono flex-1 min-w-0">
          {node.name}
        </span>
        {node.git_status && (
          <span
            className={`ml-1.5 px-1.5 py-0.5 rounded text-[10px] font-medium ${statusColor}`}
          >
            {node.git_status}
          </span>
        )}
      </div>

      <ContextMenu
        show={showMenu}
        onClose={() => setShowMenu(false)}
        onCreateFile={() => { handleCreateFile(); setShowMenu(false); }}
        onCreateFolder={() => { handleCreateFolder(); setShowMenu(false); }}
        onRename={() => { handleRename(); setShowMenu(false); }}
        onDuplicate={() => { handleDuplicate(); setShowMenu(false); }}
        onDelete={() => { handleDelete(); setShowMenu(false); }}
      />

      {/* Children */}
      <AnimatePresence>
        {expanded && node.is_directory && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            {folderContent}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

const fileIcons = {
  default: <File className="w-4 h-4 text-slate-400" />,
  folder: <Folder className="w-4 h-4 text-amber-400" />,
  folderOpen: <Folder className="w-4 h-4 text-amber-300" />,
  code: <FileCode className="w-4 h-4 text-blue-400" />,
  text: <FileText className="w-4 h-4 text-slate-400" />,
  image: <Image className="w-4 h-4 text-green-400" />,
  db: <Database className="w-4 h-4 text-orange-400" />,
  lock: <Lock className="w-4 h-4 text-red-400" />,
  git: <GitBranch className="w-4 h-4 text-purple-400" />,
};

const getFileIcon = (name: string, isDir: boolean): React.ReactNode => {
  if (isDir) return fileIcons.folder;
  const ext = name.split(".").pop()?.toLowerCase();
  const iconMap: Record<string, React.ReactNode> = {
    ts: <FileCode className="w-4 h-4 text-blue-400" />,
    tsx: <FileCode className="w-4 h-4 text-blue-400" />,
    js: <FileCode className="w-4 h-4 text-yellow-400" />,
    jsx: <FileCode className="w-4 h-4 text-yellow-400" />,
    go: <FileCode className="w-4 h-4 text-cyan-400" />,
    py: <FileCode className="w-4 h-4 text-green-400" />,
    rs: <FileCode className="w-4 h-4 text-orange-400" />,
    java: <FileCode className="w-4 h-4 text-red-400" />,
    cpp: <FileCode className="w-4 h-4 text-pink-400" />,
    c: <FileCode className="w-4 h-4 text-blue-500" />,
    h: <FileCode className="w-4 h-4 text-blue-500" />,
    cs: <FileCode className="w-4 h-4 text-purple-400" />,
    rb: <FileCode className="w-4 h-4 text-red-500" />,
    php: <FileCode className="w-4 h-4 text-purple-400" />,
    swift: <FileCode className="w-4 h-4 text-orange-400" />,
    kt: <FileCode className="w-4 h-4 text-purple-400" />,
    json: <FileText className="w-4 h-4 text-green-400" />,
    yaml: <FileText className="w-4 h-4 text-green-400" />,
    yml: <FileText className="w-4 h-4 text-green-400" />,
    toml: <FileText className="w-4 h-4 text-green-400" />,
    xml: <FileText className="w-4 h-4 text-slate-400" />,
    html: <FileCode className="w-4 h-4 text-orange-400" />,
    css: <FileCode className="w-4 h-4 text-pink-400" />,
    scss: <FileCode className="w-4 h-4 text-pink-400" />,
    sql: <Database className="w-4 h-4 text-orange-400" />,
    md: <FileText className="w-4 h-4 text-slate-400" />,
    txt: <FileText className="w-4 h-4 text-slate-400" />,
    dockerfile: <FileCode className="w-4 h-4 text-blue-400" />,
    sh: <FileCode className="w-4 h-4 text-green-400" />,
    dockerignore: <FileText className="w-4 h-4 text-slate-400" />,
    gitignore: <FileText className="w-4 h-4 text-slate-400" />,
    env: <Lock className="w-4 h-4 text-red-400" />,
    conf: <FileText className="w-4 h-4 text-slate-400" />,
    config: <FileText className="w-4 h-4 text-slate-400" />,
    ini: <FileText className="w-4 h-4 text-slate-400" />,
    tf: <FileCode className="w-4 h-4 text-purple-400" />,
    tfvars: <FileCode className="w-4 h-4 text-purple-400" />,
    hcl: <FileCode className="w-4 h-4 text-purple-400" />,
    proto: <FileCode className="w-4 h-4 text-blue-400" />,
    graphql: <FileCode className="w-4 h-4 text-pink-400" />,
    gql: <FileCode className="w-4 h-4 text-pink-400" />,
  };
  return iconMap[ext || ""] || fileIcons.default;
};

const statusColors: Record<string, string> = {
  modified: "text-yellow-400",
  added: "text-green-400",
  deleted: "text-red-400",
  untracked: "text-purple-400",
  clean: "text-slate-500",
};

interface FileTreeProps {
  envId: string;
  selectedPath: string;
  onSelectFile: (path: string) => void;
  enabled?: boolean;
}

export function FileTree({
  envId,
  selectedPath,
  onSelectFile,
  enabled = true,
}: FileTreeProps) {
  const [rootNodes, setRootNodes] = useState<FileNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTree = useCallback<() => Promise<void>>(async () => {
    if (!envId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.files.list(envId, ".");
      const nodes: FileNode[] = (data.files || []).map((f: any) => ({
        path: f.path,
        name: f.name,
        is_directory: f.is_directory,
        git_status: f.git_status,
        children: f.is_directory ? [] : undefined,
      }));
      setRootNodes(nodes);
    } catch (err: any) {
      setError(err.message || "Failed to load files");
    } finally {
      setLoading(false);
    }
  }, [envId]);

  useEffect(() => {
    if (enabled) {
      void loadTree().catch(console.error);
    }
  }, [loadTree, enabled]);

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center bg-slate-900/50">
        <Loader2 className="w-8 h-8 text-primary-fixed animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full flex flex-col items-center justify-center p-8 bg-slate-900/50">
        <AlertCircle className="w-10 h-10 text-error mb-3" />
        <p className="text-white font-medium mb-2">Failed to load files</p>
        <p className="text-slate-400 text-sm mb-4">{error}</p>
        <button
          onClick={loadTree}
          className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)]"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col bg-slate-900/50">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/50">
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="Search files..."
            className="flex-1 bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-1.5 pl-8 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiM5NWExYjMiIHN0cm9rZS13aWR0aD0iMiI+PGNpcmNsZSBjeD0iMTEiIGN5PSIxMSIgcj0iOCIvPjxwYXRoIGQ9Ik0yMSAyMUwxNi42NSAxNi42NSIvPjwvc3ZnPg==')] bg-no-repeat bg-[left_8px_center]"
          />
          <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="Refresh">
            <Loader2 className="w-4 h-4" />
          </button>
          <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="Collapse all">
            <ChevronLeft className="w-4 h-4" />
          </button>
          <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="New file">
            <Plus className="w-4 h-4" />
          </button>
          <button className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800" title="New folder">
            <Folder className="w-4 h-4" />
          </button>
        </div>

        {/* File Tree */}
        <div className="h-full flex flex-col">
          {rootNodes.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full p-8 bg-slate-900/50">
              <Folder className="w-16 h-16 text-slate-600 mb-4" />
              <p className="text-slate-400">No files in workspace</p>
              <p className="text-slate-500 text-sm mt-1">Create a file or folder to get started</p>
            </div>
          ) : (
            <div className="h-full overflow-y-auto p-2">
              {rootNodes.map((node) => (
                <FileTreeNode
                  key={node.path}
                  node={node}
                  envId={envId}
                  selectedPath={selectedPath}
                  onSelectFile={onSelectFile}
                  depth={0}
                  enabled={enabled}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
