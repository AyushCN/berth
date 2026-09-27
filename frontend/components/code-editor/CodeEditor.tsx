"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useEditorStore } from "@/stores/editor";
import { api } from "@/lib/api";
import { Loader2, Save, ChevronDown, Copy, AlertCircle } from "lucide-react";
import dynamic from "next/dynamic";
import toast from "react-hot-toast";

const MonacoEditor = dynamic(
  () => import("@monaco-editor/react").then((mod) => mod.Editor),
  { ssr: false, loading: () => <EditorLoading /> }
);

function EditorLoading() {
  return (
    <div className="h-full flex items-center justify-center bg-slate-950">
      <Loader2 className="w-8 h-8 text-primary-fixed animate-spin" />
    </div>
  );
}

interface CodeEditorProps {
  envId: string;
  filePath: string;
}

export function CodeEditor({ envId, filePath }: CodeEditorProps) {
  const [content, setContent] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [language, setLanguage] = useState<string>("plaintext");
  const [hasChanges, setHasChanges] = useState(false);
  const [originalContent, setOriginalContent] = useState<string>("");

  const editorRef = useRef<any>(null);
  const { openFiles, activeFileId, updateFileContent, markFileChanged, getFile } = useEditorStore();

  // Determine language from file path
  useEffect(() => {
    const ext = filePath.split(".").pop()?.toLowerCase();
    const langMap: Record<string, string> = {
      ts: "typescript",
      tsx: "typescript",
      js: "javascript",
      jsx: "javascript",
      go: "go",
      py: "python",
      rs: "rust",
      java: "java",
      cpp: "cpp",
      c: "c",
      h: "c",
      cs: "csharp",
      rb: "ruby",
      php: "php",
      swift: "swift",
      kt: "kotlin",
      scala: "scala",
      clj: "clojure",
      hs: "haskell",
      ml: "ocaml",
      fs: "fsharp",
      dart: "dart",
      lua: "lua",
      pl: "perl",
      r: "r",
      sh: "shell",
      bash: "shell",
      zsh: "shell",
      fish: "shell",
      ps1: "powershell",
      bat: "batch",
      cmd: "batch",
      dockerfile: "dockerfile",
      json: "json",
      yaml: "yaml",
      yml: "yaml",
      toml: "toml",
      xml: "xml",
      html: "html",
      htm: "html",
      css: "css",
      scss: "scss",
      sass: "sass",
      less: "less",
      sql: "sql",
      md: "markdown",
      txt: "plaintext",
      gitignore: "gitignore",
      dockerignore: "dockerignore",
      env: "dotenv",
      conf: "nginx",
      config: "ini",
      ini: "ini",
      tf: "terraform",
      tfvars: "terraform",
      hcl: "hcl",
      proto: "protobuf",
      graphql: "graphql",
      gql: "graphql",
    };
    setLanguage(filePath.split(".").pop()?.toLowerCase() ? "plaintext" : "plaintext");
  }, [filePath]);

  // Load file content
  const loadFile = useCallback(async () => {
    if (!filePath) return;
    setLoading(true);
    try {
      const content = await api.files.getContent(envId, filePath);
      setContent(content);
      setOriginalContent(content);
      setHasChanges(false);
      setError(null);
    } catch (err: any) {
      setError(err.message || "Failed to load file");
      setContent("");
    } finally {
      setLoading(false);
    }
  }, [envId, filePath]);

  useEffect(() => {
    loadFile();
  }, [loadFile]);

  // Sync with editor store
  useEffect(() => {
    const file = getFile(filePath);
    if (file && file.content !== content) {
      setContent(file.content);
      setOriginalContent(file.content);
    }
  }, [filePath, getFile]);

  const handleEditorChange = (value: string | undefined) => {
    const val = value ?? "";
    setContent(val);
    setHasChanges(val !== originalContent);
  };

  const handleSave = async () => {
    if (!hasChanges) return;
    setSaving(true);
    try {
      await api.files.updateContent(envId, filePath, content);
      setOriginalContent(content);
      setHasChanges(false);
      toast.success("File saved");
    } catch (err: any) {
      setError(err.message || "Failed to save file");
    } finally {
      setSaving(false);
    }
  };

  const handleEditorMount = (editor: any, monaco: any) => {
    editor.updateOptions({
      lineNumbers: "on",
      renderLineHighlight: "all",
      fontSize: 14,
      fontFamily: '"JetBrains Mono", "Fira Code", "Monaco", "Consolas", monospace',
      lineHeight: 22,
      tabSize: 2,
      insertSpaces: true,
      wordWrap: "on",
      minimap: { enabled: true, maxColumn: 120 },
      bracketPairColorization: { enabled: true },
      guides: { bracketPairs: true },
      renderWhitespace: "selection",
      renderControlCharacters: true,
      smoothScrolling: true,
      cursorBlinking: "smooth",
      cursorSmoothCaretAnimation: "on",
    });

    editor.onDidChangeModelContent((e: any) => {
      const newContent = editor.getValue();
      if (newContent !== content) {
        setContent(newContent);
        setHasChanges(newContent !== originalContent);
      }
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      handleSave();
    });
  };

  const ErrorDisplay = () => (
    <div className="h-full flex flex-col">
      <div className="h-10 px-4 border-b border-slate-700 bg-slate-900/50 flex items-center gap-2">
        <span className="text-sm font-medium text-white truncate">{filePath.split("/").pop()}</span>
        <span className="px-2 py-0.5 rounded text-xs bg-red-500/10 text-red-400 border border-red-500/20">
          Error loading file
        </span>
      </div>
      <div className="h-full flex items-center justify-center bg-slate-950 p-8">
        <div className="text-center">
          <AlertCircle className="w-12 h-12 text-error mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-white mb-2">Failed to Load File</h3>
          <p className="text-slate-400 text-center mb-4 max-w-md">{error}</p>
          <button
            onClick={() => loadFile()}
            className="px-4 py-2 bg-primary-container text-on-primary-fixed-variant rounded-xl font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)]"
          >
            Retry
          </button>
        </div>
      </div>
    </div>
  );

  if (loading) {
    return <EditorLoading />;
  }

  if (error && !content) {
    return <ErrorDisplay />;
  }

  return (
    <div className="h-full flex flex-col bg-slate-950">
      {/* Editor Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/50">
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-slate-400 truncate max-w-[200px]">{filePath}</span>
          {hasChanges && (
            <span className="px-1.5 py-0.5 rounded text-xs bg-amber-500/10 text-amber-400 border border-amber-500/20">
              Modified
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            className="bg-slate-800/50 border border-slate-700 rounded-lg px-2 py-1 text-xs text-white focus:outline-none focus:ring-2 focus:ring-primary-fixed/20 focus:border-primary-fixed"
          >
            <option value="plaintext">Plain Text</option>
            <option value="typescript">TypeScript</option>
            <option value="javascript">JavaScript</option>
            <option value="go">Go</option>
            <option value="python">Python</option>
            <option value="rust">Rust</option>
            <option value="java">Java</option>
            <option value="cpp">C++</option>
            <option value="c">C</option>
            <option value="csharp">C#</option>
            <option value="ruby">Ruby</option>
            <option value="php">PHP</option>
            <option value="swift">Swift</option>
            <option value="kotlin">Kotlin</option>
            <option value="scala">Scala</option>
            <option value="json">JSON</option>
            <option value="yaml">YAML</option>
            <option value="toml">TOML</option>
            <option value="xml">XML</option>
            <option value="html">HTML</option>
            <option value="css">CSS</option>
            <option value="scss">SCSS</option>
            <option value="sql">SQL</option>
            <option value="markdown">Markdown</option>
            <option value="dockerfile">Dockerfile</option>
            <option value="shell">Shell</option>
            <option value="plaintext">Plain Text</option>
          </select>
          <button
            onClick={() => {
              navigator.clipboard.writeText(content);
              toast.success("Copied!");
            }}
            className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-800"
            title="Copy content"
          >
            <Copy className="w-4 h-4" />
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className="px-3 py-1.5 bg-primary-container text-on-primary-fixed-variant rounded-lg font-medium hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-[0.98] transition-all disabled:opacity-50 flex items-center justify-center gap-1.5"
          >
            <Save className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Save</span>
            {saving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
          </button>
        </div>
      </div>

      <div className="h-full relative">
        <MonacoEditor
          height="100%"
          defaultLanguage={language}
          value={content}
          onChange={handleEditorChange}
          onMount={handleEditorMount}
          options={{
                        
            lineNumbers: "on",
            renderLineHighlight: "all",
            fontSize: 14,
            fontFamily: '"JetBrains Mono", "Fira Code", "Monaco", "Consolas", monospace',
            lineHeight: 22,
            tabSize: 2,
            insertSpaces: true,
            wordWrap: "on",
            minimap: { enabled: true, maxColumn: 120 },
            bracketPairColorization: { enabled: true },
            guides: { bracketPairs: true },
            renderWhitespace: "selection",
            renderControlCharacters: true,
            smoothScrolling: true,
            cursorBlinking: "smooth",
            cursorSmoothCaretAnimation: "on",
            automaticLayout: true,
          }}
          theme="vs-dark"
        />
      </div>
    </div>
  );
}