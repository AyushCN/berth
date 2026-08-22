import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Berth — Ephemeral Sandboxes',
  description: 'Research-grade ephemeral development environments',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen">{children}</body>
    </html>
  );
}
