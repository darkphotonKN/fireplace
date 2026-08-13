'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

type CtaVariant = 'primary' | 'outline' | 'quiet';

const VARIANTS: Record<CtaVariant, string> = {
  // On-coral text comes from the token. Never `text-white`: white on #F76F53 is
  // 2.86:1, under the 4.5:1 AA needs for normal text.
  primary:
    'rounded-md bg-primary px-8 py-3 font-medium text-primary-foreground transition-opacity hover:opacity-90',
  outline:
    'rounded-md border border-primary px-8 py-3 font-medium text-primary transition-colors hover:bg-primary/10',
  quiet: 'text-sm text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline',
};

/**
 * The tour's calls to action. One definition so the coral button looks the same
 * in the hero, the closing section, and the chrome bar — and so the on-coral
 * token is applied in exactly one place rather than re-typed per CTA.
 */
export default function CtaLink({
  href,
  variant = 'primary',
  className,
  children,
}: {
  href: string;
  variant?: CtaVariant;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Link href={href} className={cn(VARIANTS[variant], className)}>
      {children}
    </Link>
  );
}

/** Auth targets, unchanged from the pre-tour landing. */
export const SIGN_UP_HREF = '/auth?tab=signup';
export const SIGN_IN_HREF = '/auth';
