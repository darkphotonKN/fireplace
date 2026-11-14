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
        // A 40px tap target below md, the compact desktop size from md (FS-49CQ2 R17).
        'group flex h-10 min-w-10 shrink-0 items-center justify-center gap-1.5 rounded-md px-2 text-sm transition-colors md:h-8',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isHome
          ? 'font-medium text-primary'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground',
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
