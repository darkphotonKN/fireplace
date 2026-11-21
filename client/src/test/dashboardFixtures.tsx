import { format } from 'date-fns';
import { render } from '@testing-library/react';
import type { Plan } from '@/api/plans';
import type { ChecklistItem } from '@/api/checklists';

/**
 * Fixtures and the render helper for the `Dashboard.*.test.tsx` files, with no
 * mocks of their own. `Dashboard.tick.test.tsx` mocks `@/api/client` rather
 * than the API layer, so it takes these and not `dashboardHarness`, which
 * mocks the API layer on import.
 */

/** A project plan named `Plan ${id}`, updated at `updatedAt`. */
export const plan = (
  id: string,
  updatedAt = '2026-09-01T00:00:00Z',
  extra: Partial<Plan> = {},
): Plan =>
  ({
    id,
    name: `Plan ${id}`,
    focus: `focus of ${id}`,
    description: '',
    planType: 'project',
    dailyReset: false,
    userId: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt,
    ...extra,
  }) as Plan;

/** A long-term, live, undone top-level task described as its id. */
export const item = (id: string, extra: Partial<ChecklistItem> = {}): ChecklistItem =>
  ({
    id,
    planId: 'p',
    description: id,
    done: false,
    archived: false,
    scope: 'longterm',
    type: 'task',
    sequence: '1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...extra,
  }) as ChecklistItem;

/** `count` plans p0..p{count-1}, oldest first in the response so ordering has to be earned. */
export const plansAged = (count: number) =>
  Array.from({ length: count }, (_, i) =>
    plan(`p${i}`, `2026-01-${String(i + 1).padStart(2, '0')}T00:00:00Z`),
  );

/** What `GET /api/insights/day-plan` returns for a day nobody has planned. */
export const notPlanned = (planId: string) => ({
  planId,
  date: format(new Date(), 'yyyy-MM-dd'),
  status: 'not_planned',
  groups: [],
  refreshesRemaining: 3,
});

/**
 * Renders home. Dashboard is imported here, at render time, so whatever the
 * test file or harness mocked is in place before it first loads.
 *
 * `onStartPlan` belongs to the first-run invitation (I-KSJFR-7): it puts the
 * focus prompt back on screen. Nothing in these files exercises that path, so
 * they take a noop and the prop stays required on the real component.
 */
export async function renderHome() {
  const Dashboard = (await import('@/components/home/Dashboard')).default;
  return render(<Dashboard onStartPlan={() => {}} />);
}
