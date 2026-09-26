import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusLine from './StatusLine';
import type { HomeWork } from '@/lib/homeWork';

/**
 * FS-KSJFR slice 4 (I-KSJFR-4): the dashboard's first line (R16–R17).
 *
 * These are copy assertions on purpose. The line is the product's one
 * statement of how far home looked, and "across 3 plans" disappearing would be
 * invisible to every other test in the suite.
 */

// A Monday, so the weekday in every expectation below is a fact and not a
// property of the machine running the test.
const MONDAY = new Date(2026, 2, 16, 9, 0);

const work = (over: Partial<HomeWork> = {}): HomeWork => ({
  status: 'ready',
  mode: 'today',
  rows: [],
  dueCount: 0,
  waitingCount: 0,
  planCount: 3,
  ...over,
});

beforeEach(() => vi.setSystemTime(MONDAY));
afterEach(() => vi.useRealTimers());

describe('StatusLine', () => {
  it('should say the weekday, the count due, and how many plans it came from', () => {
    render(<StatusLine work={work({ mode: 'today', dueCount: 2, planCount: 3 })} />);
    expect(screen.getByText('Monday · 2 due across 3 plans')).toBeInTheDocument();
  });

  it('should count one plan as one plan', () => {
    render(<StatusLine work={work({ mode: 'today', dueCount: 1, planCount: 1 })} />);
    expect(screen.getByText('Monday · 1 due across 1 plan')).toBeInTheDocument();
  });

  it('should say nothing is due, and how much is waiting, in the Next up case', () => {
    render(
      <StatusLine work={work({ mode: 'nextUp', waitingCount: 6, planCount: 3 })} />
    );
    expect(
      screen.getByText('Monday · nothing due — 6 waiting across 3 plans')
    ).toBeInTheDocument();
  });

  it('should drop the waiting clause when there is nothing waiting either', () => {
    render(<StatusLine work={work({ mode: 'nextUp', waitingCount: 0, planCount: 2 })} />);
    expect(screen.getByText('Monday · nothing due across 2 plans')).toBeInTheDocument();
  });

  it('should claim nothing about the day while the fan-out is in flight', () => {
    render(<StatusLine work={work({ status: 'loading', planCount: 1 })} />);
    expect(screen.getByText('Monday')).toBeInTheDocument();
  });

  it('should fall back to the weekday alone when no plan was read', () => {
    // `GET /api/plans` failed, or the account has none: "nothing due" would be
    // a statement about a day nobody looked at.
    render(<StatusLine work={work({ mode: 'nextUp', planCount: 0 })} />);
    expect(screen.getByText('Monday')).toBeInTheDocument();
  });

  it('should fall back to the weekday alone when every item fetch failed', () => {
    // §Edge States: all three item fetches fail. A count here would be a
    // number home never read.
    render(<StatusLine work={work({ status: 'failed', mode: 'today', dueCount: 0, planCount: 0 })} />);
    expect(screen.getByText('Monday')).toBeInTheDocument();
  });

  it('should carry its muted line on a <p> with the muted token and no dark: patch', () => {
    // The compliant spelling, which only became possible with I-0061: the
    // global element rules moved into `@layer base`, so `text-muted-foreground`
    // wins in both themes. Before that this line had to be a <div> to escape
    // `.dark p`. R50 still forbids solving it with a `dark:` variant.
    const { container } = render(<StatusLine work={work({ mode: 'today', dueCount: 2 })} />);
    const line = container.querySelector('p');
    expect(line?.className).toContain('text-muted-foreground');
    expect(container.innerHTML).not.toMatch(/dark:/);
  });

  it('should stay neutral — the coral belongs to the block below it (R49)', () => {
    const { container } = render(<StatusLine work={work({ mode: 'today', dueCount: 2 })} />);
    expect(container.querySelector('[class*="primary"]')).toBeNull();
  });
});
