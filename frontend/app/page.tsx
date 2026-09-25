"use client";

import React, { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { motion, useScroll, useTransform } from "framer-motion";
import {
  Terminal,
  Code2,
  Box,
  ArrowRight,
  ChevronRight,
  Sparkles
} from "lucide-react";

// Modern Glass Card with Gradient Borders
const GlassCard = ({
  children,
  className = "",
  delay = 0,
}: {
  children: React.ReactNode;
  className?: string;
  delay?: number;
}) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 30 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-100px" }}
      transition={{ duration: 0.7, delay, type: "spring", bounce: 0.4 }}
      className={`relative group rounded-3xl p-[1px] overflow-hidden ${className}`}
    >
      <div className="absolute inset-0 bg-gradient-to-br from-primary-fixed/40 via-surface-container-high to-primary-fixed/10 opacity-50 group-hover:opacity-100 transition-opacity duration-500" />
      <div className="relative h-full bg-surface-container-lowest/80 backdrop-blur-2xl rounded-[23px] p-8 flex flex-col border border-white/5">
        <div className="absolute inset-0 bg-gradient-to-br from-primary-fixed/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none" />
        {children}
      </div>
    </motion.div>
  );
};

// Subtle Animated Background Grid
const GridBackground = () => {
  return (
    <div className="fixed inset-0 pointer-events-none z-0 overflow-hidden">
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#00f0ff08_1px,transparent_1px),linear-gradient(to_bottom,#00f0ff08_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_60%_at_50%_0%,#000_70%,transparent_100%)]" />
      <div className="absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2 w-[1000px] h-[500px] bg-primary-fixed/20 blur-[120px] rounded-full pointer-events-none mix-blend-screen" />
    </div>
  );
};

