# Frontend Architecture

## Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Next.js | 15.0 | App Router, SSR, RSC |
| React | 18.2 | UI Components |
| TypeScript | 5.3 | Type Safety |
| Tailwind CSS | 3.4 | Styling |
| Zustand | 4.5 | State Management |
| Monaco Editor | 0.45 | Code Editor |
| xterm.js | 5.3 | Terminal |
| lucide-react | 0.4 | Icons |
| framer-motion | 11 | Animations |
| date-fns | 3.3 | Date Formatting |
| react-hot-toast | 2.4 | Notifications |

---

## Project Structure

```
frontend/
├── app/                          # Next.js App Router
│   ├── (main)/                   # Authenticated routes
│   │   ├── layout.tsx            # Main layout with nav
│   │   ├── page.tsx              # Dashboard (redirects)
│   │   ├── dashboard/            # Sandbox list
│   │   ├── predictions/          # Prediction UI
│   │   ├── projects/             # Project management
│   │   ├── env/[id]/             # Sandbox IDE
│   │   ├── join/[code]/          # Share link join
│   │   └── profile/              # User profile
│   ├── login/                    # Login page
│   ├── globals.css               # Global styles + Tailwind
│   └── layout.tsx                # Root layout
├── components/                   # Reusable components
│   ├── code-editor/              # Monaco wrapper
│   ├── terminal/                 # xterm.js wrapper
│   ├── git-panel/                # Git operations
│   ├── file-tree/                # File explorer
│   ├── share/                    # Share link components
│   └── presence-bar/             # User presence
├── stores/                       # Zustand stores
│   ├── auth.ts                   # Auth state
│   ├── env.ts                    # Sandbox state
│   └── editor.ts                 # Editor state
├── lib/                          # Utilities
│   ├── api.ts                    # API client
│   └── utils.ts                  # Helpers
├── components/ui/                # Base UI components
└── public/                       # Static assets
```

---

## State Management (Zustand)

### Auth Store (`stores/auth.ts`)
```typescript
interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  setUser: (user: User) => void;
}
```

### Environment Store (`stores/env.ts`)
```typescript
interface EnvState {
  environments: Environment[];
  currentEnv: Environment | null;
  isLoading: boolean;
  error: string | null;
  setEnvironments: (envs: Environment[]) => void;
  setCurrentEnv: (env: Environment) => void;
  updateEnv: (id: string, updates: Partial<Environment>) => void;
  removeEnv: (id: string) => void;
}
```

### Editor Store (`stores/editor.ts`)
```typescript
interface EditorState {
  openFiles: Map<string, FileContent>;
  activeFile: string | null;
  cursorPosition: { line: number, column: number };
  openFile: (path: string, content: string) => void;
  closeFile: (path: string) => void;
  updateFileContent: (path: string, content: string) => void;
}
```

---

## Key Components

### Code Editor (`components/code-editor/CodeEditor.tsx`)
- **Monaco Editor** with TypeScript/JavaScript/Go/Python/Rust support
- **LSP Integration** - hover, completion, diagnostics
- **Multi-tab** - Multiple files via tabs
- **Auto-save** - Debounced save to backend
- **Theme** - Dark theme matching Tailwind

### Terminal (`components/terminal/Terminal.tsx`)
- **xterm.js** with `xterm-addon-fit`, `xterm-addon-web-links`
- **WebSocket** - PTY over NATS/WebSocket
- **Features**: Copy/paste, scrollback, bell, cursor blink
- **Resize** - Fit addon handles container resize

### Git Panel (`components/git-panel/GitPanel.tsx`)
- **Status** - Clean/dirty, ahead/behind
- **Branches** - List, create, checkout, delete
- **Commits** - Log with messages, author, date
- **Diff** - Unified diff view with syntax highlighting
- **Actions** - Stage, commit, push, pull, fetch

### File Tree (`components/file-tree/FileTree.tsx`)
- **Tree View** - Expand/collapse folders
- **Actions** - New file/folder, rename, delete, duplicate
- **Drag & Drop** - Move files (planned)
- **Context Menu** - Right-click actions
- **Search** - Filter by name (planned)

