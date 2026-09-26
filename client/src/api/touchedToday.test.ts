import { describe, it, expect, vi, beforeEach } from 'vitest';
import { wasTouchedToday } from '@/lib/touchedToday';

/**
 * FS-KSJFR R11–R14: the touched-today stamp is written here, in the API layer,
 * because this is the one place every mutation already passes through. A new
 * mutation that forgets the stamp is the bug this placement prevents — so the
 * cover below is a row per mutating function, not a sample.
 */

const api = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
  PATCH: vi.fn(),
  DELETE: vi.fn(),
}));

vi.mock('./client', () => ({
  api,
  apiErrorFrom: (_error: unknown, status: number) =>
    new Error(`request failed with ${status}`),
  ApiError: class ApiError extends Error {},
}));

import {
  createPlan,
  updatePlan,
  deletePlan,
  toggleDailyReset,
  listPlans,
  getPlan,
  listSharedPlans,
  searchPlans,
} from './plans';
import {
  createChecklistItem,
  updateChecklistItem,
  updateChecklistDates,
  reorderChecklists,
  archiveChecklistItem,
  deleteChecklistItem,
  listChecklists,
  listArchivedChecklists,
  listUpcomingChecklists,
  getChecklist,
} from './checklists';
import { getProfile } from './profile';

const succeed = () => {
  const ok = { data: [], error: undefined, response: { status: 200 } };
  for (const verb of [api.GET, api.POST, api.PATCH, api.DELETE]) {
    verb.mockResolvedValue(ok);
  }
};

const reject = () => {
  const failed = {
    data: undefined,
    error: { code: 'VALIDATION_FAILED' },
    response: { status: 422 },
  };
  for (const verb of [api.GET, api.POST, api.PATCH, api.DELETE]) {
    verb.mockResolvedValue(failed);
  }
};

/**
 * The call never reaching a server at all — offline, DNS, a cut connection.
 * `openapi-fetch` hands that back as a rejected promise rather than as an
 * `{error}` field, so it is a different path through every function below and
 * the stamp has to stay unwritten on both (R14).
 */
const crash = () => {
  for (const verb of [api.GET, api.POST, api.PATCH, api.DELETE]) {
    verb.mockRejectedValue(new TypeError('Failed to fetch'));
  }
};

const mutations: [string, () => Promise<unknown>][] = [
  ['createPlan', () => createPlan({ name: 'n', focus: 'f', planType: 'personal' })],
  ['updatePlan', () => updatePlan('plan-1', { name: 'n' })],
  ['deletePlan', () => deletePlan('plan-1')],
  ['toggleDailyReset', () => toggleDailyReset('plan-1')],
  ['createChecklistItem', () => createChecklistItem('plan-1', { description: 'd' })],
  ['updateChecklistItem', () => updateChecklistItem('plan-1', 'item-1', { done: true })],
  ['updateChecklistDates', () => updateChecklistDates('plan-1', 'item-1', { dueDate: '2026-03-09' })],
  [
    'reorderChecklists',
    () => reorderChecklists('plan-1', { ids: ['item-1'], parentId: null, scope: 'daily' }),
  ],
  ['archiveChecklistItem', () => archiveChecklistItem('plan-1', 'item-1', true)],
  ['deleteChecklistItem', () => deleteChecklistItem('plan-1', 'item-1')],
];

const reads: [string, () => Promise<unknown>][] = [
  ['listPlans', () => listPlans()],
  ['getPlan', () => getPlan('plan-1')],
  ['listSharedPlans', () => listSharedPlans()],
  ['searchPlans', () => searchPlans('term')],
  ['listChecklists', () => listChecklists('plan-1')],
  ['listArchivedChecklists', () => listArchivedChecklists('plan-1')],
  ['listUpcomingChecklists', () => listUpcomingChecklists('plan-1')],
  ['getChecklist', () => getChecklist('plan-1', 'item-1')],
  ['getProfile', () => getProfile()],
];

describe('the touched-today stamp in the API layer', () => {
  beforeEach(() => {
    window.localStorage.clear();
    succeed();
  });

  it.each(mutations)('should record a touch after %s succeeds', async (_name, call) => {
    await call();

    expect(wasTouchedToday()).toBe(true);
  });

  it.each(mutations)('should record nothing when %s is rejected', async (_name, call) => {
    reject();

    await expect(call()).rejects.toThrow();
    expect(wasTouchedToday()).toBe(false);
  });

  it.each(mutations)(
    'should record nothing when %s never reaches the server',
    async (_name, call) => {
      crash();

      await expect(call()).rejects.toThrow(/failed to fetch/i);
      expect(wasTouchedToday()).toBe(false);
    },
  );

  it.each(reads)('should record nothing for %s — reading is not touching', async (_name, call) => {
    await call();

    expect(wasTouchedToday()).toBe(false);
  });
});
