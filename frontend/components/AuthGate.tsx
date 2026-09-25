"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader2, RefreshCw, ShieldAlert } from "lucide-react";
import { useAuthStore } from "@/stores/auth";

export function AuthGate({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, isLoading, authError } = useAuthStore();

  useEffect(() => {
    if (!isLoading && !user && !authError) router.replace("/login");
  }, [authError, isLoading, router, user]);

  if (authError) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center px-6 text-center">
        <div className="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl border border-amber-300/20 bg-amber-300/10 text-amber-200">
          <ShieldAlert className="h-6 w-6" />
        </div>
        <h1 className="text-xl font-bold text-white">Couldn’t connect to Berth</h1>
        <p className="mt-2 text-sm leading-6 text-white/55">{authError}</p>
        <button
          onClick={() => window.location.reload()}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-primary-fixed px-4 py-2.5 text-sm font-bold text-on-primary-fixed transition hover:brightness-110"
        >
          <RefreshCw className="h-4 w-4" /> Try again
        </button>
      </div>
    );
  }

  if (isLoading || !user) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center text-primary-fixed">
        <Loader2 className="h-7 w-7 animate-spin" aria-label="Loading your account" />
      </div>
    );
  }

  return children;
}