### Preview Proxy (`/p/:id/*`)
- **Next.js Middleware** - Rewrites `/p/:id/*` → `http://sandbox-{id}.domain/*`
- **Path Rewriting** - Strips `/p/{id}` prefix
- **WebSocket Upgrade** - Passes through for terminal

---

## Pages

### Dashboard (`/dashboard`)
- **Sandbox Grid** - Card view with status badges
- **Create Modal** - Git URL, branch, project selection
- **Fork Action** - One-click workspace fork
- **Status Badges** - RUNNING, BUILDING, STOPPED, FAILED
- **Empty State** - CTA to create first sandbox

### Sandbox IDE (`/env/[id]`)
- **Three-Panel Layout** - File tree | Editor | Terminal/Git
- **Resizable Panels** - Drag handles for resize
- **Header** - Sandbox name, status, actions (stop, restart, delete)
- **Tabs** - Editor, Terminal, Git, Preview
- **Auto-reconnect** - WebSocket reconnection logic

### Predictions (`/predictions`)
- **Tabs** - Predict | Models | History
- **Predict Tab** - Dynamic feature inputs per type
- **Models Tab** - Model cards with metrics, retrain/export
- **History Tab** - Table with confidence bars

### Projects (`/projects` & `/projects/[id]`)
- **List** - Card grid with collaborator avatars
- **Detail** - Tabs: Overview, Sandboxes, Share Links, Change Requests
- **Create Modal** - Name, description, org, visibility

### Share Links (`/join/[code]`)
- **Validation** - Check code validity
- **Join Flow** - Accept invite, create fork
- **Expiry/Usage** - Display remaining uses/time

---

## Styling

### Tailwind Configuration
```javascript
// tailwind.config.js
module.exports = {
  darkMode: 'class',
  content: ['./app/**/*.{js,ts,jsx,tsx}', './components/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: { /* ... */ },
        surface: { /* ... */ },
        on: { /* ... */ },
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
}
```

### Design Tokens
- **Colors**: Primary (cyan), Surface (dark grays), On-surface (text)
- **Spacing**: 4px base unit
- **Border Radius**: 8px (cards), 4px (inputs), 16px (modals)
- **Shadows**: Layered elevation system

---

## API Client

### `lib/api.ts`
```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

async function fetchAPI(path: string, options: RequestInit = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...options.headers },
  });
  // Error handling, JSON parsing
}

export const api = {
  auth: { devLogin, logout, me },
  environments: { list, create, get, delete, exec, logs, stop, start, restart, fork },
  files: { list, getContent, updateContent, create, delete, move, duplicate },
  git: { status, branches, createBranch, checkout, pull, commit, push, log, diff },
  projects: { list, create, get, sandboxes },
  orgs: { list, create, members, addMember },
  shareLinks: { create, list, revoke, join, validate },
  changeRequests: { create, list, get, update, merge, close, diff },
  predictions: { predictBuildTime, predictImageSize, predictCacheHit, predictFailureRisk,
                 getHistory, getModelMetrics, retrainModel, exportModel, activateModel },
};
```

---

## Development

```bash
# Install dependencies
npm install

# Development server
npm run dev

# Type checking
npm run type-check

# Linting
npm run lint

# Build
npm run build

# Preview production build
npm start
```

### Environment Variables
```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## Building for Production

```bash
npm run build
# Output: .next/ (static + serverless)
```

### Docker
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./
EXPOSE 3000
CMD ["npm", "start"]
```

---

## Testing

```bash
# Unit tests (Jest + React Testing Library)
npm test

# E2E tests (Playwright)
npm run test:e2e

# Type checking
npm run type-check
```

---

## Performance

- **Code Splitting** - Automatic per-route via App Router
- **Static Generation** - Dashboard, login, static pages
- **Dynamic Routes** - `/env/[id]`, `/projects/[id]` - SSR
- **Image Optimization** - Next.js Image component
- **Font Optimization** - `next/font` with JetBrains Mono
- **Bundle Analysis** - `@next/bundle-analyzer`

---

## Accessibility

- **Semantic HTML** - Proper heading hierarchy, landmarks
- **Keyboard Navigation** - Focus management, skip links
- **ARIA Labels** - Buttons, inputs, custom components
- **Color Contrast** - WCAG AA compliant
- **Focus Indicators** - Visible focus rings
- **Screen Readers** - Live regions for terminal output