import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { format } from 'date-fns';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/**
 * FS-KSJFR slice 2 (I-KSJFR-2): the dashboard shell and the capped fan-out.
 *
 * The load-bearing assertions here are about *request shape*, because that is
 * what ADR-0013 decided: four requests whatever the account holds, the three
 * item fetches in parallel, and one of them failing does not blank the page.
 * The rest — progress, partial failure, the retry — are what the user sees when
 * that shape holds or breaks.
 */

const listPlans = vi.fn();
const listChecklists = vi.fn();

vi.mock('@/api/plans', () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
}));
vi.mock('@/api/checklists', () => ({
  listChecklists: (...a: unknown[]) => listChecklists(...a),
}));

import Dashboard from './Dashboard';

/**
 * `onStartPlan` belongs to the first-run invitation (I-KSJFR-7) — it puts the
 * focus prompt back on screen. Nothing in these files exercises that path, so
 * they take a noop and the prop stays required on the real component.
 */
const Home = () => <Dashboard onStartPlan={() => {}} />;
import type { Plan } from '@/api/plans';
import type { ChecklistItem } from '@/api/checklists';

/** A promise a test resolves by hand, so "in flight" is an observable state. */
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  // Nothing is ever left unhandled: every deferred in this file is settled.
  return { promise, resolve, reject };
}

/**
 * Let everything already scheduled run, renders included.
 *
 * A request budget is an assertion about what does NOT happen, and `waitFor`
 * cannot make one: it resolves the instant the count it is waiting for is
 * reached, so a fetch issued on a later tick — a re-render re-running the
 * effect, a second fan-out — lands after the test has already looked. Settling
 * first gives the over-fetch its chance to happen and be counted.
 */
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 32));
  });

const plan = (id: string, updatedAt: string, extra: Partial<Plan> = {}): Plan =>
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

