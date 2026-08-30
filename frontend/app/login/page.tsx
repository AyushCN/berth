'use client';

import React, { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const InteractiveNetworkCanvas = () => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const mouseRef = useRef({ x: -1000, y: -1000 });

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let width: number;
    let height: number;
    let nodes: { x: number; y: number; vx: number; vy: number; radius: number }[] = [];
    const nodeCount = 35; // Denser network

    const initNodes = () => {
      nodes = [];
      for (let i = 0; i < nodeCount; i++) {
        nodes.push({
          x: Math.random() * width,
          y: Math.random() * height,
          vx: (Math.random() - 0.5) * 1.2,
          vy: (Math.random() - 0.5) * 1.2,
          radius: Math.random() * 2 + 1,
        });
      }
    };

    const resize = () => {
      if (!canvas.parentElement) return;
      width = canvas.width = canvas.parentElement.offsetWidth;
      height = canvas.height = canvas.parentElement.offsetHeight;
      initNodes();
    };

    const draw = () => {
      ctx.clearRect(0, 0, width, height);
      ctx.lineWidth = 1;

      // Update and draw connections
      for (let i = 0; i < nodes.length; i++) {
        const n1 = nodes[i];
        n1.x += n1.vx;
        n1.y += n1.vy;

        // Bounce
        if (n1.x < 0 || n1.x > width) n1.vx *= -1;
        if (n1.y < 0 || n1.y > height) n1.vy *= -1;

        ctx.beginPath();
        ctx.arc(n1.x, n1.y, n1.radius, 0, Math.PI * 2);
        ctx.fillStyle = '#0ea5e9'; // berth-500
        ctx.fill();

        // Connect nodes to each other
        for (let j = i + 1; j < nodes.length; j++) {
          const n2 = nodes[j];
          const dist = Math.hypot(n1.x - n2.x, n1.y - n2.y);
          if (dist < 150) {
            ctx.beginPath();
            ctx.moveTo(n1.x, n1.y);
            ctx.lineTo(n2.x, n2.y);
            ctx.strokeStyle = `rgba(14, 165, 233, ${0.15 - (dist / 150) * 0.15})`;
            ctx.stroke();
          }
        }

        // Connect to mouse
        const mouseDist = Math.hypot(n1.x - mouseRef.current.x, n1.y - mouseRef.current.y);
        if (mouseDist < 250) {
          ctx.beginPath();
          ctx.moveTo(n1.x, n1.y);
          ctx.lineTo(mouseRef.current.x, mouseRef.current.y);
          ctx.strokeStyle = `rgba(14, 165, 233, ${0.4 - (mouseDist / 250) * 0.4})`;
          ctx.lineWidth = 1.5;
          ctx.stroke();
          ctx.lineWidth = 1;
        }
      }
      animationFrameId = requestAnimationFrame(draw);
    };

    const handleMouseMove = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      mouseRef.current = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
    };

    const handleMouseLeave = () => {
      mouseRef.current = { x: -1000, y: -1000 };
    };

    window.addEventListener('resize', resize);
    canvas.addEventListener('mousemove', handleMouseMove);
    canvas.addEventListener('mouseleave', handleMouseLeave);

    resize();
    draw();

    return () => {
      window.removeEventListener('resize', resize);
      canvas.removeEventListener('mousemove', handleMouseMove);
      canvas.removeEventListener('mouseleave', handleMouseLeave);
      cancelAnimationFrame(animationFrameId);
    };
  }, []);

  return <canvas ref={canvasRef} className="absolute top-0 left-0 w-full h-full z-0 cursor-crosshair" />;
};

