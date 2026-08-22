import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  username: string;
  avatar_url: string;
}

interface AuthState {
  user: User | null;
  isLoading: boolean;
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  setUser: (user) => set({ user, isLoading: false }),
  setLoading: (isLoading) => set({ isLoading }),
  logout: () => {
    document.cookie = 'berth_token=; Max-Age=0; path=/';
    set({ user: null });
  },
}));
