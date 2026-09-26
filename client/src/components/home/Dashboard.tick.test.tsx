import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { wasTouchedToday } from '@/lib/touchedToday';

/**
 * FS-KSJFR slice 5 (I-KSJFR-5): ticking a row off from home (R33–R35, D9,
 * §Edge States).
 *
 * **Why this file mocks the HTTP client and not the API layer.** Every other
 * Dashboard test stubs `@/api/checklists`. This one must not: the touched-today
 * stamp (I-KSJFR-1, R12) lives *inside* `updateChecklistItem`, and a stubbed
 * module would let the tick pass while the stamp quietly stopped firing —
 * exactly the regression that would send the day gate off again after the user
 * had already acted from home. So `@/api/client` is the seam, and everything
 * above it is the real thing: the API layer, the selector, the overlay.
 *
 * The other load-bearing assertions are about what does NOT happen: the row
 * does not move, the heading does not change, and nothing is re-fetched.
 */

const api = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
  PATCH: vi.fn(),
  DELETE: vi.fn(),
}));
const toast = vi.hoisted(() => vi.fn());

vi.mock('@/api/client', () => ({
  api,
  apiErrorFrom: (_error: unknown, status: number) =>
    new Error(`request failed with ${status}`),
  ApiError: class ApiError extends Error {},
}));
vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast }),
}));

import Dashboard from './Dashboard';

/**
 * `onStartPlan` belongs to the first-run invitation (I-KSJFR-7) — it puts the
 * focus prompt back on screen. Nothing in these files exercises that path, so
 * they take a noop and the prop stays required on the real component.
 */
const Home = () => <Dashboard onStartPlan={() => {}} />;

const TODAY = new Date(2026, 2, 16, 9, 0); // Monday 16 March 2026, local
const DUE = '2026-03-16T00:00:00Z';

const plan = {
  id: 'p1',
  name: 'Ship it',
  focus: '',
  description: '',
  planType: 'project',
  dailyReset: false,
  userId: 'u1',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-03-01T00:00:00Z',
};

const item = (id: string) => ({
  id,
  planId: 'p1',
  description: id,
  done: false,
  archived: false,
  scope: 'longterm',
  type: 'task',
  sequence: '1',
  dueDate: DUE,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
});

const ITEMS = ['first', 'second', 'third'].map(item);

const ok = (data: unknown) => ({
  data,
  error: undefined,
  response: { status: 200 },
});

/** A promise the test settles by hand, so "in flight" is observable. */
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

const rejected = {
  data: undefined,
  error: { code: 'INTERNAL' },
  response: { status: 500 },
};

/** Let everything already scheduled run, so an over-fetch gets its chance. */
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

/** The Today block only: a Continue card is a list item too. */
const block = () => screen.getByRole('region', { name: 'Today' });
const tickBox = (name: string) => screen.getByRole('checkbox', { name });
const rowOrder = () =>
  within(block())
    .getAllByRole('listitem')
    .map((li) => li.textContent);
const ticked = () =>
  within(block())
    .getAllByRole('checkbox')
    .map((box) => box.getAttribute('aria-checked'));

beforeEach(() => {
  vi.setSystemTime(TODAY);
  toast.mockReset();
  api.GET.mockReset();
  api.PATCH.mockReset();
  api.GET.mockImplementation((path: string) =>
    Promise.resolve(path === '/api/plans' ? ok([plan]) : ok(ITEMS)),
  );
  api.PATCH.mockResolvedValue(ok({ ...ITEMS[0], done: true }));
});
afterEach(() => vi.useRealTimers());

/** Renders home and waits for the three due rows to land. */
async function openHome() {
  const user = userEvent.setup();
  render(<Home />);
  await screen.findByRole('heading', { name: 'Today' });
  await waitFor(() => expect(ticked()).toHaveLength(3));
  return user;
}

describe('Dashboard — ticking a row off home (R33)', () => {
  it('should mark the row done before the write resolves', async () => {
    const write = deferred<unknown>();
    api.PATCH.mockReturnValue(write.promise);
    const user = await openHome();

    await user.click(tickBox('second'));

    // Nothing has come back yet, and the row already reads done.
    expect(ticked()).toEqual(['false', 'true', 'false']);
    expect(screen.getByText(/2 due across 1 plan/)).toBeInTheDocument();

    await act(async () => {
      write.resolve(ok({ ...ITEMS[1], done: true }));
    });
    expect(ticked()).toEqual(['false', 'true', 'false']);
  });

  it('should write through the API layer, against the row’s own plan', async () => {
    const user = await openHome();

    await user.click(tickBox('second'));
    await settle();

    expect(api.PATCH).toHaveBeenCalledTimes(1);
    expect(api.PATCH).toHaveBeenCalledWith(
      '/api/plans/{id}/checklists/{checklist_id}',
      {
        params: { path: { id: 'p1', checklist_id: 'second' } },
        body: { done: true },
      },
    );
  });

  it('should record touched-today, so the day gate does not fire again', async () => {
    // R12 via I-KSJFR-1. The stamp is written inside `updateChecklistItem`, and
    // this assertion is what keeps the tick going through it.
    expect(wasTouchedToday()).toBe(false);
    const user = await openHome();

    await user.click(tickBox('second'));
    await waitFor(() => expect(wasTouchedToday()).toBe(true));
  });

  it('should leave the ticked row exactly where it was', async () => {
    const user = await openHome();

    await user.click(tickBox('second'));
    await settle();

    expect(rowOrder()).toEqual([
      expect.stringContaining('first'),
      expect.stringContaining('second'),
      expect.stringContaining('third'),
    ]);
  });

  it('should re-fetch nothing', async () => {
    const user = await openHome();
    const reads = api.GET.mock.calls.length;

    await user.click(tickBox('second'));
    await settle();

    expect(api.GET.mock.calls.length).toBe(reads);
  });

  it('should keep the sentence and the list agreeing about the count', async () => {
    // One `selectHomeWork` feeds both, and the overlay lands on that one object
    // rather than on two pieces of component state (R16).
    const user = await openHome();
    expect(screen.getByText(/3 due across 1 plan/)).toBeInTheDocument();

    await user.click(tickBox('first'));
    await settle();
    expect(screen.getByText(/2 due across 1 plan/)).toBeInTheDocument();

    await user.click(tickBox('third'));
    await settle();
    expect(screen.getByText(/1 due across 1 plan/)).toBeInTheDocument();
  });
});

