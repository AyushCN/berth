"use client";

import { useAuthStore } from "@/stores/auth";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import toast from "react-hot-toast";

export function LogoutButton() {
  const { user, logout } = useAuthStore();
  const router = useRouter();

  if (!user) return null;

  const handleLogout = () => {
    logout();
    toast.success("Signed out successfully");
    router.push("/");
  };

  return (
    <button
      onClick={handleLogout}
      className="text-sm font-medium text-white/70 hover:text-white transition-colors flex items-center gap-2"
    >
      <LogOut className="w-4 h-4" />
      Sign out
    </button>
  );
}
