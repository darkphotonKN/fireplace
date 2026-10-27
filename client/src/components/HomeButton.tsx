'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { House } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * The way back to the dashboard from anywhere in the app. The title links home
 * too, but nothing about it says so. On `/` itself it wears the page's coral,
 * the same "you are here" mark the sidebar gives the active plan.
 */
export default function HomeButton() {
  const pathname = usePathname();
  const isHome = pathname === '/';

  return (
    <Link
      href="/"
      aria-current={isHome ? 'page' : undefined}
      title="Home"
      className={cn(
        'group flex h-8 items-center gap-1.5 rounded-md px-2 text-sm transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isHome
          ? 'font-medium text-primary'
          : 'text-foreground/70 hover:bg-black/5 hover:text-foreground dark:hover:bg-white/10',
      )}
    >
      <House
        aria-hidden
        className={cn(
          'h-4 w-4 transition-transform group-hover:-translate-y-px',
          !isHome && 'group-hover:text-primary',
        )}
      />
      <span className="hidden sm:inline">Home</span>
    </Link>
  );
}
