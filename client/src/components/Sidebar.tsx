'use client';

import { useState, useEffect, useLayoutEffect, useRef } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { PanelLeftClose, PanelLeftOpen, Plus } from 'lucide-react';
import { listPlans } from '@/api/plans';
import { useSidebar } from './LayoutWrapper';
import SharedMark from './SharedMark';
import { cn } from '@/lib/utils';

interface Plan {
  id: string;
  name: string;
  planType: string;
  focus?: string;
  description?: string;
  relation?: 'owner' | 'member';
}

type PlanFilter = 'learning' | 'project';

const PANEL_ID = 'plans-panel';

/** Hover intent: a cursor crossing the rail doesn't flash the panel (FS-S6WVY R22). */
const PEEK_DELAY_MS = 150;

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

/** The coral focus ring every sidebar control shares (FS-S6WVY R30). */
const FOCUS_RING = 'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring';

const FILTERS: { value: PlanFilter; label: string }[] = [
  { value: 'project', label: 'Projects' },
  { value: 'learning', label: 'Learning' },
];

export default function Sidebar() {
  const pathname = usePathname();
  const planId = pathname?.split('/plan/')?.[1] || '';
  const { isCollapsed, setIsCollapsed } = useSidebar();

  // The peek: the hidden panel shown over the page while the pointer rests on
  // the rail. Local to the sidebar and never remembered (R21-R24). It records
  // the route it opened on, so navigating closes it, as does the panel opening
  // (here or in another tab), without an effect.
  const [peekedAt, setPeekedAt] = useState<string | null>(null);
  if (peekedAt !== null && (!isCollapsed || peekedAt !== pathname)) setPeekedAt(null);
  const isPeeking = isCollapsed && peekedAt === pathname;
  const peekTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const closePeek = () => {
    clearTimeout(peekTimer.current);
    setPeekedAt(null);
  };

  const startPeek = () => {
    if (!isCollapsed) return;
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

  const [plans, setPlans] = useState<Plan[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeFilter, setActiveFilter] = useState<PlanFilter>('project');

  useEffect(() => {
    const fetchPlans = async () => {
      setIsLoading(true);
      setError(null);

      try {
        // Bare list, not data.result — this endpoint is serialized (FS-0004).
        setPlans(await listPlans());
      } catch (err) {
        console.error('Error fetching plans:', err);
        setError('Failed to load plans');
        setPlans([]);
      } finally {
        setIsLoading(false);
      }
    };

    fetchPlans();
  }, []);

  const filteredPlans = plans
    .filter((plan) => plan.planType === activeFilter)
    .map((plan) => ({
      name: plan.name,
      href: `/plan/${plan.id}`,
      isActive: plan.id === planId,
      relation: plan.relation,
    }));

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
        {/* Header: the filter, beside the toggle's spot. */}
        <div className="shrink-0 pb-3 pl-12 pr-3 pt-3">
          <div className="flex rounded-md bg-muted p-0.5">
            {FILTERS.map(({ value, label }) => {
              const isActive = activeFilter === value;
              return (
                <button
                  key={value}
                  type="button"
                  aria-pressed={isActive}
                  onClick={() => setActiveFilter(value)}
                  className={cn(
                    'h-7 flex-1 rounded-sm text-xs font-medium transition-colors',
                    FOCUS_RING,
                    isActive
                      ? 'bg-card text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Plan list: scrolls inside the panel; New plan stays pinned below. */}
        <nav aria-label="Plans" className="min-h-0 flex-1 overflow-y-auto px-3">
          <ul className="space-y-0.5">
            {isLoading ? (
              <li className="px-3 py-2 text-sm text-muted-foreground">Loading…</li>
            ) : error ? (
              <li className="px-3 py-2 text-sm text-muted-foreground">{error}</li>
            ) : filteredPlans.length === 0 ? (
              <li className="px-3 py-2 text-sm text-muted-foreground">
                No {activeFilter === 'project' ? 'projects' : 'learning plans'} yet
              </li>
            ) : (
              filteredPlans.map((item) => (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    aria-current={item.isActive ? 'page' : undefined}
                    className={cn(
                      'relative flex items-center rounded-md px-3 py-2 text-sm transition-colors hover:bg-muted focus-visible:ring-inset',
                      FOCUS_RING,
                      item.isActive ? 'font-medium text-primary' : 'text-foreground',
                    )}
                  >
                    {/* The view's one coral moment (R27). */}
                    {item.isActive && (
                      <span
                        aria-hidden
                        className="absolute inset-y-1.5 left-0 w-0.5 rounded-full bg-primary"
                      />
                    )}
                    <span className="min-w-0 truncate">{item.name}</span>
                    <SharedMark plan={item} className="ml-2" />
                  </Link>
                </li>
              ))
            )}
          </ul>
        </nav>

        <div className="shrink-0 p-3">
          <Link
            href="/create-plan"
            className={cn(
              'flex w-full items-center justify-center gap-2 rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted',
              FOCUS_RING,
            )}
          >
            <Plus aria-hidden className="h-4 w-4" />
            New plan
          </Link>
        </div>
      </aside>
    </div>
  );
}
