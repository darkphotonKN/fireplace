'use client';

import { cn } from '@/lib/utils';
import Logo from '@/components/Logo';
import ThemeToggle from '@/components/ThemeToggle';
import CtaLink, { SIGN_UP_HREF } from './CtaLink';
import { useScrolledPast } from './motion/useScrolledPast';

/**
 * Fraction of the viewport scrolled before the bar arrives. Slightly under the
 * hero's own height, so it fades in as the hero leaves rather than after a gap.
 */
const REVEAL_AT = 0.8;

/**
 * The tour's own chrome.
 *
 * Absent over the hero — the first impression is the fire and nothing else — and
 * fading in past it so the CTA stays one click away through the remaining five
 * sections. `LayoutContent` short-circuits for the logged-out home route and
 * supplies no chrome at all, so this is the landing's, not the app's.
 */
export default function ChromeBar() {
  const visible = useScrolledPast(REVEAL_AT);

  return (
    <div
      data-testid="landing-chrome"
      // `fixed`, so arriving and leaving never reflows the sections behind it.
      // `inert` while hidden for the same reason the sections use it: an
      // invisible bar must not hand a keyboard visitor an invisible CTA.
      inert={!visible}
      className={cn(
        'fixed inset-x-0 top-0 z-40 border-b backdrop-blur-sm transition-opacity duration-300',
        'border-border bg-background/85',
        visible ? 'opacity-100' : 'pointer-events-none opacity-0'
      )}
    >
      <div className="mx-auto flex w-full max-w-5xl items-center justify-between px-6 py-3">
        <span className="flex items-center gap-2">
          <Logo />
          <span className="text-lg font-bold text-foreground">Fireplace</span>
        </span>

        <span className="flex items-center gap-3">
          <ThemeToggle />
          <CtaLink href={SIGN_UP_HREF} className="px-5 py-2 text-sm">
            Start your plan
          </CtaLink>
        </span>
      </div>
    </div>
  );
}
