import { create } from 'zustand';

interface Environment {
  id: string;
  name: string;
  git_url: string;
  git_branch: string;
  state: string;
  created_at: string;
}

interface EnvState {
  environments: Environment[];
  selectedId: string | null;
  isLoading: boolean;
  setEnvironments: (envs: Environment[]) => void;
  addEnvironment: (env: Environment) => void;
  removeEnvironment: (id: string) => void;
  selectEnvironment: (id: string | null) => void;
  setLoading: (loading: boolean) => void;
}

export const useEnvStore = create<EnvState>((set) => ({
  environments: [],
  selectedId: null,
  isLoading: false,
  setEnvironments: (environments) => set({ environments }),
  addEnvironment: (env) => set((s) => ({ environments: [env, ...s.environments] })),
  removeEnvironment: (id) =>
    set((s) => ({ environments: s.environments.filter((e) => e.id !== id) })),
  selectEnvironment: (selectedId) => set({ selectedId }),
  setLoading: (isLoading) => set({ isLoading }),
}));