describe('Dashboard — a tick that fails (R34, §Edge States)', () => {
  it('should put the row back, restore the count and say so', async () => {
    api.PATCH.mockResolvedValue(rejected);
    const user = await openHome();

    await user.click(tickBox('second'));
    await waitFor(() => expect(ticked()).toEqual(['false', 'false', 'false']));

    expect(screen.getByText(/3 due across 1 plan/)).toBeInTheDocument();
    expect(toast).toHaveBeenCalledTimes(1);
    expect(toast.mock.calls[0][0]).toMatchObject({ position: 'bottom-left' });
    expect(rowOrder()).toEqual([
      expect.stringContaining('first'),
      expect.stringContaining('second'),
      expect.stringContaining('third'),
    ]);
  });

  it('should stamp no day for a write that was refused (R14)', async () => {
    // The stamp is "you mutated something", so a rejected call must not leave
    // one — and the only reason this holds is that the tick goes through the API
    // layer, where the stamp sits after the throw.
    api.PATCH.mockResolvedValue(rejected);
    const user = await openHome();

    await user.click(tickBox('second'));
    await waitFor(() => expect(toast).toHaveBeenCalledTimes(1));
    expect(wasTouchedToday()).toBe(false);
  });

  it('should put the row back when the call never reaches a server', async () => {
    api.PATCH.mockRejectedValue(new Error('offline'));
    const user = await openHome();

    await user.click(tickBox('second'));
    await waitFor(() => expect(toast).toHaveBeenCalledTimes(1));
    expect(ticked()).toEqual(['false', 'false', 'false']);
    expect(screen.getByText(/3 due across 1 plan/)).toBeInTheDocument();
  });

  it('should leave the row tickable again after the restore', async () => {
    api.PATCH.mockResolvedValueOnce(rejected);
    const user = await openHome();

    await user.click(tickBox('second'));
    await waitFor(() => expect(toast).toHaveBeenCalledTimes(1));

    await user.click(tickBox('second'));
    await settle();
    expect(ticked()).toEqual(['false', 'true', 'false']);
    expect(screen.getByText(/2 due across 1 plan/)).toBeInTheDocument();
  });

  it('should restore only the failing row when two ticks are in flight', async () => {
    // §Edge States: two ticks in flight. Each resolves against its own row and
    // its own share of the count.
    const first = deferred<unknown>();
    const third = deferred<unknown>();
    api.PATCH.mockImplementation((_path: string, opts: any) =>
      opts.params.path.checklist_id === 'first' ? first.promise : third.promise,
    );
    const user = await openHome();

    await user.click(tickBox('first'));
    await user.click(tickBox('third'));
    expect(ticked()).toEqual(['true', 'false', 'true']);
    expect(screen.getByText(/1 due across 1 plan/)).toBeInTheDocument();

    // The later one lands, the earlier one is refused.
    await act(async () => {
      third.resolve(ok({ ...ITEMS[2], done: true }));
      first.resolve(rejected);
    });

    await waitFor(() => expect(ticked()).toEqual(['false', 'false', 'true']));
    expect(screen.getByText(/2 due across 1 plan/)).toBeInTheDocument();
    expect(toast).toHaveBeenCalledTimes(1);
  });
});

describe('Dashboard — every row ticked (R35)', () => {
  /**
   * A second plan with unticked, undated work is what makes this test bite.
   * The hazard D9 names is pulling in replacement rows *from another plan*, and
   * with a single-plan fixture there is no candidate to pull — a re-derivation
   * would have nothing to show and the test would pass for the wrong reason.
   * `waiting` is the row a re-selection would reach for.
   */
  const second = { ...plan, id: 'p2', name: 'Later', updatedAt: '2026-02-01T00:00:00Z' };
  const waiting = { ...item('a-replacement-row'), planId: 'p2', dueDate: undefined };

  beforeEach(() => {
    api.GET.mockImplementation((path: string, opts?: { params?: { path?: { id?: string } } }) => {
      if (path === '/api/plans') return Promise.resolve(ok([plan, second]));
      return Promise.resolve(ok(opts?.params?.path?.id === 'p2' ? [waiting] : ITEMS));
    });
  });

  it('should hold the heading and pull in no replacement rows', async () => {
    const user = await openHome();

    for (const name of ['first', 'second', 'third']) {
      await user.click(tickBox(name));
    }
    await settle();

    expect(screen.getByRole('heading', { name: 'Today' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Next up' })).toBeNull();
    expect(ticked()).toEqual(['true', 'true', 'true']);
    // The other plan's waiting row was available the whole time and stayed out.
    expect(within(block()).queryByText('a-replacement-row')).toBeNull();
    // The status line stops counting rather than switching to "nothing due —
    // N waiting", which would be a re-derivation the spec forbids mid-visit.
    expect(screen.getByText(/nothing due across 2 plans/)).toBeInTheDocument();
  });
});
