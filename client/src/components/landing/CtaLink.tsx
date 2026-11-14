'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

type CtaVariant = 'primary' | 'muted' | 'quiet';

const PILL = 'inline-flex items-center justify-center rounded-control px-8 py-3 font-medium transition-colors';

const VARIANTS: Record<CtaVariant, string> = {
  // The solid pill. `--action` (jet in light, the theme's own in dark), not
  // coral: coral is reserved for the hero's one underline. Text from the
  // paired token, never a hardcoded white.
  primary: cn(PILL, 'bg-action text-action-foreground hover:bg-action/90'),
  // The grey pill: the quieter of a matched pair, still a button in shape.
  muted: cn(PILL, 'bg-muted text-foreground hover:bg-muted/70'),
  quiet: 'text-sm text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline',
};

/**
 * The landing's calls to action. One definition so the solid pill looks the
 * same in the hero, the chrome bar and the close, and so its colour comes from
 * the `--action` token in exactly one place rather than being re-typed per CTA.
 * Callers resize with `className` (e.g. the chrome bar's `px-4 py-2 text-sm`).
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