const item = (id: string, done: boolean): ChecklistItem =>
  ({
    id,
    planId: 'p',
    description: id,
    done,
    archived: false,
    scope: 'longterm',
    type: 'task',
    sequence: '1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  }) as ChecklistItem;

/** `count` plans, oldest first in the response so ordering has to be earned. */
const plansAged = (count: number) =>
  Array.from({ length: count }, (_, i) =>
    plan(`p${i}`, `2026-01-${String(i + 1).padStart(2, '0')}T00:00:00Z`),
  );

beforeEach(() => {
  listPlans.mockReset();
  listChecklists.mockReset();
  listChecklists.mockResolvedValue([]);
});

describe('Dashboard — the capped fan-out (ADR-0013)', () => {
  it.each([
    { plans: 1, requests: 2 },
    { plans: 3, requests: 4 },
    { plans: 4, requests: 4 },
    { plans: 12, requests: 4 },
  ])(
    'should issue $requests requests for an account with $plans plans',
    async ({ plans, requests }) => {
      listPlans.mockResolvedValue(plansAged(plans));

      render(<Home />);
      await screen.findByRole('heading', { name: /continue where you left off/i });
      await waitFor(() =>
        expect(listChecklists).toHaveBeenCalledTimes(requests - 1),
      );
      // The budget is a ceiling as much as a floor, so nothing is asserted
      // until every request this render could still issue has been issued.
      await settle();

      expect(listPlans).toHaveBeenCalledTimes(1);
      expect(listChecklists).toHaveBeenCalledTimes(requests - 1);
      expect(listChecklists.mock.calls.length + listPlans.mock.calls.length).toBe(
        requests,
      );
    },
  );

  it('should fetch items for the three most recently updated plans only', async () => {
    // 12 plans, oldest first: p11 / p10 / p9 are the three most recent.
    listPlans.mockResolvedValue(plansAged(12));

    render(<Home />);
    await waitFor(() => expect(listChecklists).toHaveBeenCalledTimes(3));
    await settle();

    expect(listChecklists.mock.calls.map((c) => c[0]).sort()).toEqual([
      'p10',
      'p11',
      'p9',
    ]);
  });

  it('should issue the three item fetches in parallel, not one after another', async () => {
    const pending = [deferred<ChecklistItem[]>(), deferred<ChecklistItem[]>(), deferred<ChecklistItem[]>()];
    let next = 0;
    listPlans.mockResolvedValue(plansAged(3));
    listChecklists.mockImplementation(() => pending[next++].promise);

    render(<Home />);

    // All three are out while none has settled — a sequential chain could only
    // ever have one call outstanding here.
    await waitFor(() => expect(listChecklists).toHaveBeenCalledTimes(3));
    pending.forEach((p) => p.resolve([]));
    await screen.findAllByText('0/0');
  });
});

describe('Dashboard — Continue where you left off', () => {
  it('should show the three most recent plans with name, focus, type and progress', async () => {
    listPlans.mockResolvedValue([
      plan('a', '2026-09-01T00:00:00Z', { name: 'Ship the thing', focus: 'get it out' }),
      plan('b', '2026-08-01T00:00:00Z'),
      plan('c', '2026-07-01T00:00:00Z'),
      plan('d', '2026-06-01T00:00:00Z', { name: 'Forgotten' }),
    ]);
    listChecklists.mockImplementation(async (id: string) =>
      id === 'a' ? [item('1', true), item('2', false), item('3', true)] : [],
    );

    render(<Home />);

    // Scoped to the Continue section on purpose: a plan name legitimately
    // appears more than once on the assembled page — Today's rows name the
    // plan they came from (R30), and the fourth plan shows under Your other
    // plans. This assertion is about the cards, so it asks the cards.
    const heading = await screen.findByText('Continue where you left off');
    const cards = within(heading.closest('section') as HTMLElement);

    expect(cards.getByText('Ship the thing')).toBeInTheDocument();
    expect(cards.getByText('get it out')).toBeInTheDocument();
    expect(cards.getAllByText(/project/i).length).toBeGreaterThan(0);
    expect(await cards.findByText('2/3')).toBeInTheDocument();
    // Only three cards; the fourth plan belongs to Your other plans (slice 3).
    expect(cards.queryByText('Forgotten')).toBeNull();
    expect(cards.getByRole('link', { name: /ship the thing/i })).toHaveAttribute(
      'href',
      '/plan/a',
    );
  });

  it('should render a plan with no items as 0/0 rather than hiding it', async () => {
    listPlans.mockResolvedValue([plan('empty', '2026-09-01T00:00:00Z', { name: 'Blank slate' })]);
    listChecklists.mockResolvedValue([]);

    render(<Home />);

    expect(await screen.findByText('Blank slate')).toBeInTheDocument();
    expect(await screen.findByText('0/0')).toBeInTheDocument();
  });

  it('should show what exists below three plans, with no empty placeholders', async () => {
    listPlans.mockResolvedValue([
      plan('a', '2026-09-01T00:00:00Z'),
      plan('b', '2026-08-01T00:00:00Z'),
    ]);

    render(<Home />);
    await screen.findByRole('heading', { name: /continue where you left off/i });

    await waitFor(() => expect(screen.getAllByRole('listitem')).toHaveLength(2));
  });

  it('should paint plan-derived content before the item data arrives', async () => {
    const items = deferred<ChecklistItem[]>();
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z', { name: 'Early paint' })]);
    listChecklists.mockReturnValue(items.promise);

    render(<Home />);

    // The card is on screen with no progress and no full-page spinner.
    expect(await screen.findByText('Early paint')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).toBeNull();
    expect(screen.queryByText(/loading/i)).toBeNull();
    expect(screen.getByTestId('continue-progress-a')).toHaveTextContent('');

    items.resolve([item('1', true)]);
    expect(await screen.findByText('1/1')).toBeInTheDocument();
  });

  it('should hold the progress slot open so the card does not change height as items land', async () => {
    const items = deferred<ChecklistItem[]>();
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);
    listChecklists.mockReturnValue(items.promise);

    render(<Home />);

    const slotWhileLoading = await screen.findByTestId('continue-progress-a');
    expect(slotWhileLoading).toBeInTheDocument();

    items.resolve([item('1', false)]);
    await screen.findByText('0/1');

    // Same element, filled in — not a node that appeared and pushed the row.
    expect(screen.getByTestId('continue-progress-a')).toBe(slotWhileLoading);
  });
});

describe('Dashboard — partial and total failure', () => {
  it('should render the other two cards when one item fetch rejects', async () => {
    listPlans.mockResolvedValue([
      plan('a', '2026-09-01T00:00:00Z'),
      plan('b', '2026-08-01T00:00:00Z'),
      plan('c', '2026-07-01T00:00:00Z'),
    ]);
    listChecklists.mockImplementation(async (id: string) => {
      if (id === 'b') throw new Error('nope');
      return [item('1', true)];
    });

    render(<Home />);

    expect(await screen.findByText('Plan a')).toBeInTheDocument();
    expect(screen.getByText('Plan b')).toBeInTheDocument();
    expect(screen.getByText('Plan c')).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getAllByText('1/1')).toHaveLength(2),
    );
  });

  it('should show no progress — not a wrong 0/0 — for the plan whose items failed', async () => {
    listPlans.mockResolvedValue([plan('b', '2026-09-01T00:00:00Z')]);
    listChecklists.mockRejectedValue(new Error('nope'));

    render(<Home />);

    await screen.findByText('Plan b');
    await waitFor(() =>
      expect(screen.getByTestId('continue-progress-b')).toHaveTextContent(''),
    );
    expect(screen.queryByText('0/0')).toBeNull();
  });

  it('should offer a retry rather than a blank page when the plans fetch fails', async () => {
    listPlans.mockRejectedValueOnce(new Error('down'));
    listPlans.mockResolvedValueOnce([plan('a', '2026-09-01T00:00:00Z', { name: 'Back again' })]);

    render(<Home />);

    const retry = await screen.findByRole('button', { name: /try again/i });
    expect(listChecklists).not.toHaveBeenCalled();

    await userEvent.click(retry);

    expect(await screen.findByText('Back again')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /try again/i })).toBeNull();
  });

  it('should keep the status line when the plans fetch fails', async () => {
    // §Edge States: home shows the status line's neutral form AND an error
    // state with a retry — the weekday is true whatever the network did, and
    // dropping it makes the page a bare error message.
    listPlans.mockRejectedValue(new Error('down'));

    render(<Home />);

    await screen.findByRole('button', { name: /try again/i });
    expect(screen.getByText(format(new Date(), 'EEEE'))).toBeInTheDocument();
  });

  it('should offer a retry in Today, and no count, when all three item fetches fail', async () => {
    listPlans.mockResolvedValue(plansAged(3));
    listChecklists.mockRejectedValue(new Error('nope'));

    render(<Home />);

    // Today owns the failure; the status line drops back to the weekday alone
    // rather than claiming a day it never read.
    expect(await screen.findByText(/couldn.t load today/i)).toBeInTheDocument();
    expect(screen.getByText(format(new Date(), 'EEEE'))).toBeInTheDocument();
    expect(screen.queryByText(/due across/)).toBeNull();
    expect(screen.queryByText(/nothing due/)).toBeNull();
    // The Continue cards still render, without progress.
    expect(screen.getByText('Plan p2')).toBeInTheDocument();
    expect(screen.getByTestId('continue-progress-p2')).toHaveTextContent('');
  });

  it('should re-run the fan-out from Today\'s retry', async () => {
    listPlans.mockResolvedValue(plansAged(1));
    listChecklists.mockRejectedValueOnce(new Error('nope'));
    listChecklists.mockResolvedValue([item('1', false)]);

    render(<Home />);

    await userEvent.click(await screen.findByRole('button', { name: /try again/i }));

    expect(await screen.findByText('0/1')).toBeInTheDocument();
    expect(screen.queryByText(/couldn.t load today/i)).toBeNull();
  });
});

