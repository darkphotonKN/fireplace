'use client';

import { useState, useEffect, useLayoutEffect, useRef } from 'react';
import { usePathname } from 'next/navigation';
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { useSidebar } from './LayoutWrapper';
import PlansPanel from './PlansPanel';
import { FOCUS_RING } from '@/lib/chrome';
import { useMediaQuery } from './landing/motion/useMediaQuery';
import { cn } from '@/lib/utils';

const PANEL_ID = 'plans-panel';

/** Hover intent: a cursor crossing the rail doesn't flash the panel (FS-S6WVY R22). */
const PEEK_DELAY_MS = 150;

/**
 * Only a device that can really hover peeks (FS-49CQ2 R11): a tap on a touch
 * screen fires a synthetic `mouseenter`, and the overlay it opened would only
 * close on a `mouseleave` that never comes.
 */
const CAN_HOVER_QUERY = '(hover: hover)';

/**
 * Stamp when the panel was last seen open, for the discovery tip's 24h reminder
 * (R9). Not the remembered state; blocked storage just skips it (R8).
 */
function stampLastOpen() {
  try {
    localStorage.setItem('lastSidebarOpen', new Date().toISOString());
  } catch {
    // Blocked: the reminder may come a little early, which is harmless.
  }
}

export default function Sidebar() {
  const pathname = usePathname();
  const { isCollapsed, setIsCollapsed } = useSidebar();

  // The peek: the hidden panel shown over the page while the pointer rests on
  // the rail. Local to the sidebar and never remembered (R21-R24). It records
  // the route it opened on, so navigating closes it, as does the panel opening
  // (here or in another tab), without an effect.
  const [peekedAt, setPeekedAt] = useState<string | null>(null);
  if (peekedAt !== null && (!isCollapsed || peekedAt !== pathname)) setPeekedAt(null);
  const isPeeking = isCollapsed && peekedAt === pathname;
  const peekTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  // Starts false and stays false when the answer is unknown: no peek.
  const canHover = useMediaQuery(CAN_HOVER_QUERY);

  const closePeek = () => {
    clearTimeout(peekTimer.current);
    setPeekedAt(null);
  };

  const startPeek = () => {
    if (!isCollapsed || !canHover) return;
    clearTimeout(peekTimer.current);
    peekTimer.current = setTimeout(() => {
      setPeekedAt(pathname);
      stampLastOpen();
    }, PEEK_DELAY_MS);
  };

  useEffect(() => () => clearTimeout(peekTimer.current), []);

  // When the panel goes out of reach (a peek closes, or it's hidden from
  // another tab) with focus inside, hand focus to the toggle rather than let
  // it drop to the page as the panel turns inert (R20, R22). A layout effect,
  // so it runs before the browser's own inert focus fix-up.
  const isPanelReachable = !isCollapsed || isPeeking;
  const panelRef = useRef<HTMLElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);
  useLayoutEffect(() => {
    if (isPanelReachable) return;
    if (panelRef.current?.contains(document.activeElement)) toggleRef.current?.focus();
  }, [isPanelReachable]);

  useEffect(() => {
    if (!isPeeking) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setPeekedAt(null);
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [isPeeking]);

  const toggle = () => {
    const next = !isCollapsed;
    setIsCollapsed(next);
    closePeek();
    if (!next) {
      stampLastOpen();
    }
  };

  const toggleLabel = isCollapsed ? 'Show plans panel' : 'Hide plans panel';

  return (
    // The dock: beneath the top bar, on the page's own surface. Its width (16rem
    // open, a 3rem rail hidden) comes from `<html data-sidebar>` in globals.css,
    // as does main's matching offset, so the first frame is already right and
    // the two animate together (FS-S6WVY R5, R15–R19).
    <div
      data-peek={isPeeking}
      className="fp-sidebar-dock fixed bottom-0 left-0 top-16 z-20 border-r border-border bg-layout"
      onMouseEnter={startPeek}
      onMouseLeave={closePeek}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) closePeek();
      }}
    >
      {/* The one toggle, at the top of the column in both states (R16). Both
          icons render and CSS shows the one for the painted state, so a hidden
          load has the right icon before React knows the state. */}
      <button
        ref={toggleRef}
        type="button"
        onClick={toggle}
        aria-label={toggleLabel}
        title={toggleLabel}
        aria-expanded={!isCollapsed}
        aria-controls={PANEL_ID}
        className={cn(
          'absolute left-2 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground',
          FOCUS_RING,
        )}
      >
        <PanelLeftClose aria-hidden className="fp-sidebar-when-open h-4 w-4" />
        <PanelLeftOpen aria-hidden className="fp-sidebar-when-hidden h-4 w-4" />
      </button>

      <aside
        id={PANEL_ID}
        aria-label="Plans panel"
        ref={panelRef}
        inert={!isPanelReachable}
        className={cn(
          'fp-sidebar-panel absolute inset-y-0 left-0 flex flex-col border-r border-border bg-layout',
          isPeeking && 'shadow-lg',
        )}
      >
        {/* The filter sits beside the toggle's spot. */}
        <PlansPanel headerClassName="pl-12 pr-3" />
      </aside>
    </div>
  );
}
