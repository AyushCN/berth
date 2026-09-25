import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  username: string;
  avatar_url: string;
  max_sandboxes?: number;
  max_builds_per_hour?: number;
  created_at?: string;
}

interface AuthState {
  user: User | null;
  isLoading: boolean;
  authError: string | null;
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  setAuthError: (error: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  authError: null,
  setUser: (user) => set({ user, isLoading: false, authError: null }),
  setLoading: (isLoading) => set({ isLoading }),
  setAuthError: (authError) => set({ authError, isLoading: false }),
  logout: () => {
    document.cookie = 'berth_token=; Max-Age=0; path=/';
    set({ user: null, isLoading: false, authError: null });
  },
}));