describe('Dashboard — Today, against the wire format (I-0059)', () => {
  // A fixed Monday, so "2 due" and the weekday are facts of the fixture rather
  // than of the machine running the test.
  beforeEach(() => vi.setSystemTime(new Date(2026, 2, 16, 9, 0)));
  afterEach(() => vi.useRealTimers());

  it('should head the block Today and count the due items in the status line', async () => {
    // Before I-0059 these dates parsed to null and every account fell through
    // to Next up, so this is the first test where the due arm is live end to
    // end: selector, block heading and status line together.
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);
    listChecklists.mockResolvedValue([
      { ...item('due', false), dueDate: '2026-03-16T00:00:00Z' },
      { ...item('late', false), dueDate: '2026-03-14T00:00:00Z' },
      { ...item('later', false), dueDate: '2026-04-16T00:00:00Z' },
    ]);

    render(<Home />);

    expect(await screen.findByRole('heading', { name: 'Today' })).toBeInTheDocument();
    expect(screen.getByText('Monday · 2 due across 1 plan')).toBeInTheDocument();
    expect(screen.queryByText(/nothing due/)).toBeNull();
  });

  it('should still fall back to Next up when nothing is due', async () => {
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);
    listChecklists.mockResolvedValue([
      { ...item('later', false), dueDate: '2026-04-16T00:00:00Z' },
    ]);

    render(<Home />);

    expect(await screen.findByRole('heading', { name: 'Next up' })).toBeInTheDocument();
    expect(
      screen.getByText('Monday · nothing due — 1 waiting across 1 plan')
    ).toBeInTheDocument();
  });
});

