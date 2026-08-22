'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

export default function LoginPage() {
  const router = useRouter();
  const { setUser } = useAuthStore();

  useEffect(() => {
    // Auto-login with dev endpoint for now
    api.auth.devLogin().then(() => {
      return api.auth.me();
    }).then((user) => {
      setUser(user);
      router.push('/dashboard');
    }).catch((err) => {
      console.error('Login failed', err);
    });
  }, [router, setUser]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold mb-4">Berth</h1>
        <p className="text-gray-400">Initializing dev session...</p>
      </div>
    </div>
  );
}
