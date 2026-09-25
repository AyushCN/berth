'use client';

import { useCallback, useEffect, useState, useRef } from 'react';
import Editor, { OnMount } from '@monaco-editor/react';
import { api } from '@/lib/api';
import { Save } from 'lucide-react';
import toast from 'react-hot-toast';

const getLanguageFromPath = (path: string): string => {
  const ext = path.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'js':
    case 'jsx':
      return 'javascript';
    case 'ts':
    case 'tsx':
      return 'typescript';
    case 'json':
      return 'json';
    case 'py':
      return 'python';
    case 'go':
      return 'go';
    case 'md':
      return 'markdown';
    case 'html':
      return 'html';
    case 'css':
      return 'css';
    default:
      return 'plaintext';
  }
};

export function CodeEditor({ envId, filePath }: { envId: string; filePath: string }) {
  const [content, setContent] = useState('');
  const [originalContent, setOriginalContent] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [loadError, setLoadError] = useState('');
  const editorRef = useRef<any>(null);

  const loadFile = useCallback(async () => {
    setIsLoading(true);
    setLoadError('');
    try {
      const data = await api.files.getContent(envId, filePath);
      const text = typeof data === 'string' ? data : JSON.stringify(data);
      setContent(text);
      setOriginalContent(text);
    } catch (err: any) {
      setLoadError(err.message || 'Could not load this file.');
    } finally {
      setIsLoading(false);
    }
  }, [envId, filePath]);

  useEffect(() => {
    void loadFile();
  }, [loadFile]);

  const handleSave = async () => {
    if (!editorRef.current) return;
    const value = editorRef.current.getValue();
    setIsSaving(true);
    try {
      const result = await api.files.updateContent(envId, filePath, value);
      setOriginalContent(value);
      toast.success(result?.reloadSignaled ? 'Saved — container reload signaled' : 'Saved');
    } catch (err: any) {
      toast.error(err.message || 'Failed to save');
      console.warn(err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    
    // Add Ctrl+S / Cmd+S save action
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      handleSave();
    });
  };

  if (isLoading) {
    return <div className="flex items-center justify-center h-full text-white/40">Loading {filePath}...</div>;
  }
  if (loadError) {
    return <div className="flex h-full flex-col items-center justify-center gap-3 text-sm text-red-300"><p>{loadError}</p><button onClick={() => void loadFile()} className="underline underline-offset-2">Retry</button></div>;
  }

  const language = getLanguageFromPath(filePath);
  const isDirty = content !== originalContent;

  return (
    <div className="h-full flex flex-col bg-[#1e1e1e]">
      <div className="flex items-center justify-between px-4 py-2 border-b border-white/10 bg-[#252526]">
        <div className="flex items-center gap-2">
          <span className={`text-sm ${isDirty ? "text-yellow-400 font-medium" : "text-gray-300"}`}>
            {filePath} {isDirty && "●"}
          </span>
          {isDirty && <span className="text-xs text-white/40 ml-2">Unsaved changes</span>}
        </div>
        <button
          onClick={handleSave}
          disabled={!isDirty || isSaving}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded transition-colors ${
            !isDirty
              ? "text-white/30 bg-white/5 cursor-not-allowed" 
              : isSaving 
                ? "text-primary-fixed bg-primary-fixed/20 animate-pulse" 
                : "text-white bg-primary-fixed hover:bg-primary-fixed/80"
          }`}
        >
          <Save size={14} />
          {isSaving ? "Saving..." : "Save"}
        </button>
      </div>
      
      <div className="flex-1 relative">
        <Editor
          height="100%"
          width="100%"
          theme="vs-dark"
          path={filePath} // helps monaco with syntax checking
          language={language}
          value={content}
          onChange={(value) => setContent(value || '')}
          onMount={handleEditorDidMount}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            fontFamily: "'JetBrains Mono', 'Fira Code', 'Menlo', 'Monaco', monospace",
            wordWrap: 'on',
            scrollBeyondLastLine: false,
            smoothScrolling: true,
            padding: { top: 16 },
            automaticLayout: true,
          }}
        />
      </div>
    </div>
  );
}