describe('Dashboard — visual conformance (R18, R47–R50)', () => {
  it('should not wear the /myplans header card', async () => {
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);

    const { container } = render(<Home />);
    await screen.findByText('Plan a');

    expect(container.querySelector('.backdrop-blur-sm')).toBeNull();
    expect(container.querySelector('.rounded-2xl')).toBeNull();
    expect(container.querySelector('.shadow-lg')).toBeNull();
  });

  it('should carry muted lines on <p> with the muted token, not a dark: patch', async () => {
    // Possible only since I-0061 moved the global element rules into
    // `@layer base`; before that `.dark p` outranked `text-muted-foreground`
    // and these lines had to avoid <p> entirely. R50 still forbids a `dark:`
    // variant as the fix.
    listPlans.mockRejectedValue(new Error('down'));

    const { container } = render(<Home />);
    await screen.findByRole('button', { name: /try again/i });

    const muted = container.querySelectorAll('p[class*="text-muted-foreground"]');
    expect(muted.length).toBeGreaterThan(0);
    expect(container.innerHTML).not.toMatch(/dark:/);
  });

  it('should style from tokens only — no raw colour, no gray utility', async () => {
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);

    const { container } = render(<Home />);
    await screen.findByText('Plan a');

    const markup = container.innerHTML;
    expect(markup).not.toMatch(/rgba?\(/);
    expect(markup).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(markup).not.toMatch(/-gray-/);
  });

  it('should leave coral to slice 4 — Continue cards are not the page\'s accent', async () => {
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);

    const { container } = render(<Home />);
    await screen.findByText('Plan a');

    // R49: the one coral moment belongs to Today/Next up, which is not here yet.
    expect(container.querySelector('[class*="bg-primary"]')).toBeNull();
    expect(container.querySelector('[class*="text-primary"]')).toBeNull();
    // ...and no h1, so nothing picks up the global coral heading colour either.
    expect(container.querySelector('h1')).toBeNull();
  });
});

describe('Dashboard — first run (R43)', () => {
  it('should show one invitation instead of three empty blocks at zero plans', async () => {
    listPlans.mockResolvedValue([]);

    render(<Home />);

    expect(await screen.findByRole('heading', { name: 'Your first plan' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Today' })).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Next up' })).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Continue where you left off' })).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Your other plans' })).toBeNull();
  });

  it('should cost one request at zero plans, not four', async () => {
    listPlans.mockResolvedValue([]);

    render(<Home />);
    await screen.findByRole('heading', { name: 'Your first plan' });

    expect(listPlans).toHaveBeenCalledTimes(1);
    expect(listChecklists).not.toHaveBeenCalled();
  });

  it('should put the blocks back as soon as one plan exists', async () => {
    listPlans.mockResolvedValue([plan('a', '2026-09-01T00:00:00Z')]);
    listChecklists.mockResolvedValue([]);

    render(<Home />);

    expect(
      await screen.findByRole('heading', { name: 'Continue where you left off' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Your first plan' })).toBeNull();
  });

  it('should not render the invitation while the plans fetch is still in flight', () => {
    listPlans.mockReturnValue(new Promise(() => {}));

    render(<Home />);

    expect(screen.queryByRole('heading', { name: 'Your first plan' })).toBeNull();
  });
});
