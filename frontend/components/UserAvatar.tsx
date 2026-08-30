"use client";

import Link from "next/link";
import { useAuthStore } from "@/stores/auth";
import { User } from "lucide-react";

export function UserAvatar() {
  const { user } = useAuthStore();
  
  const initials = user?.email ? user.email.slice(0, 2).toUpperCase() : null;

  return (
    <Link
      href="/dashboard"
      className="w-8 h-8 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center text-primary-fixed hover:bg-primary-fixed/20 transition-colors"
      title="Account"
    >
      {initials ? (
        <span className="text-xs font-black leading-none">{initials}</span>
      ) : (
        <User className="w-4 h-4" />
      )}
    </Link>
  );
}