export default function LandingPage() {
  const { scrollYProgress } = useScroll();
  const yHero = useTransform(scrollYProgress, [0, 1], [0, 300]);
  const opacityHero = useTransform(scrollYProgress, [0, 0.5], [1, 0]);

  return (
    <div className="min-h-screen text-on-surface font-sans selection:bg-primary-fixed/30 selection:text-primary-fixed overflow-hidden relative">
      <GridBackground />

      {/* Navigation */}
      <nav className="fixed top-0 w-full z-50 border-b border-white/10 bg-[#050505]/70 backdrop-blur-xl">
        <div className="max-w-7xl mx-auto px-6 h-20 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-fixed to-primary-container flex items-center justify-center shadow-[0_0_20px_rgba(0,240,255,0.3)]">
              <Code2 className="w-5 h-5 text-on-primary-fixed font-black" />
            </div>
            <span className="text-2xl font-black tracking-tighter text-white">
              Berth
            </span>
          </div>
          <div className="flex items-center gap-6">
            <Link
              href="/login"
              className="text-sm font-semibold text-on-surface-variant hover:text-white transition-colors"
            >
              Sign In
            </Link>
            <Link
              href="/login"
              className="group relative px-6 py-2.5 rounded-full overflow-hidden bg-white text-black font-bold text-sm transition-all hover:scale-105 active:scale-95"
            >
              <span className="relative z-10 flex items-center gap-2">
                Get Started <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
              </span>
              <div className="absolute inset-0 bg-gradient-to-r from-primary-fixed to-blue-400 opacity-0 group-hover:opacity-20 transition-opacity" />
            </Link>
          </div>
        </div>
      </nav>

      <main className="relative z-10 pt-32 pb-24 px-6 max-w-7xl mx-auto">
        {/* Hero Section */}
        <motion.section 
          style={{ y: yHero, opacity: opacityHero }}
          className="flex flex-col items-center text-center py-20"
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.8, ease: "easeOut" }}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary-fixed/10 border border-primary-fixed/20 text-primary-fixed text-sm font-semibold mb-8 backdrop-blur-md"
          >
            <Sparkles className="w-4 h-4" />
            <span>Single-host sandbox prototype</span>
          </motion.div>

          <motion.h1 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 0.1, ease: "easeOut" }}
            className="text-6xl md:text-8xl font-black text-white tracking-tighter mb-8 leading-[1.1] max-w-5xl drop-shadow-2xl"
          >
            Repository Sandboxes.
            <br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary-fixed via-blue-400 to-indigo-500">
              Built for a fast local demo.
            </span>
          </motion.h1>

          <motion.p 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 0.2, ease: "easeOut" }}
            className="text-xl md:text-2xl text-on-surface-variant max-w-3xl mb-12 font-medium"
          >
            Berth clones a public GitHub repository, detects its runtime from project files, and starts it in a containerd-managed workspace. This prototype uses host networking and is intended for trusted, single-host use.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 0.3, ease: "easeOut" }}
            className="flex flex-col sm:flex-row gap-4 w-full sm:w-auto"
          >
            <Link
              href="/login"
              className="relative group px-8 py-4 rounded-full bg-primary-fixed text-black font-black text-lg shadow-[0_0_40px_rgba(0,240,255,0.3)] hover:shadow-[0_0_60px_rgba(0,240,255,0.5)] transition-all hover:-translate-y-1"
            >
              Start Coding Free
            </Link>
            <a
              href="#features"
              className="px-8 py-4 rounded-full bg-white/5 border border-white/10 text-white font-bold text-lg hover:bg-white/10 backdrop-blur-md transition-all flex items-center justify-center gap-2"
            >
              Explore Features <ChevronRight className="w-5 h-5" />
            </a>
          </motion.div>
        </motion.section>

        {/* Example provisioning flow */}
        <motion.section
          initial={{ opacity: 0, y: 60 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.6, type: "spring", bounce: 0.3 }}
          className="relative max-w-5xl mx-auto mt-12 mb-32"
        >
          <div className="absolute -inset-1 bg-gradient-to-r from-primary-fixed via-blue-500 to-indigo-500 rounded-[2rem] blur-2xl opacity-20" />
          <div className="relative rounded-[2rem] border border-white/10 bg-[#0A0A0B] shadow-2xl overflow-hidden flex flex-col">
            <div className="h-12 border-b border-white/10 bg-[#121214] flex items-center px-4 gap-4">
              <div className="flex gap-2">
                <div className="w-3 h-3 rounded-full bg-red-500/80" />
                <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
                <div className="w-3 h-3 rounded-full bg-green-500/80" />
              </div>
              <div className="text-xs font-mono text-white/40 font-medium">berth / workspace</div>
            </div>
            <div className="p-6 font-mono text-sm leading-relaxed text-blue-300">
              <div className="flex items-center gap-2 text-white/50 mb-2">
                <span className="text-emerald-400">➜</span>
                <span className="text-primary-fixed">~</span>
                <span>git clone https://github.com/user/project.git</span>
              </div>
              <div className="text-white/70 mb-4">Cloning into 'project'...</div>
              
              <div className="flex items-center gap-2 text-white/50 mb-2">
                <span className="text-emerald-400">➜</span>
                <span className="text-primary-fixed">~</span>
                <span>berth inspect</span>
              </div>
              <div className="text-primary-fixed mb-1 flex items-center gap-2">
                <Box className="w-4 h-4" /> Detecting runtime from repository files...
              </div>
              <div className="text-white/70 ml-6 mb-1">Detected Language: Node.js (TypeScript)</div>
              <div className="text-white/70 ml-6 mb-1">Detected Start Command: npm run dev</div>
              <div className="text-white/70 ml-6 mb-4">Base Image Selected: node:20-alpine</div>

              <div className="flex items-center gap-2 text-white/50 mb-2">
                <span className="text-emerald-400">➜</span>
                <span className="text-primary-fixed">~</span>
                <span>berth up</span>
              </div>
              <div className="text-emerald-400 mb-1 flex items-center gap-2">
                <Box className="w-4 h-4" /> Starting container through containerd (runc by default)...
              </div>
              <div className="text-white/70 ml-6 mt-2">
                Preview route: /p/&lt;sandbox-id&gt;/ (single-host demo)
              </div>
              <div className="text-primary-fixed ml-6 mt-1 animate-pulse">_</div>
            </div>
          </div>
        </motion.section>

        {/* Feature Grid */}
        <section id="features" className="py-24">
          <div className="text-center mb-16">
            <h2 className="text-4xl md:text-5xl font-black text-white mb-4 tracking-tight">
              Powerful By Design.
            </h2>
            <p className="text-xl text-on-surface-variant max-w-2xl mx-auto">
              The current prototype flow, with clear limits on what is ready.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <GlassCard delay={0.1}>
              <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center mb-6 border border-blue-500/20 text-blue-400">
                <Terminal className="w-6 h-6" />
              </div>
              <h3 className="text-2xl font-bold text-white mb-3">Workspace UI</h3>
              <p className="text-on-surface-variant leading-relaxed">
                The workspace connects file browsing and saves, terminal sessions, status, and preview to the backend.
              </p>
            </GlassCard>

            <GlassCard delay={0.3}>
              <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 flex items-center justify-center mb-6 border border-emerald-500/20 text-emerald-400">
                <Box className="w-6 h-6" />
              </div>
              <h3 className="text-2xl font-bold text-white mb-3">Containerd Runtime</h3>
              <p className="text-on-surface-variant leading-relaxed">
                The worker creates containers through containerd. Local development defaults to runc; gVisor is not the default verified configuration.
              </p>
            </GlassCard>

            <GlassCard delay={0.4}>
              <div className="w-12 h-12 rounded-2xl bg-orange-500/10 flex items-center justify-center mb-6 border border-orange-500/20 text-orange-400">
                <Terminal className="w-6 h-6" />
              </div>
              <h3 className="text-2xl font-bold text-white mb-3">Runtime Detection</h3>
              <p className="text-on-surface-variant leading-relaxed">
                Simple repository-file rules select setup commands for Node.js, Python, Go, or Rust after cloning.
              </p>
            </GlassCard>
          </div>
        </section>

        {/* CTA Section */}
        <section className="py-32 text-center relative">
          <div className="absolute inset-0 bg-primary-fixed/5 blur-3xl rounded-full" />
          <h2 className="text-5xl font-black text-white mb-6 relative z-10 tracking-tight">
            Ready to dive in?
          </h2>
          <p className="text-xl text-on-surface-variant mb-10 max-w-2xl mx-auto relative z-10">
            Try the single-host prototype. It is not intended for untrusted or multi-tenant workloads.
          </p>
          <Link
            href="/login"
            className="relative z-10 inline-flex items-center gap-3 px-10 py-5 rounded-full bg-white text-black font-black text-xl hover:scale-105 active:scale-95 transition-all shadow-2xl"
          >
            Create Your First Sandbox <ArrowRight className="w-6 h-6" />
          </Link>
        </section>
      </main>

      {/* Minimal Footer */}
      <footer className="border-t border-white/10 bg-[#050505] relative z-10">
        <div className="max-w-7xl mx-auto px-6 py-12 flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="flex items-center gap-2 text-white/50 font-semibold">
            <Code2 className="w-5 h-5" />
            <span>© 2026 Berth. All rights reserved.</span>
          </div>
          <div className="flex gap-8 text-sm font-medium text-white/40">
            <a href="#" className="hover:text-white transition-colors">Privacy</a>
            <a href="#" className="hover:text-white transition-colors">Terms</a>
            <a href="#" className="hover:text-white transition-colors">GitHub</a>
          </div>
        </div>
      </footer>
    </div>
  );
}
