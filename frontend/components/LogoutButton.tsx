"use client";

import { useAuthStore } from "@/stores/auth";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import toast from "react-hot-toast";
import { api } from "@/lib/api";

export function LogoutButton() {
  const { user, logout } = useAuthStore();
  const router = useRouter();

  if (!user) return null;

  const handleLogout = async () => {
    try {
      await api.auth.logout();
    } catch {
      // Clear the local session even if the API is temporarily unreachable.
    }
    logout();
    toast.success("Signed out successfully");
    router.replace("/login");
  };

  return (
    <button
      onClick={handleLogout}
      className="flex shrink-0 items-center gap-2 rounded-lg px-2 py-2 text-xs font-semibold text-white/60 transition-colors hover:bg-white/5 hover:text-white sm:px-2.5 sm:text-sm"
    >
      <LogOut className="w-4 h-4" />
      Sign out
    </button>
  );
}
