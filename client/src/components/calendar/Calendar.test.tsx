import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';

/**
 * The plan calendar, rendering what the API actually returns (I-0060).
 *
 * This is the assertion the parser fix needed and a unit test could not make.
 * `parseISODate` returning null is silent: `layoutItem` reads it as an item
 * with no dates, gives it shape "none", and `WeekRow` filters it out — so a
 * calendar full of scheduled items draws an empty grid and nothing anywhere
 * reports an error. The fixtures below carry RFC3339 dates, the shape a
 * `*time.Time` reaches the client in, and what is pinned is that a bar and a
 * chip are actually on the page.
 */

const getPlanCalendar = vi.fn();

vi.mock('@/api/calendar', () => ({
  getPlanCalendar: (...a: unknown[]) => getPlanCalendar(...a),
}));

import { Calendar } from './Calendar';

// March 2026, so the month window is fixed and every fixture below sits in it.
const NOW = new Date(2026, 2, 10, 9, 0);

const items = [
  {
    id: 'i1',
    description: 'Write the integration test plan',
    scope: 'longterm',
    done: false,
    startDate: '2026-03-09T00:00:00Z',
    dueDate: '2026-03-13T00:00:00Z',
  },
  {
    id: 'i2',
    description: 'Ship the parser fix',
    scope: 'longterm',
    done: false,
    startDate: '2026-03-16T00:00:00Z',
    dueDate: '2026-03-16T00:00:00Z',
  },
];

describe('Calendar', () => {
  beforeEach(() => {
    // shouldAdvanceTime: the fetch effect resolves on a real microtask, and
    // findBy* waits on timers.
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(NOW);
    getPlanCalendar.mockResolvedValue({
      planId: 'plan-1',
      view: 'month',
      windowStart: '2026-03-01',
      windowEnd: '2026-03-31',
      items,
    });
  });

  afterEach(() => vi.useRealTimers());

  it('should draw a bar and a chip for items carrying the wire format', async () => {
    render(<Calendar planId="plan-1" />);

    expect(await screen.findByText('Write the integration test plan')).toBeInTheDocument();
    expect(await screen.findByText('Ship the parser fix')).toBeInTheDocument();
  });

  it('should span the bar across the days its range covers', async () => {
    render(<Calendar planId="plan-1" />);

    const bar = await screen.findByText('Write the integration test plan');
    // WeekRow places a bar on the overlay grid: the 9th is a Monday, so the
    // range 9th–13th starts in column 2 and spans five days. A range read as
    // no dates at all would not be on the page to ask.
    const placed = bar.closest('[style*="grid-column"]') as HTMLElement;
    expect(placed.style.gridColumn).toBe('2 / span 5');
  });

  it('should place a one-day item on its own day as a single column', async () => {
    render(<Calendar planId="plan-1" />);

    const chip = await screen.findByText('Ship the parser fix');
    const placed = chip.closest('[style*="grid-column"]') as HTMLElement;
    // The 16th is a Monday too — column 2 of its own week row, one day wide.
    expect(placed.style.gridColumn).toBe('2 / span 1');
  });

  it('should draw nothing for an item the API returns with no dates', async () => {
    getPlanCalendar.mockResolvedValue({
      planId: 'plan-1',
      view: 'month',
      windowStart: '2026-03-01',
      windowEnd: '2026-03-31',
      items: [
        { id: 'i3', description: 'Someday, maybe', scope: 'longterm', done: false, startDate: '', dueDate: '' },
      ],
    });

    render(<Calendar planId="plan-1" />);

    // The heading proves the calendar finished rendering, so the absence below
    // is a decision and not a race.
    expect(await screen.findByText('March 2026')).toBeInTheDocument();
    expect(screen.queryByText('Someday, maybe')).not.toBeInTheDocument();
  });
});
