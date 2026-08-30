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
    throw new APIError(res.status, text);
  }

  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  auth: {
    devLogin: () => fetchAPI('/api/auth/dev-login'),
    me: () => fetchAPI('/api/user/me'),
  },
  environments: {
    list: () => fetchAPI('/api/environments'),
    create: (data: { name: string; git_url: string; git_branch: string }) =>
      fetchAPI('/api/environments', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => fetchAPI(`/api/environments/${id}`),
    delete: (id: string) => fetchAPI(`/api/environments/${id}`, { method: 'DELETE' }),
    exec: (id: string, command: string[]) =>
      fetchAPI(`/api/environments/${id}/exec`, { method: 'POST', body: JSON.stringify({ command }) }),
    logs: (id: string) => fetchAPI(`/api/environments/${id}/logs`),
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
  },
  files: {
    list: (id: string, path: string = '.') =>
      fetchAPI(`/api/environments/${id}/files?path=${encodeURIComponent(path)}`),
    getContent: async (id: string, path: string) => {
      const res = await fetch(`${API_BASE}/api/environments/${id}/files/content?path=${encodeURIComponent(path)}`, { credentials: 'include' });
      if (!res.ok) throw new Error("Failed to fetch file content");
      return res.text();
    },
    updateContent: (id: string, path: string, content: string) =>
      fetchAPI(`/api/environments/${id}/files/content?path=${encodeURIComponent(path)}`, {
        method: 'PUT',
        body: content,
        headers: { 'Content-Type': 'application/octet-stream' },
      }),
  },
};
