'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Plus } from 'lucide-react';
import SharedMark from './SharedMark';
import { usePlansList, type PlanFilter } from './PlansList';
import { FOCUS_RING } from '@/lib/chrome';
import { cn } from '@/lib/utils';

const FILTERS: { value: PlanFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'project', label: 'Projects' },
  { value: 'learning', label: 'Learning' },
];

/** What the list says when the filter matches nothing (R31, FS-S6WVY R40). */
const EMPTY: Record<PlanFilter, string> = {
  all: 'No plans yet',
  project: 'No projects yet',
  learning: 'No learning plans yet',
};

/**
 * The plans panel's body: the All / Projects / Learning filter, the plan list
 * (with the shared mark) and "New plan" pinned at the bottom. The dock and the
 * narrow-width drawer both render it, so its look is defined once (FS-S6WVY
 * R25–R31, FS-49CQ2 R4). The list and the filter come from the shell's
 * `PlansListProvider`, so both surfaces show the same ones. The caller supplies
 * the surface it sits on, and the dock indents the title past its toggle.
 */
export default function PlansPanel({ titleClassName }: { titleClassName?: string }) {
  const pathname = usePathname();
  const planId = pathname?.split('/plan/')?.[1] || '';

  const { plans, isLoading, error, activeFilter, setActiveFilter } = usePlansList();

  const filteredPlans = plans
    // All is every plan in API order, unknown types included (FS-S6WVY R39).
    .filter((plan) => activeFilter === 'all' || plan.planType === activeFilter)
    .map((plan) => ({
      name: plan.name,
      href: `/plan/${plan.id}`,
      isActive: plan.id === planId,
      relation: plan.relation,
    }));

  return (
    <>
      {/* Header: the title on the toggle's row, then the filter across the
          panel's full width, so three labels never crowd beside the toggle. */}
      <div className="shrink-0 px-3 pb-3 pt-3">
        <h2
          className={cn(
            'flex h-8 items-center text-sm font-semibold text-foreground',
            titleClassName ?? 'px-3',
          )}
        >
          Plans
        </h2>
        <div className="mt-2 flex rounded-control bg-muted p-0.5">
          {FILTERS.map(({ value, label }) => {
            const isActive = activeFilter === value;
            return (
              <button
                key={value}
                type="button"
                aria-pressed={isActive}
                onClick={() => setActiveFilter(value)}
                className={cn(
                  // Sized to the label, the leftover shared: All stays
                  // compact and the longer labels keep their breathing room.
                  'h-7 flex-auto whitespace-nowrap rounded-control px-2 text-xs font-medium transition-colors',
                  FOCUS_RING,
                  isActive
                    ? 'bg-background text-foreground shadow-sm dark:bg-card'
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
            <li className="px-3 py-2 text-sm text-muted-foreground">{EMPTY[activeFilter]}</li>
          ) : (
            filteredPlans.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={item.isActive ? 'page' : undefined}
                  className={cn(
                    'relative flex items-center rounded-control px-3 py-2 text-sm transition-colors hover:bg-muted focus-visible:ring-inset',
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
            'flex w-full items-center justify-center gap-2 rounded-control border border-border px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted',
            FOCUS_RING,
          )}
        >
          <Plus aria-hidden className="h-4 w-4" />
          New plan
        </Link>
      </div>
    </>
  );
}
