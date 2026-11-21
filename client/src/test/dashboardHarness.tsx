import { beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { notPlanned, renderHome } from './dashboardFixtures';

export * from './dashboardFixtures';

/**
 * The shared harness for `Dashboard.test.tsx` and `Dashboard.milestones.test.tsx`.
 *
 * Importing this module stubs the API layer home reads and writes through
 * (`@/api/plans`, `@/api/checklists`, `@/api/insights`) and the toast, and
 * resets them before each test. Each stub is a `vi.fn()` exported on `api`:
 *
 *   import { api, plan, renderHome } from '@/test/dashboardHarness';
 *   api.listPlans.mockResolvedValue([plan('a')]);
 *
 * Defaults: no items, every day not planned, every write accepted.
 * `listPlans` has none, so each test says what the account holds.
 */
// Hoisted so the mock factories below can hand these out; vitest won't export a
// hoisted binding directly, hence the re-export.
const stubs = vi.hoisted(() => ({
  api: {
    listPlans: vi.fn(),
    listChecklists: vi.fn(),
    updateChecklistItem: vi.fn(),
    getDayPlan: vi.fn(),
    planMyDay: vi.fn(),
  },
  toast: vi.fn(),
}));
export const { api, toast } = stubs;

vi.mock('@/api/plans', () => ({ listPlans: stubs.api.listPlans }));
vi.mock('@/api/checklists', () => ({
  listChecklists: stubs.api.listChecklists,
  updateChecklistItem: stubs.api.updateChecklistItem,
}));
vi.mock('@/api/insights', () => ({
  getDayPlan: stubs.api.getDayPlan,
  planMyDay: stubs.api.planMyDay,
}));
vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: stubs.toast }),
}));

beforeEach(() => {
  Object.values(api).forEach((fn) => fn.mockReset());
  toast.mockReset();
  api.listChecklists.mockResolvedValue([]);
  api.getDayPlan.mockImplementation((planId: string) => Promise.resolve(notPlanned(planId)));
  api.updateChecklistItem.mockResolvedValue({});
});

/** Renders home and waits for the Continue section, so the plans have landed. */
export async function renderHomeWithPlans() {
  const view = await renderHome();
  await screen.findByRole('heading', { name: /continue where you left off/i });
  return view;
}

/** A Continue card's progress slot: done/total, the milestone strip, or empty. */
export const progressSlot = (planId: string) =>
  screen.getByTestId(`continue-progress-${planId}`);
