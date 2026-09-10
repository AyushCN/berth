"use client";

import Link from "next/link";
import { useAuthStore } from "@/stores/auth";
import { User } from "lucide-react";

export function UserAvatar() {
  const { user } = useAuthStore();
  
  const initials = user?.email ? user.email.slice(0, 2).toUpperCase() : null;

  return (
    <Link
      href="/profile"
      className="w-8 h-8 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center text-primary-fixed hover:bg-primary-fixed/20 transition-colors overflow-hidden"
      title="Account"
    >
      {user?.avatar_url ? (
        <img src={user.avatar_url} alt="Profile" className="w-full h-full object-cover" />
      ) : initials ? (
        <span className="text-xs font-black leading-none">{initials}</span>
      ) : (
        <User className="w-4 h-4" />
      )}
    </Link>
  );
}
