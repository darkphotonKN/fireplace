'use client';

import { cn } from '@/lib/utils';
import Logo from '@/components/Logo';
import ThemeToggle from '@/components/ThemeToggle';
import CtaLink, { SIGN_IN_HREF, SIGN_UP_HREF } from './CtaLink';
import { useScrolledPast } from './motion/useScrolledPast';

/**
 * Fraction of the viewport scrolled before the bar arrives. Slightly under the
 * hero's own height, so it fades in as the hero leaves rather than after a gap.
 */
const REVEAL_AT = 0.8;

/**
 * x.ai-sized from `sm` up: smaller than the hero's pills. Below `sm` the
 * padding tightens and the wordmark drops, so logo, toggle and both pills
 * fit one line on a 320px phone. `whitespace-nowrap shrink-0` keep a pill
 * from wrapping its label or being squeezed when space runs short.
 */
const BAR_PILL = 'shrink-0 whitespace-nowrap px-3 py-2 text-sm sm:px-4';

/**
 * The tour's own chrome.
 *
 * Absent over the hero — the first impression is the fire and nothing else — and
 * fading in past it so sign-up and sign-in stay one click away through the
 * grid and the close. No profile menu: signed-in visitors never see the
 * landing. `LayoutContent` short-circuits for the logged-out home route and
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
      <div className="mx-auto flex w-full max-w-5xl items-center justify-between px-4 py-3 sm:px-6">
        <span className="flex items-center gap-2 text-base sm:text-lg">
          <Logo />
          <span className="hidden text-base font-bold text-foreground sm:inline sm:text-lg">Fireplace</span>
        </span>

        <span className="flex items-center gap-2">
          <ThemeToggle />
          <CtaLink href={SIGN_IN_HREF} variant="muted" className={BAR_PILL}>
            Sign in
          </CtaLink>
          <CtaLink href={SIGN_UP_HREF} className={BAR_PILL}>
            Start your plan
          </CtaLink>
        </span>
      </div>
    </div>
  );
}
