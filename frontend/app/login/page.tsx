'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { motion, AnimatePresence } from 'framer-motion';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const FloatingOrb = ({ color, size, initialX, initialY, duration, delay }: any) => {
  return (
    <motion.div
      className="absolute rounded-full mix-blend-screen filter blur-[80px] opacity-40"
      style={{
        backgroundColor: color,
        width: size,
        height: size,
        left: initialX,
        top: initialY,
      }}
      animate={{
        x: [0, 100, -50, 0],
        y: [0, -100, 50, 0],
        scale: [1, 1.2, 0.8, 1],
      }}
      transition={{
        duration: duration,
        ease: 'easeInOut',
        repeat: Infinity,
        delay: delay,
      }}
    />
  );
};

export default function LoginPage() {
  const router = useRouter();
  const { setUser } = useAuthStore();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showLoader, setShowLoader] = useState(false);
  const [loadingStep, setLoadingStep] = useState(0);

  const loadingSteps = [
    'Establishing secure connection...',
    'Authenticating credentials...',
    'Initializing core modules...',
    'Provisioning workspace...',
  ];

  const handleDevLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await api.auth.devLogin();
      const user = await api.auth.me();
      setShowLoader(true);
      
      // Step through the loading sequence
      for (let i = 0; i < loadingSteps.length; i++) {
        setLoadingStep(i);
        await new Promise((resolve) => setTimeout(resolve, 600));
      }
      
      setUser(user);
      router.push('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Login failed');
      setLoading(false);
    }
  };

  return (
    <div className="relative min-h-screen bg-[#050505] text-gray-100 overflow-hidden flex items-center justify-center selection:bg-cyan-900/50">
      {/* Dynamic Background */}
      <div className="absolute inset-0 z-0 overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(14,165,233,0.03)_0%,transparent_100%)]"></div>
        <FloatingOrb color="#0ea5e9" size="400px" initialX="10%" initialY="20%" duration={15} delay={0} />
        <FloatingOrb color="#38bdf8" size="300px" initialX="70%" initialY="60%" duration={20} delay={2} />
        <FloatingOrb color="#0369a1" size="500px" initialX="40%" initialY="-10%" duration={25} delay={5} />
        
        {/* Subtle Grid */}
        <div 
          className="absolute inset-0 opacity-[0.15]" 
          style={{
            backgroundImage: `linear-gradient(rgba(14, 165, 233, 0.2) 1px, transparent 1px), linear-gradient(90deg, rgba(14, 165, 233, 0.2) 1px, transparent 1px)`,
            backgroundSize: '40px 40px',
            maskImage: 'radial-gradient(ellipse at center, black 40%, transparent 80%)',
            WebkitMaskImage: 'radial-gradient(ellipse at center, black 40%, transparent 80%)'
          }}
        />
      </div>

      <AnimatePresence mode="wait">
        {!showLoader ? (
          <motion.main
            key="login-form"
            initial={{ opacity: 0, y: 20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, scale: 1.05, filter: 'blur(10px)' }}
            transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
            className="z-10 w-full max-w-md px-6 relative"
          >
            {/* Glassmorphic Card */}
            <div className="backdrop-blur-xl bg-gray-950/40 border border-white/5 rounded-3xl p-10 shadow-[0_0_80px_rgba(14,165,233,0.1)] relative overflow-hidden group">
              {/* Subtle hover glow on card */}
              <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none" />
              
              <div className="mb-12 text-center relative z-10">
                <motion.div
                  initial={{ scale: 0.8, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ delay: 0.2, duration: 0.5, type: 'spring' }}
                >
                  <Link href="/" className="inline-flex items-center gap-2 mb-8 hover:opacity-80 transition-opacity">
                    <span className="material-symbols-outlined text-cyan-400 text-[40px] drop-shadow-[0_0_15px_rgba(34,211,238,0.5)]" style={{ fontVariationSettings: "'FILL' 1" }}>
                      dataset
                    </span>
                    <span className="text-3xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-sky-600 tracking-tight">
                      Berth
                    </span>
                  </Link>
                </motion.div>
                <motion.h2 
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.3 }}
                  className="text-2xl font-bold text-gray-100 mb-2 tracking-tight"
                >
                  Enter the Sandbox
                </motion.h2>
                <motion.p 
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.4 }}
                  className="text-sm text-gray-400"
                >
                  Authentication gateway for isolated workspaces
                </motion.p>
              </div>

              <div className="space-y-4 relative z-10">
                <motion.button
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  onClick={handleDevLogin}
                  disabled={loading}
                  className="w-full relative group overflow-hidden bg-cyan-950/30 text-cyan-50 font-semibold py-3.5 rounded-xl border border-cyan-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="absolute inset-0 bg-gradient-to-r from-cyan-600/20 to-sky-600/20 translate-y-full group-hover:translate-y-0 transition-transform duration-300 ease-out" />
                  <span className="relative flex items-center justify-center gap-2">
                    {loading ? (
                      <span className="w-5 h-5 border-2 border-cyan-400/30 border-t-cyan-400 rounded-full animate-spin" />
                    ) : (
                      <span className="material-symbols-outlined text-[20px]">terminal</span>
                    )}
                    {loading ? 'Authenticating...' : 'Developer Login'}
                  </span>
                </motion.button>

                <motion.button
                  whileHover={{ scale: 1.02 }}
                  disabled
                  className="w-full flex items-center justify-center gap-3 bg-white/5 text-gray-300 font-semibold py-3.5 rounded-xl cursor-not-allowed opacity-50 border border-white/10 transition-colors"
                >
                  <svg height="20" aria-hidden="true" viewBox="0 0 16 16" width="20" className="fill-current">
                    <path d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z"></path>
                  </svg>
                  Continue with GitHub
                </motion.button>
              </div>
              
              <AnimatePresence>
                {error && (
                  <motion.div 
                    initial={{ opacity: 0, height: 0, marginTop: 0 }}
                    animate={{ opacity: 1, height: 'auto', marginTop: 16 }}
                    exit={{ opacity: 0, height: 0, marginTop: 0 }}
                    className="bg-red-950/50 border border-red-900/50 text-red-300 px-4 py-3 rounded-lg text-sm text-center overflow-hidden"
                  >
                    {error}
                  </motion.div>
                )}
              </AnimatePresence>

              <div className="mt-8 pt-6 border-t border-white/5 text-center relative z-10">
                <p className="text-xs text-gray-500 uppercase tracking-widest font-semibold mb-1">Notice</p>
                <p className="text-xs text-gray-500/80">OAuth disabled for local development.</p>
              </div>
            </div>
          </motion.main>
        ) : (
          <motion.main
            key="loader"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            className="z-10 flex flex-col items-center justify-center relative"
          >
            {/* High-tech Loader */}
            <div className="relative w-40 h-40 flex items-center justify-center mb-12">
              <motion.div 
                animate={{ rotate: 360 }}
                transition={{ duration: 4, repeat: Infinity, ease: "linear" }}
                className="absolute inset-0 border-2 border-transparent border-t-cyan-400/80 border-r-cyan-400/20 rounded-full"
              />
              <motion.div 
                animate={{ rotate: -360 }}
                transition={{ duration: 3, repeat: Infinity, ease: "linear" }}
                className="absolute inset-4 border-2 border-transparent border-b-sky-400/60 border-l-sky-400/20 rounded-full"
              />
              <motion.div
                animate={{ scale: [1, 1.1, 1], opacity: [0.7, 1, 0.7] }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
                className="relative z-10 text-cyan-400 drop-shadow-[0_0_15px_rgba(34,211,238,0.5)]"
              >
                <span className="material-symbols-outlined text-[56px]" style={{ fontVariationSettings: "'FILL' 0, 'wght' 200" }}>
                  hexagon
                </span>
                <span className="material-symbols-outlined absolute inset-0 m-auto flex items-center justify-center text-[20px]" style={{ fontVariationSettings: "'FILL' 1, 'wght' 400" }}>
                  memory
                </span>
              </motion.div>
            </div>

            {/* Dynamic Status Text */}
            <div className="h-16 flex flex-col items-center justify-center text-center">
              <AnimatePresence mode="wait">
                <motion.p
                  key={loadingStep}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  transition={{ duration: 0.3 }}
                  className="text-cyan-400 font-mono text-sm tracking-widest uppercase drop-shadow-[0_0_8px_rgba(34,211,238,0.4)]"
                >
                  {loadingSteps[loadingStep]}
                </motion.p>
              </AnimatePresence>
              <div className="mt-4 flex gap-1.5">
                {loadingSteps.map((_, i) => (
                  <motion.div
                    key={i}
                    initial={false}
                    animate={{
                      backgroundColor: i <= loadingStep ? '#22d3ee' : '#164e63',
                      scale: i === loadingStep ? 1.2 : 1
                    }}
                    className="w-1.5 h-1.5 rounded-full"
                  />
                ))}
              </div>
            </div>
          </motion.main>
        )}
      </AnimatePresence>
    </div>
  );
}
