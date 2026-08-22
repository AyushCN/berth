'use client';

import { useEffect } from 'react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('App Error Boundary caught:', error);
  }, [error]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-900 p-8">
      <div className="w-full max-w-lg bg-red-950/50 border border-red-900/50 text-red-200 p-6 rounded-lg backdrop-blur-sm">
        <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
          <span className="text-red-500">⚠</span> Something went wrong!
        </h2>
        <div className="bg-black/30 p-4 rounded text-sm font-mono break-all mb-6 overflow-auto max-h-48">
          {error.message || 'Unknown error occurred'}
        </div>
        <div className="flex gap-4">
          <button
            onClick={() => reset()}
            className="flex-1 bg-red-800 hover:bg-red-700 transition px-4 py-2 rounded font-medium"
          >
            Try again
          </button>
          <button
            onClick={() => window.location.href = '/login'}
            className="flex-1 bg-gray-800 hover:bg-gray-700 transition px-4 py-2 rounded font-medium"
          >
            Go to Login
          </button>
        </div>
      </div>
    </div>
  );
}
