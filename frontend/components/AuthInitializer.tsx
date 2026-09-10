"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";

export function AuthInitializer() {
  const { user, setUser, setLoading } = useAuthStore();

  useEffect(() => {
    if (!user) {
      api.auth.me().then((data) => {
        if (data) {
          setUser(data);
        }
      }).catch((e) => {
        // Expected if user is not logged in
      }).finally(() => {
        setLoading(false);
      });
    }
  }, [user, setUser, setLoading]);

  return null;
}
