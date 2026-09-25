import Link from "next/link";
import { Code2 } from "lucide-react";
import { AuthGate } from "@/components/AuthGate";
import { LogoutButton } from "@/components/LogoutButton";
import { UserAvatar } from "@/components/UserAvatar";

const links = [
  { href: "/dashboard", label: "Sandboxes" },
  { href: "/projects", label: "Projects" },
];

export default function MainLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="min-h-screen bg-[#080a0b] text-on-surface">
      <nav className="sticky top-0 z-50 w-full border-b border-white/[0.08] bg-[#080a0b]/85 backdrop-blur-xl">
        <div className="mx-auto flex min-h-16 max-w-screen-2xl flex-wrap items-center gap-x-5 px-4 py-2 sm:h-16 sm:flex-nowrap sm:px-6 sm:py-0">
          <Link href="/dashboard" className="flex shrink-0 items-center gap-2.5 text-white transition hover:text-primary-fixed">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-primary-fixed/20 bg-primary-fixed/10 text-primary-fixed">
              <Code2 className="h-5 w-5" />
            </span>
            <span className="text-lg font-extrabold tracking-tight">Berth</span>
          </Link>

          <div className="ml-auto flex shrink-0 items-center gap-3">
            <UserAvatar />
            <LogoutButton />
          </div>
          <div className="order-3 w-full overflow-x-auto sm:order-none sm:ml-auto sm:w-auto">
            <div className="flex min-w-max items-center gap-1 sm:gap-2">
            {links.map(({ href, label }) => (
              <Link key={href} href={href} className="whitespace-nowrap rounded-lg px-2.5 py-2 text-xs font-semibold text-on-surface-variant transition hover:bg-white/5 hover:text-white sm:px-3 sm:text-sm">
                {label}
              </Link>
            ))}
            </div>
          </div>
        </div>
      </nav>
      <main className="mx-auto w-full max-w-screen-2xl px-4 py-7 sm:px-6 sm:py-8">
        <AuthGate>{children}</AuthGate>
      </main>
    </div>
  );
}
