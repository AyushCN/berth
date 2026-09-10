"use client";

import { useAuthStore } from "@/stores/auth";
import { motion } from "framer-motion";
import { Github, Mail, Shield, Zap, Box, Clock } from "lucide-react";

export default function ProfilePage() {
  const { user } = useAuthStore();

  if (!user) {
    return (
      <div className="w-full h-full flex items-center justify-center min-h-[500px]">
        <div className="w-10 h-10 border-4 border-primary-fixed border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto py-8">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center gap-8 bg-surface-container-low border border-outline-variant p-8 rounded-3xl backdrop-blur-xl"
      >
        <div className="relative">
          {user.avatar_url ? (
            <img 
              src={user.avatar_url} 
              alt="Avatar" 
              className="w-32 h-32 rounded-2xl object-cover shadow-2xl border-2 border-primary-fixed/20"
            />
          ) : (
            <div className="w-32 h-32 rounded-2xl bg-primary-fixed/10 flex items-center justify-center text-4xl font-black text-primary-fixed shadow-2xl border-2 border-primary-fixed/20">
              {user.email.slice(0, 2).toUpperCase()}
            </div>
          )}
          <div className="absolute -bottom-3 -right-3 bg-surface p-2 rounded-xl border border-outline-variant shadow-lg">
            <Github className="w-5 h-5 text-on-surface" />
          </div>
        </div>

        <div className="flex-1">
          <h1 className="text-4xl font-bold tracking-tight text-on-surface mb-2">
            {user.username || "Anonymous Developer"}
          </h1>
          <div className="flex items-center gap-4 text-on-surface-variant">
            <span className="flex items-center gap-1.5 text-sm font-medium">
              <Mail className="w-4 h-4" />
              {user.email}
            </span>
            <span className="flex items-center gap-1.5 text-sm font-medium">
              <Shield className="w-4 h-4 text-emerald-400" />
              Verified User
            </span>
          </div>
        </div>
      </motion.div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-8">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="bg-surface-container-low border border-outline-variant p-6 rounded-3xl"
        >
          <div className="w-10 h-10 rounded-xl bg-primary-fixed/10 flex items-center justify-center mb-4">
            <Box className="w-5 h-5 text-primary-fixed" />
          </div>
          <p className="text-sm font-medium text-on-surface-variant mb-1">Max Sandboxes</p>
          <p className="text-3xl font-bold text-on-surface">{user.max_sandboxes || "∞"}</p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="bg-surface-container-low border border-outline-variant p-6 rounded-3xl"
        >
          <div className="w-10 h-10 rounded-xl bg-orange-500/10 flex items-center justify-center mb-4">
            <Zap className="w-5 h-5 text-orange-400" />
          </div>
          <p className="text-sm font-medium text-on-surface-variant mb-1">Max Builds / Hour</p>
          <p className="text-3xl font-bold text-on-surface">{user.max_builds_per_hour || "∞"}</p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="bg-surface-container-low border border-outline-variant p-6 rounded-3xl"
        >
          <div className="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center mb-4">
            <Clock className="w-5 h-5 text-blue-400" />
          </div>
          <p className="text-sm font-medium text-on-surface-variant mb-1">Account Created</p>
          <p className="text-lg font-bold text-on-surface mt-2">
            {new Date(user.created_at || Date.now()).toLocaleDateString(undefined, { 
              year: 'numeric', 
              month: 'long', 
              day: 'numeric' 
            })}
          </p>
        </motion.div>
      </div>
    </div>
  );
}
