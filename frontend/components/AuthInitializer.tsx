"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";

export function AuthInitializer() {
  const { user, setUser, setLoading, setAuthError } = useAuthStore();

  useEffect(() => {
    if (user) return;

    let cancelled = false;
    setLoading(true);
    api.auth.me()
      .then((data) => {
        if (cancelled) return;
        if (data) setUser(data);
        else setLoading(false);
      })
      .catch((error: unknown) => {
        if (cancelled) return;
        if ((error as { status?: number })?.status === 401) {
          // Expected for a signed-out visitor; protected routes redirect to login.
          setLoading(false);
          setAuthError(null);
          return;
        }
        setAuthError("Could not reach the Berth API. Check the API connection and retry.");
      });

    return () => {
      cancelled = true;
    };
  }, [user, setUser, setLoading, setAuthError]);

  return null;
}