export default function LoginPage() {
  const router = useRouter();
  const { setUser } = useAuthStore();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showLoader, setShowLoader] = useState(false);

  const handleDevLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await api.auth.devLogin();
      const user = await api.auth.me();
      setShowLoader(true);
      
      // Simulate the beautiful loader animation for a bit before navigating
      setTimeout(() => {
        setUser(user);
        router.push('/dashboard');
      }, 1500);
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  if (showLoader) {
    return (
      <div className="bg-gray-950 text-gray-100 min-h-screen overflow-hidden flex flex-col items-center justify-center relative font-sans selection:bg-berth-900 selection:text-white">
        <style jsx global>{`
          .blueprint-bg {
            background-image: linear-gradient(rgba(14, 165, 233, 0.03) 1px, transparent 1px),
              linear-gradient(90deg, rgba(14, 165, 233, 0.03) 1px, transparent 1px);
            background-size: 40px 40px;
            background-position: center center;
          }
          .spinner-ring {
            border: 2px solid transparent;
            border-top-color: #0ea5e9;
            border-right-color: rgba(14, 165, 233, 0.3);
            border-radius: 50%;
            animation: spin 1.5s linear infinite;
            box-shadow: 0 0 15px rgba(14, 165, 233, 0.2), inset 0 0 10px rgba(14, 165, 233, 0.1);
          }
          .spinner-ring-inner {
            border: 2px solid transparent;
            border-left-color: #38bdf8;
            border-bottom-color: rgba(56, 189, 248, 0.3);
            border-radius: 50%;
            animation: spin-reverse 2s linear infinite;
          }
          .pulse-glow {
            animation: pulse-op 2s ease-in-out infinite alternate;
          }
          .terminal-text {
            overflow: hidden;
            border-right: 0.15em solid #0ea5e9;
            white-space: nowrap;
            margin: 0 auto;
            letter-spacing: 0.15em;
            animation: typing 2.5s steps(40, end), blink-caret 0.75s step-end infinite;
          }
          @keyframes spin {
            100% {
              transform: rotate(360deg);
            }
          }
          @keyframes spin-reverse {
            100% {
              transform: rotate(-360deg);
            }
          }
          @keyframes pulse-op {
            0% {
              opacity: 0.6;
              text-shadow: 0 0 5px rgba(14, 165, 233, 0.2);
            }
            100% {
              opacity: 1;
              text-shadow: 0 0 15px rgba(14, 165, 233, 0.6);
            }
          }
          @keyframes typing {
            from {
              width: 0;
            }
            to {
              width: 100%;
            }
          }
          @keyframes blink-caret {
            from,
            to {
              border-color: transparent;
            }
            50% {
              border-color: #0ea5e9;
            }
          }
        `}</style>

        {/* Background Grid */}
        <div className="absolute inset-0 blueprint-bg z-0 pointer-events-none opacity-50"></div>
        <main className="z-10 flex flex-col items-center justify-center flex-grow w-full max-w-[1440px] px-8 relative">
          {/* Logo & Spinner Container */}
          <div className="relative w-48 h-48 flex items-center justify-center mb-12">
            <div className="absolute inset-0 spinner-ring"></div>
            <div className="absolute inset-4 spinner-ring-inner"></div>
            <div className="relative z-10 text-berth-500 drop-shadow-[0_0_15px_rgba(14,165,233,0.5)]">
              <span className="material-symbols-outlined text-[64px]" style={{ fontVariationSettings: "'FILL' 0, 'wght' 200" }}>
                hexagon
              </span>
              <span
                className="material-symbols-outlined absolute inset-0 m-auto flex items-center justify-center text-[24px]"
                style={{ fontVariationSettings: "'FILL' 1, 'wght' 400" }}
              >
                memory
              </span>
            </div>
            <div className="absolute top-0 left-1/2 -translate-x-1/2 -translate-y-1/2 w-2 h-2 bg-berth-500 rounded-full shadow-[0_0_8px_#0ea5e9]"></div>
            <div className="absolute bottom-0 left-1/2 -translate-x-1/2 translate-y-1/2 w-2 h-2 bg-berth-500 rounded-full shadow-[0_0_8px_#0ea5e9]"></div>
          </div>

          {/* Text */}
          <h1 className="text-2xl md:text-3xl font-bold text-berth-500 tracking-widest uppercase pulse-glow text-center mb-4">
            Preparing your environment...
          </h1>
          <p className="text-base text-gray-400 opacity-70 tracking-wide">Initializing core modules</p>
        </main>

        {/* Removed Footer */}
      </div>
    );
  }

  return (
    <div className="bg-gray-950 text-gray-100 min-h-screen selection:bg-berth-900/50">
      <style jsx global>{`
        .blueprint-bg {
          background-image: radial-gradient(circle at 2px 2px, rgba(14, 165, 233, 0.05) 1px, transparent 0);
          background-size: 40px 40px;
        }
        .glow-effect {
          box-shadow: 0 0 20px rgba(14, 165, 233, 0.15);
        }
      `}</style>
      <main className="flex min-h-screen overflow-hidden">
        {/* Left Side: Visual Experience (55%) */}
        <section className="hidden lg:flex lg:w-[55%] relative flex-col justify-center items-center p-16 bg-gray-900 overflow-hidden border-r border-gray-800">
          <div className="absolute inset-0 z-0 bg-[radial-gradient(circle_at_50%_50%,rgba(14,165,233,0.05),transparent_70%)]">
            <InteractiveNetworkCanvas />
          </div>

          <div className="relative z-10 text-center max-w-[36rem] mx-auto">
            <h1 className="text-5xl font-bold text-gray-100 mb-6 tracking-tighter">Welcome Back</h1>
            <p className="text-lg text-gray-400 mb-16">
              Build faster with isolated sandboxes. Deploy, test, and scale with infrastructure that feels like magic.
            </p>
          </div>

          <div className="absolute bottom-8 left-0 w-full text-center px-10">
            <p className="text-xs text-gray-600 uppercase tracking-[0.2em] font-bold">Trusted by developers shipping 10x faster</p>
          </div>
        </section>

        {/* Right Side: Login Form (45%) */}
        <section className="w-full lg:w-[45%] flex flex-col justify-center items-center px-4 md:px-10 py-16 bg-gray-950 relative">
          <div className="w-full max-w-[420px]">
            <div className="mb-16 text-center">
              <Link href="/" className="inline-flex items-center gap-1 mb-6 hover:opacity-80 transition-opacity">
                <span className="material-symbols-outlined text-berth-500 text-[32px]" style={{ fontVariationSettings: "'FILL' 1" }}>
                  dataset
                </span>
                <span className="text-2xl font-bold text-berth-500 tracking-tight">Berth</span>
              </Link>
              <h2 className="text-3xl font-bold text-gray-100 mb-1">Sign in to your workspace</h2>
              <p className="text-base text-gray-400">Access your projects and sandboxes</p>
            </div>

            <div className="space-y-4 mb-8">
              <button
                onClick={handleDevLogin}
                disabled={loading}
                className="w-full flex items-center justify-center gap-3 bg-berth-600 text-white font-bold py-4 rounded-lg hover:bg-berth-700 active:scale-95 transition-all duration-200 text-lg border border-berth-500/30 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Authenticating...' : 'Developer Login'}
              </button>

              <button
                disabled
                className="w-full flex items-center justify-center gap-3 bg-[#24292e] text-white font-bold py-4 rounded-lg cursor-not-allowed opacity-50 text-lg border border-gray-800"
              >
                <svg
                  height="24"
                  aria-hidden="true"
                  viewBox="0 0 16 16"
                  version="1.1"
                  width="24"
                  data-view-component="true"
                  className="fill-current"
                >
                  <path d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z"></path>
                </svg>
                Continue with GitHub
              </button>
            </div>
            
            {error && (
              <div className="bg-red-900/30 border border-red-800 text-red-200 px-4 py-3 rounded text-sm mb-4 text-center">
                {error}
              </div>
            )}

            <div className="mt-8 text-center text-gray-500 text-sm">
              <p>GitHub OAuth is disabled for local development.</p>
              <p>Use Developer Login to bypass.</p>
            </div>
          </div>

          <div className="lg:hidden mt-auto pt-16 text-center">
            <p className="text-xs font-bold text-gray-600 opacity-50 uppercase tracking-widest">© 2026 Berth Sandbox</p>
          </div>
        </section>
      </main>
    </div>
  );
}
