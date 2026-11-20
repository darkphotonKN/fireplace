import { describe, it, expect, vi, beforeEach } from 'vitest';

const api = vi.hoisted(() => ({ GET: vi.fn(), POST: vi.fn(), PUT: vi.fn(), DELETE: vi.fn() }));

vi.mock('./client', () => ({
  api,
  apiErrorFrom: (error: { code: string }, status: number) =>
    Object.assign(new Error(`request failed with ${status}`), { code: error.code, status }),
}));

import { setDayPlanSettings } from './insights';

/** FS-02JBZ §Change: Hours a day, §API surface: setDayPlanSettings. */
const PATH = '/api/insights/day-plan/settings';

beforeEach(() => {
  api.PUT.mockReset();
});

describe('setDayPlanSettings', () => {
  it('should put the plan and its daily minutes and resolve to the saved setting', async () => {
    const saved = { dailyMinutes: 120, asked: true };
    api.PUT.mockResolvedValue({ data: saved, error: undefined, response: { status: 200 } });
    await expect(setDayPlanSettings('plan-1', 120)).resolves.toEqual(saved);
    expect(api.PUT).toHaveBeenCalledWith(PATH, { body: { planId: 'plan-1', dailyMinutes: 120 } });
  });

  it('should send an explicit null to go back to the default, never omit it', async () => {
    api.PUT.mockResolvedValue({ data: { asked: true }, error: undefined, response: { status: 200 } });
    await expect(setDayPlanSettings('plan-1', null)).resolves.toEqual({ asked: true });
    const [, init] = api.PUT.mock.calls[0];
    expect(init.body).toHaveProperty('dailyMinutes', null);
  });

  it('should throw the problem with its code', async () => {
    api.PUT.mockResolvedValue({ data: undefined, error: { code: 'NOT_FOUND' }, response: { status: 404 } });
    await expect(setDayPlanSettings('plan-1', 60)).rejects.toMatchObject({ code: 'NOT_FOUND', status: 404 });
  });
});
