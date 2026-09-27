"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  User,
  Users,
  Circle,
  Loader2,
  Settings,
  LogOut,
  Mail,
  Bell,
  Moon,
  Sun,
  ChevronDown,
  ChevronRight,
  Search,
} from "lucide-react";
import { useAuthStore } from "@/stores/auth";

interface PresenceUser {
  id: string;
  name: string;
  avatar?: string;
  color: string;
  cursor?: { line: number; column: number };
  lastActive: string;
  isCurrentUser: boolean;
}

interface PresenceBarProps {
  envId: string;
}

export function PresenceBar({ envId }: PresenceBarProps) {
  const { user } = useAuthStore();
  const [users, setUsers] = useState<PresenceUser[]>([]);
  const [showMenu, setShowMenu] = useState(false);
  const [showSettings, setShowSettings] = useState(false);

  // Simulated presence data - in real app this would come from WebSocket
  useEffect(() => {
    const mockUsers: PresenceUser[] = [
      {
        id: user?.id || "1",
        name: user?.username || "You",
        avatar: user?.avatar_url,
        color: "#00d4ff",
        cursor: { line: 10, column: 5 },
        lastActive: "Just now",
        isCurrentUser: true,
      },
      {
        id: "2",
        name: "Alice Chen",
        avatar: undefined,
        color: "#f472b6",
        cursor: { line: 45, column: 12 },
        lastActive: "2 min ago",
        isCurrentUser: false,
      },
      {
        id: "3",
        name: "Bob Smith",
        avatar: undefined,
        color: "#34d399",
        cursor: { line: 120, column: 8 },
        lastActive: "5 min ago",
        isCurrentUser: false,
      },
    ];
    setUsers(mockUsers);
  }, [envId, user]);

  return (
    <div className="fixed bottom-0 right-0 left-0 z-40 bg-slate-900/95 backdrop-blur-sm border-t border-slate-800">
      <div className="max-w-full mx-auto px-4 py-2">
        <div className="flex items-center justify-between">
          {/* Users Online */}
          <div className="flex items-center gap-1">
            <Users className="w-4 h-4 text-slate-400" />
            <span className="text-sm text-slate-300">{users.length} online</span>
          </div>

          {/* User Avatars */}
          <div className="flex -space-x-2">
            {users.slice(0, 5).map((u) => (
              <motion.div
                key={u.id}
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                className="relative"
              >
                <div
                  className={`w-8 h-8 rounded-full border-2 border-slate-900 flex items-center justify-center text-xs font-bold ${
                    u.isCurrentUser ? "ring-2 ring-primary-fixed" : ""
                  }`}
                  style={{
                    backgroundColor: u.avatar ? undefined : u.color,
                    backgroundImage: u.avatar ? `url(${u.avatar})` : undefined,
                    backgroundSize: "cover",
                    backgroundPosition: "center",
                  }}
                  title={`${u.name} • ${u.lastActive}${u.cursor ? ` • Line ${u.cursor.line}` : ""}`}
                >
                  {u.avatar ? null : u.name.charAt(0).toUpperCase()}
                </div>
                {u.cursor && (
                  <div className="absolute -top-5 left-1/2 -translate-x-1/2 px-1.5 py-0.5 bg-slate-900 border border-slate-700 rounded text-[10px] text-slate-400 whitespace-nowrap">
                    Line {u.cursor.line}, Col {u.cursor.column}
                  </div>
                )}
              </motion.div>
            ))}
            {users.length > 5 && (
              <motion.div
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                className="w-8 h-8 rounded-full border-2 border-slate-700 bg-slate-800 flex items-center justify-center text-xs font-medium text-slate-400"
              >
                +{users.length - 5}
              </motion.div>
            )}
          </div>

          {/* Right Side Actions */}
          <div className="flex items-center gap-2">
            {/* Theme Toggle */}
            <button className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors" title="Toggle theme">
              <Sun className="w-4 h-4" />
            </button>

            {/* Notifications */}
            <button className="relative p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors" title="Notifications">
              <Bell className="w-4 h-4" />
              <span className="absolute -top-1 -right-1 w-4 h-4 bg-red-500 text-[10px] font-bold rounded-full flex items-center justify-center">
                3
              </span>
            </button>

            {/* User Menu */}
            <div className="relative">
              <button
                onClick={() => setShowMenu(!showMenu)}
                className="flex items-center gap-2 p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              >
                <div className="w-7 h-7 rounded-full bg-primary-fixed/20 border border-primary-fixed/30 flex items-center justify-center">
                  {user?.avatar_url ? (
                    <img src={user.avatar_url} alt="" className="w-full h-full rounded-full" />
                  ) : (
                    <span className="text-sm font-bold text-primary-fixed">
                      {user?.username?.charAt(0).toUpperCase() || "U"}
                    </span>
                  )}
                </div>
                <span className="text-sm font-medium text-white hidden sm:block">
                  {user?.username || "User"}
                </span>
                <ChevronDown className="w-4 h-4 text-slate-400" />
              </button>

              <AnimatePresence>
                {showMenu && (
                  <motion.div
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -10 }}
                    className="fixed bottom-16 right-4 z-50 bg-slate-900 border border-slate-700 rounded-xl shadow-xl py-1 min-w-[200px]"
                  >
                    <div className="px-3 py-2 border-b border-slate-700">
                      <p className="text-sm font-medium text-white">{user?.username}</p>
                      <p className="text-xs text-slate-400 truncate max-w-[200px]">{user?.email}</p>
                    </div>
                    <div className="py-1">
                      <button className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2">
                        <User className="w-4 h-4" /> Profile
                      </button>
                      <button className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2">
                        <Settings className="w-4 h-4" /> Settings
                      </button>
                      <button className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 flex items-center gap-2">
                        <Bell className="w-4 h-4" /> Notifications
                      </button>
                      <div className="border-t border-slate-700 my-1" />
                      <button className="w-full px-3 py-2 text-sm text-red-400 hover:bg-red-500/10 flex items-center gap-2">
                        <LogOut className="w-4 h-4" /> Sign Out
                      </button>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}