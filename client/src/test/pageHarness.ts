import { beforeEach, vi } from 'vitest';
import type { ChecklistItem } from '@/api/checklists';

/**
 * The shared harness for the single-item page tests (MilestonePage, NotePage).
 *
 * Importing this module mocks the item calls both pages make through
 * `@/services/api`, and the router, and resets them (and any console spy)
 * before each test. Each mock is a `vi.fn()` exported here, so a test
 * configures or inspects it directly:
 *
 *   import { api, item, router } from '@/test/pageHarness';
 *   api.fetchChecklist.mockResolvedValue([item('m1', 'Learn Go')]);
 *
 * A reset mock answers undefined: each file sets the answers it relies on in
 * its own beforeEach. Modules only one page talks to (duplicate, insights,
 * comments) stay mocked in that page's file.
 */

export const api = {
  fetchChecklist: vi.fn(),
  updateChecklistItem: vi.fn(),
  createChecklistItem: vi.fn(),
  deleteChecklistItem: vi.fn(),
  archiveChecklistItem: vi.fn(),
  updateChecklistDates: vi.fn(),
};

export const router = { push: vi.fn(), replace: vi.fn(), refresh: vi.fn() };

vi.mock('next/navigation', () => ({
  useRouter: () => router,
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => api.fetchChecklist(...a),
  updateChecklistItem: (...a: unknown[]) => api.updateChecklistItem(...a),
  createChecklistItem: (...a: unknown[]) => api.createChecklistItem(...a),
  deleteChecklistItem: (...a: unknown[]) => api.deleteChecklistItem(...a),
  archiveChecklistItem: (...a: unknown[]) => api.archiveChecklistItem(...a),
  updateChecklistDates: (...a: unknown[]) => api.updateChecklistDates(...a),
}));

beforeEach(() => {
  vi.restoreAllMocks();
  Object.values(api).forEach((fn) => fn.mockReset());
  Object.values(router).forEach((fn) => fn.mockReset());
});

/** A live long-term task on plan-1, edited just now, unless overridden. */
export const item = (
  id: string,
  description: string,
  extra: Partial<ChecklistItem> = {}
) =>
  ({
    id,
    description,
    done: false,
    scope: 'longterm',
    type: 'task',
    planId: 'plan-1',
    archived: false,
    updatedAt: new Date().toISOString(),
    ...extra,
  }) as ChecklistItem;

/** Silences the console.error an expected failure logs; restored before the next test. */
export const quietErrors = () => vi.spyOn(console, 'error').mockImplementation(() => {});
