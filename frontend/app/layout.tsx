import type { Metadata } from 'next';
import './globals.css';
import { Toaster } from "react-hot-toast";
import { AuthInitializer } from "@/components/AuthInitializer";

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
    <html lang="en" className="dark h-full antialiased">
      <head>
        <link
          href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-full flex flex-col bg-background text-on-surface font-sans selection:bg-primary-container selection:text-on-primary-container">
        <AuthInitializer />
        {children}
        <Toaster
          position="bottom-right"
          toastOptions={{ className: "glass-panel text-white" }}
        />
      </body>
    </html>
  );
}
