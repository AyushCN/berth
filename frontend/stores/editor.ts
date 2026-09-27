"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

interface OpenFile {
  id: string;
  path: string;
  name: string;
  content: string;
  language: string;
  hasChanges: boolean;
  cursorPosition?: { line: number; column: number };
  scrollPosition?: { top: number; left: number };
}

interface EditorState {
  openFiles: OpenFile[];
  activeFileId: string | null;
  setActiveFile: (path: string) => void;
  openFile: (file: Omit<OpenFile, "hasChanges">) => void;
  closeFile: (path: string) => void;
  updateFileContent: (path: string, content: string) => void;
  markFileChanged: (path: string, hasChanges: boolean) => void;
  updateCursorPosition: (path: string, position: { line: number; column: number }) => void;
  updateScrollPosition: (path: string, position: { top: number; left: number }) => void;
  closeAllFiles: () => void;
  getFile: (path: string) => OpenFile | undefined;
}

const getLanguageFromPath = (path: string): string => {
  const ext = path.split(".").pop()?.toLowerCase();
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
    m: "objective-c",
    mm: "objective-cpp",
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
  return langMap[ext || ""] || "plaintext";
};

export const useEditorStore = create<EditorState>()(
  persist(
    (set, get) => ({
      openFiles: [],
      activeFileId: null,

      setActiveFile: (path: string) => {
        const file = get().openFiles.find((f) => f.path === path);
        if (file) {
          set({ activeFileId: path });
        }
      },

      openFile: (file) => {
        const existing = get().openFiles.find((f) => f.path === file.path);
        if (existing) {
          set({ activeFileId: file.path });
          return;
        }

        const newFile: OpenFile = {
          ...file,
          language: getLanguageFromPath(file.path),
          hasChanges: false,
        };

        set((state) => ({
          openFiles: [...state.openFiles, newFile],
          activeFileId: file.path,
        }));
      },

      closeFile: (path: string) => {
        const { openFiles, activeFileId } = get();
        const index = openFiles.findIndex((f) => f.path === path);
        if (index === -1) return;

        const newFiles = openFiles.filter((f) => f.path !== path);
        let newActive = activeFileId;

        if (activeFileId === path) {
          if (index < newFiles.length) {
            newActive = newFiles[index].path;
          } else if (newFiles.length > 0) {
            newActive = newFiles[newFiles.length - 1].path;
          } else {
            newActive = null;
          }
        }

        set({ openFiles: newFiles, activeFileId: newActive });
      },

      updateFileContent: (path: string, content: string) => {
        set((state) => ({
          openFiles: state.openFiles.map((f) =>
            f.path === path ? { ...f, content } : f
          ),
        }));
      },

      markFileChanged: (path: string, hasChanges: boolean) => {
        set((state) => ({
          openFiles: state.openFiles.map((f) =>
            f.path === path ? { ...f, hasChanges } : f
          ),
        }));
      },

      updateCursorPosition: (path: string, position: { line: number; column: number }) => {
        set((state) => ({
          openFiles: state.openFiles.map((f) =>
            f.path === path ? { ...f, cursorPosition: position } : f
          ),
        }));
      },

      updateScrollPosition: (path: string, position: { top: number; left: number }) => {
        set((state) => ({
          openFiles: state.openFiles.map((f) =>
            f.path === path ? { ...f, scrollPosition: position } : f
          ),
        }));
      },

      closeAllFiles: () => {
        set({ openFiles: [], activeFileId: null });
      },

      getFile: (path: string) => {
        return get().openFiles.find((f) => f.path === path);
      },
    }),
    {
      name: "berth-editor-store",
      partialize: (state) => ({
        openFiles: state.openFiles,
        activeFileId: state.activeFileId,
      }),
    }
  )
);