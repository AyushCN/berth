const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function fetchAPI(path: string, options: RequestInit = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!res.ok) {
    const text = await res.text();
    let errMsg = text;
    try {
      const parsed = JSON.parse(text);
      if (parsed.error) errMsg = parsed.error;
    } catch (e) {}
    throw new APIError(res.status, errMsg);
  }

  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  auth: {
    devLogin: () => fetchAPI('/api/auth/dev-login'),
    logout: () => fetchAPI('/api/auth/logout', { method: 'POST' }),
    me: async () => {
      const response = await fetchAPI('/api/user/me');
      // Berth's API wraps the profile in `{ user }`; keep callers working with
      // the User object itself, as the other authenticated pages expect.
      return response?.user ?? response;
    },
  },
  environments: {
    list: () => fetchAPI('/api/environments'),
    create: (data: { name: string; git_url: string; git_branch: string; project_id?: string }) =>
      fetchAPI('/api/environments', { method: 'POST', body: JSON.stringify(data) }),
    fork: (id: string, data: { name: string; project_id?: string }) =>
      fetchAPI(`/api/environments/${id}/fork`, { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => fetchAPI(`/api/environments/${id}`),
    delete: (id: string) => fetchAPI(`/api/environments/${id}`, { method: 'DELETE' }),
    exec: (id: string, command: string[]) =>
      fetchAPI(`/api/environments/${id}/exec`, { method: 'POST', body: JSON.stringify({ command }) }),
    logs: (id: string) => fetchAPI(`/api/environments/${id}/logs`),
    stop: (id: string) => fetchAPI(`/api/environments/${id}/stop`, { method: 'POST' }),
    start: (id: string) => fetchAPI(`/api/environments/${id}/start`, { method: 'POST' }),
    restart: (id: string) => fetchAPI(`/api/environments/${id}/restart`, { method: 'POST' }),
  },
  git: {
    status: (id: string) => fetchAPI(`/api/environments/${id}/git/status`),
    branches: (id: string) => fetchAPI(`/api/environments/${id}/git/branches`),
    createBranch: (id: string, branch: string) => fetchAPI(`/api/environments/${id}/git/branch`, { method: 'POST', body: JSON.stringify({ branch }) }),
    checkout: (id: string, branch: string, force?: boolean) => fetchAPI(`/api/environments/${id}/git/checkout`, { method: 'POST', body: JSON.stringify({ branch, force }) }),
    pull: (id: string) => fetchAPI(`/api/environments/${id}/git/pull`, { method: 'POST' }),
    commit: (id: string, message: string) => fetchAPI(`/api/environments/${id}/git/commit`, { method: 'POST', body: JSON.stringify({ message }) }),
    push: (id: string) => fetchAPI(`/api/environments/${id}/git/push`, { method: 'POST' }),
    log: (id: string) => fetchAPI(`/api/environments/${id}/git/log`),
    diff: (id: string, filePath: string) => fetchAPI(`/api/environments/${id}/git/diff?file=${encodeURIComponent(filePath)}`),
  },
  files: {
    list: (id: string, path: string = '.') =>
      fetchAPI(`/api/environments/${id}/files?path=${encodeURIComponent(path)}`),
    getContent: async (id: string, path: string) => {
      const res = await fetch(`${API_BASE}/api/environments/${id}/files/content?path=${encodeURIComponent(path)}`, { credentials: 'include' });
      if (!res.ok) {
        const text = await res.text();
        let message = text || `Request failed (${res.status})`;
        try {
          const parsed = JSON.parse(text);
          if (parsed.error) message = parsed.error;
        } catch {}
        throw new APIError(res.status, message);
      }
      return res.text();
    },
    updateContent: (id: string, path: string, content: string) =>
      fetchAPI(`/api/environments/${id}/files/content?path=${encodeURIComponent(path)}`, {
        method: 'PUT',
        body: content,
        headers: { 'Content-Type': 'application/octet-stream' },
      }),
    create: (id: string, path: string, is_dir: boolean) =>
      fetchAPI(`/api/environments/${id}/files/create`, {
        method: 'POST',
        body: JSON.stringify({ path, is_dir }),
      }),
    delete: (id: string, path: string) =>
      fetchAPI(`/api/environments/${id}/files/delete`, {
        method: 'POST',
        body: JSON.stringify({ path }),
      }),
    move: (id: string, path: string, newPath: string) =>
      fetchAPI(`/api/environments/${id}/files/move`, {
        method: 'POST',
        body: JSON.stringify({ path, new_path: newPath }),
      }),
    duplicate: (id: string, path: string) =>
      fetchAPI(`/api/environments/${id}/files/duplicate`, {
        method: 'POST',
        body: JSON.stringify({ path }),
      }),
  },
  orgs: {
    list: () => fetchAPI('/api/orgs'),
    create: (data: { name: string }) => fetchAPI('/api/orgs', { method: 'POST', body: JSON.stringify(data) }),
    members: (id: string) => fetchAPI(`/api/orgs/${id}/members`),
    addMember: (id: string, data: { user_id: string; role: string }) => fetchAPI(`/api/orgs/${id}/members`, { method: 'POST', body: JSON.stringify(data) }),
  },
  projects: {
    list: () => fetchAPI('/api/projects'),
    listForOrg: (orgId: string) => fetchAPI(`/api/orgs/${orgId}/projects`),
    create: (data: { name: string; description?: string; owner_organization_id: string; is_public: boolean }) =>
      fetchAPI('/api/projects', { method: 'POST', body: JSON.stringify(data) }),
    sandboxes: (id: string) => fetchAPI(`/api/projects/${id}/sandboxes`),
    delete: (id: string) => fetchAPI(`/api/projects/${id}`, { method: 'DELETE' }),
    shareLinks: {
      create: (projectId: string, data: { role: 'VIEWER' | 'EDITOR'; expires_at?: string; max_uses?: number }) =>
        fetchAPI(`/api/projects/${projectId}/share-links`, { method: 'POST', body: JSON.stringify(data) }),
      list: (projectId: string) => fetchAPI(`/api/projects/${projectId}/share-links`),
      revoke: (projectId: string, linkId: string) => fetchAPI(`/api/projects/${projectId}/share-links/${linkId}`, { method: 'DELETE' }),
      join: (code: string) => fetchAPI('/api/join', { method: 'POST', body: JSON.stringify({ code }) }),
      validate: (code: string) => fetchAPI(`/api/share-links/validate?code=${encodeURIComponent(code)}`),
    },
  },
  predictions: {
    // Build time prediction
    predictBuildTime: (workspaceId: string, features: Record<string, any>) =>
      fetchAPI('/api/predictions/build-time', {
        method: 'POST',
        body: JSON.stringify({ workspace_id: workspaceId, features }),
      }),

    // Image size prediction
    predictImageSize: (workspaceId: string, features: Record<string, any>) =>
      fetchAPI('/api/predictions/image-size', {
        method: 'POST',
        body: JSON.stringify({ workspace_id: workspaceId, features }),
      }),

    // Cache hit prediction
    predictCacheHit: (workspaceId: string, features: Record<string, any>) =>
      fetchAPI('/api/predictions/cache-hit', {
        method: 'POST',
        body: JSON.stringify({ workspace_id: workspaceId, features }),
      }),

    // Failure risk prediction
    predictFailureRisk: (workspaceId: string, features: Record<string, any>) =>
      fetchAPI('/api/predictions/failure-risk', {
        method: 'POST',
        body: JSON.stringify({ workspace_id: workspaceId, features }),
      }),

    // Get prediction history
    getHistory: (workspaceId: string, type: string, limit?: number, offset?: number) => {
      const params = new URLSearchParams();
      params.set('workspace_id', workspaceId);
      params.set('type', type);
      if (limit) params.set('limit', limit.toString());
      if (offset) params.set('offset', offset.toString());
      return fetchAPI(`/api/predictions/history?${params.toString()}`);
    },

    // Get model metrics
    getModelMetrics: (type: string) =>
      fetchAPI(`/api/predictions/models/metrics?type=${encodeURIComponent(type)}`),

    // Retrain model
    retrainModel: (type: string, algorithm: string) =>
      fetchAPI('/api/predictions/models/retrain', {
        method: 'POST',
        body: JSON.stringify({ type, algorithm }),
      }),

    // Export model to ONNX
    exportModel: (modelId: string) =>
      fetchAPI('/api/predictions/models/export', {
        method: 'POST',
        body: JSON.stringify({ model_id: modelId }),
      }),

    // Activate model
    activateModel: (modelId: string) =>
      fetchAPI('/api/predictions/models/activate', {
        method: 'POST',
        body: JSON.stringify({ model_id: modelId }),
      }),
  },
};
