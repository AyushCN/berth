"use client";

import { create } from "zustand";

interface Environment {
  id: string;
  name: string;
  state: string;
  git_url?: string;
  git_branch?: string;
  created_at: string;
  public_url?: string;
  port?: number;
  owner_id: string;
  project_id?: string;
  has_uncommitted_changes?: boolean;
  last_activity_at?: string;
}

interface EnvState {
  environments: Environment[];
  activeEnvironmentId: string | null;
  isLoading: boolean;
  setEnvironments: (envs: Environment[]) => void;
  addEnvironment: (env: Environment) => void;
  updateEnvironment: (id: string, data: Partial<Environment>) => void;
  removeEnvironment: (id: string) => void;
  selectEnvironment: (id: string) => void;
  setLoading: (loading: boolean) => void;
  getActiveEnvironment: () => Environment | undefined;
}

export const useEnvStore = create<EnvState>((set, get) => ({
  environments: [],
  activeEnvironmentId: null,
  isLoading: false,

  setEnvironments: (envs) => set({ environments: envs }),

  addEnvironment: (env) =>
    set((state) => ({
      environments: [env, ...state.environments],
    })),

  updateEnvironment: (id, data) =>
    set((state) => ({
      environments: state.environments.map((e) =>
        e.id === id ? { ...e, ...data } : e
      ),
    })),

  removeEnvironment: (id) =>
    set((state) => ({
      environments: state.environments.filter((e) => e.id !== id),
      activeEnvironmentId:
        state.activeEnvironmentId === id ? null : state.activeEnvironmentId,
    })),

  selectEnvironment: (id: string) =>
    set({ activeEnvironmentId: id }),

  setLoading: (loading: boolean) => set({ isLoading: loading }),

  getActiveEnvironment: () => {
    const { environments, activeEnvironmentId } = get();
    return environments.find((e) => e.id === activeEnvironmentId);
  },
}));