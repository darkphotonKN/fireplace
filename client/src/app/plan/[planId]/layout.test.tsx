import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useEffect } from 'react';
import { render, screen } from '@testing-library/react';

/**
 * The plan's surrounding blocks (FS-043WP R2, R3): title card and the Day
 * plan above the middle slot, Calendar and videos below. The slot swaps
 * between the Items card and a milestone page; nothing around it reloads or
 * remounts. The Day plan section runs for real (FS-02JBZ R62, R63) over
 * mocked reads, so "no daily <Todo>" and the tooltip are checked on the page.
 */

const getPlan = vi.fn();
const getSuggestedVideos = vi.fn();
const getDayPlan = vi.fn();
const planMyDay = vi.fn();
const fetchChecklist = vi.fn();
const mounts = { todo: 0, calendar: 0 };

vi.mock('next/navigation', () => ({ useParams: () => ({ planId: 'plan-1' }) }));
vi.mock('@/api/plans', () => ({ getPlan: (...a: unknown[]) => getPlan(...a) }));
vi.mock('@/api/insights', () => ({
  getSuggestedVideos: (...a: unknown[]) => getSuggestedVideos(...a),
  getDayPlan: (...a: unknown[]) => getDayPlan(...a),
  planMyDay: (...a: unknown[]) => planMyDay(...a),
}));
vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  createChecklistItem: vi.fn(),
}));
// The layout mounts no <Todo> at all now; this catches one coming back.
vi.mock('@/components/Todo', () => ({
  default: function MockTodo({ fixedTaskType }: { fixedTaskType?: string }) {
    useEffect(() => {
      mounts.todo += 1;
    }, []);
    return <div data-testid={`todo-${fixedTaskType ?? 'longterm'}`} />;
  },
}));
vi.mock('@/components/calendar/CalendarCard', () => ({
  CalendarCard: function MockCalendarCard({ planId }: { planId: string }) {
    useEffect(() => {
      mounts.calendar += 1;
    }, []);
    return <div data-testid="calendar">{planId}</div>;
  },
}));

import PlanLayout from './layout';
import { ApiError } from '@/api/client';

describe('PlanLayout', () => {
  beforeEach(() => {
    getPlan.mockReset().mockResolvedValue({
      id: 'plan-1',
      name: 'Ship the thing',
      description: 'get it out',
      focus: 'momentum',
    });
    getSuggestedVideos.mockReset().mockResolvedValue([]);
    getDayPlan.mockReset().mockResolvedValue({
      planId: 'plan-1',
      date: '2026-10-25',
      status: 'not_planned',
      refreshesRemaining: 3,
      groups: [],
    });
    planMyDay.mockReset();
    fetchChecklist.mockReset().mockResolvedValue([]);
    mounts.todo = 0;
    mounts.calendar = 0;
  });

  it('should hold the title card and the Day plan above the slot, Calendar and videos below', async () => {
    render(
      <PlanLayout>
        <div data-testid="slot" />
      </PlanLayout>
    );

    expect(
      await screen.findByRole('heading', { name: 'Ship the thing' })
    ).toBeInTheDocument();
    const order = [
      screen.getByRole('heading', { name: 'Ship the thing' }),
      screen.getByRole('region', { name: 'Day plan' }),
      screen.getByTestId('slot'),
      screen.getByTestId('calendar'),
      screen.getByRole('heading', { name: 'Recommended Videos' }),
    ];
    for (let i = 1; i < order.length; i++) {
      expect(
        order[i - 1].compareDocumentPosition(order[i]) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
    }
    // The Day plan is its own section, not a <Todo>; the long-term list is
    // the plan route's (FS-02JBZ R62).
    expect(mounts.todo).toBe(0);
    expect(screen.queryByTestId('todo-daily')).toBeNull();
    expect(screen.queryByTestId('todo-longterm')).toBeNull();
  });

  it('should give the Day plan an "i" tooltip about daily progression, and never plan on load (R63, D7)', async () => {
    render(
      <PlanLayout>
        <div data-testid="slot" />
      </PlanLayout>
    );

    expect(
      await screen.findByRole('button', { name: 'Plan my day' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Information about the day plan' })
    ).toBeInTheDocument();
    expect(screen.getByRole('tooltip')).toHaveTextContent(/daily progression/i);
    expect(planMyDay).not.toHaveBeenCalled();
  });

  it('should swap only the slot: plan, Day plan, Calendar and videos load once (R3)', async () => {
    const { rerender } = render(
      <PlanLayout>
        <div data-testid="items" />
      </PlanLayout>
    );
    await screen.findByRole('heading', { name: 'Ship the thing' });

    rerender(
      <PlanLayout>
        <div data-testid="milestone" />
      </PlanLayout>
    );
    rerender(
      <PlanLayout>
        <div data-testid="items" />
      </PlanLayout>
    );

    expect(screen.getByTestId('items')).toBeInTheDocument();
    expect(getPlan).toHaveBeenCalledTimes(1);
    expect(getPlan).toHaveBeenCalledWith('plan-1');
    expect(getSuggestedVideos).toHaveBeenCalledTimes(1);
    expect(getDayPlan).toHaveBeenCalledTimes(1);
    expect(mounts).toEqual({ todo: 0, calendar: 1 });
  });

  // FS-12ZMF R32: a stranger, a removed member and a deleted plan all load as
  // 404, and the whole plan view gives way to one plain "not here".
  it('should replace the plan view with "This plan isn\'t here" on a 404', async () => {
    getPlan.mockReset().mockRejectedValue(new ApiError(undefined, 404));

    render(
      <PlanLayout>
        <div data-testid="slot" />
      </PlanLayout>
    );

    expect(
      await screen.findByRole('heading', { name: "This plan isn't here" })
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /home/i })).toHaveAttribute('href', '/');
    expect(screen.queryByTestId('slot')).toBeNull();
    expect(screen.queryByRole('region', { name: 'Day plan' })).toBeNull();
    expect(screen.queryByTestId('calendar')).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Recommended Videos' })).toBeNull();
  });

  it('should keep the plan view on any other failure', async () => {
    getPlan.mockReset().mockRejectedValue(new ApiError(undefined, 500));

    render(
      <PlanLayout>
        <div data-testid="slot" />
      </PlanLayout>
    );

    expect(await screen.findByRole('heading', { name: 'Plan Details' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: "This plan isn't here" })).toBeNull();
    expect(screen.getByTestId('slot')).toBeInTheDocument();
  });
});
