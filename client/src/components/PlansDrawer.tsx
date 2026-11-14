'use client';

import { useCallback, useEffect, useRef, useState, type MouseEvent } from 'react';
import { usePathname } from 'next/navigation';
import { Menu } from 'lucide-react';
import { useMediaQuery } from './landing/motion/useMediaQuery';
import PlansPanel from './PlansPanel';
import { DOCKED_QUERY, FOCUS_RING } from '@/lib/chrome';
import { cn } from '@/lib/utils';

const DRAWER_ID = 'plans-drawer';

/**
 * The narrow-width plans drawer's open state (FS-49CQ2 R5–R7).
 *
 * Local to the page and never remembered: it is not the sidebar context and
 * never reads or writes `sidebar`, so the docked widths' choice has one writer
 * (FS-S6WVY R3). It records the route it opened on, so any navigation closes
 * it without an effect, and growing to docked width drops it, so shrinking
 * back finds it closed. `useMediaQuery` starts false, which only matters while
 * the drawer is closed anyway.
 */
export function usePlansDrawer() {
  const pathname = usePathname();
  const isDocked = useMediaQuery(DOCKED_QUERY);
  const [openedAt, setOpenedAt] = useState<string | null>(null);
  if (openedAt !== null && (isDocked || openedAt !== pathname)) setOpenedAt(null);
  const isOpen = !isDocked && openedAt === pathname;

  const buttonRef = useRef<HTMLButtonElement>(null);

  /** Close; `returnFocus` for the dismissals that leave the user where they were (R8). */
  const close = useCallback((returnFocus = false) => {
    setOpenedAt(null);
    if (returnFocus) buttonRef.current?.focus();
  }, []);

  const toggle = () => setOpenedAt(isOpen ? null : pathname);

  /** A tap on any link closes it, even one to the route already showing (R5). */
  const closeOnLinkTap = (event: MouseEvent) => {
    if (event.target instanceof Element && event.target.closest('a')) close();
  };

  return { isOpen, toggle, close, closeOnLinkTap, buttonRef };
}

export type PlansDrawerState = ReturnType<typeof usePlansDrawer>;

/** The top bar's plans control below md; the dock's toggle takes over at md (R3, R8, R10). */
export function PlansMenuButton({ drawer }: { drawer: PlansDrawerState }) {
  const label = drawer.isOpen ? 'Close plans panel' : 'Open plans panel';
  return (
    <button
      ref={drawer.buttonRef}
      type="button"
      onClick={drawer.toggle}
      aria-label={label}
      title={label}
      aria-expanded={drawer.isOpen}
      aria-controls={DRAWER_ID}
      className={cn(
        'flex h-10 w-10 shrink-0 items-center justify-center rounded-control text-muted-foreground transition-colors hover:bg-muted hover:text-foreground md:hidden',
        FOCUS_RING,
      )}
    >
      <Menu aria-hidden className="h-5 w-5" />
    </button>
  );
}

/**
 * The plans panel as an overlay drawer beneath the top bar, over a scrim
 * (FS-49CQ2 R3, R4, R8, R9). Only mounted while open, so closed it holds
 * nothing focusable; while open the page behind it doesn't scroll and the
 * caller makes main inert.
 */
export default function PlansDrawer({ drawer }: { drawer: PlansDrawerState }) {
  const { isOpen, close, closeOnLinkTap } = drawer;
  const panelRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    panelRef.current?.querySelector<HTMLElement>('button, a[href]')?.focus();

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') close(true);
    };
    document.addEventListener('keydown', onKeyDown);

    return () => {
      document.removeEventListener('keydown', onKeyDown);
      document.body.style.overflow = previousOverflow;
    };
  }, [isOpen, close]);

  return (
    <div id={DRAWER_ID} className="md:hidden">
      {isOpen && (
        <>
          {/* Dims the page below the top bar with the scrim token. */}
          <div
            aria-hidden
            data-testid="plans-drawer-scrim"
            onClick={() => close(true)}
            className="fp-plans-scrim fixed inset-x-0 bottom-0 top-16 z-30 bg-scrim"
          />
          <aside
            ref={panelRef}
            aria-label="Plans panel"
            onClick={closeOnLinkTap}
            className="fp-plans-drawer fixed bottom-0 left-0 top-16 z-30 flex flex-col border-r border-border bg-layout shadow-lg"
          >
            <PlansPanel />
          </aside>
        </>
      )}
    </div>
  );
}
