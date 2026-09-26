import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useLayoutEffect } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';

const auth = { user: null as { name: string } | null, isAuthenticated: false, isLoading: true };
const push = vi.fn();
// One plan is enough for the dashboard to have something to render; the
// fan-out's own behaviour is covered in Dashboard.test.tsx.
const listPlans = vi.fn(async () => [
  {
    id: 'p1',
    name: 'Ship the thing',
    focus: 'get it out',
    description: '',
    planType: 'project',
    dailyReset: false,
    userId: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-09-01T00:00:00Z',
  },
]);
const listChecklists = vi.fn(async () => []);

// The two day-stamps the gate reads, by the names the storage holds them under —
// written here by hand rather than through the helpers, so a test that says
// "the gate already fired today" is arranging storage and not re-running the
// code under test. Day stamps are local calendar days (FS-KSJFR D3).
const GATE_KEY = 'gateFiredToday';
const TOUCHED_KEY = 'touchedToday';
const todayStamp = () => {
  const now = new Date();
  const month = `${now.getMonth() + 1}`.padStart(2, '0');
  const day = `${now.getDate()}`.padStart(2, '0');
  return `${now.getFullYear()}-${month}-${day}`;
};
const stampToday = (key: string) => window.localStorage.setItem(key, todayStamp());

const promptShowing = () => screen.queryByText(/what's your focus today/i);

vi.mock('@/context/AuthContext', () => ({ useAuth: () => auth }));
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }));
// The chrome bar reuses the app's ThemeToggle, so the tour needs the theme
// context. Everything else in the landing renders provider-free.
vi.mock('@/context/ThemeContext', () => ({
  useTheme: () => ({ theme: 'dark', toggleTheme: vi.fn() }),
}));
vi.mock('@/api/plans', () => ({ listPlans: () => listPlans() }));
vi.mock('@/api/checklists', () => ({ listChecklists: () => listChecklists() }));

import Home from './page';

describe('Home', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
    push.mockClear();
    listPlans.mockClear();
    listChecklists.mockClear();
    auth.user = null;
    auth.isAuthenticated = false;
    auth.isLoading = true;
  });

  it('should paint the hero while auth is still resolving', () => {
    render(<Home />);

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fireplace');
    expect(screen.queryByText(/loading/i)).toBeNull();
  });

  it('should show the focus prompt once the visitor is known to be authenticated', () => {
    auth.isAuthenticated = false;
    auth.isLoading = false;
    const { rerender } = render(<Home />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fireplace');

    auth.isAuthenticated = true;
    auth.user = { name: 'Kranti' };
    rerender(<Home />);

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(/welcome back/i);
    expect(screen.getByText(/what's your focus today/i)).toBeInTheDocument();
  });

  it('should not fetch the dashboard while the focus prompt is the state on screen', () => {
    auth.isAuthenticated = true;
    auth.isLoading = false;

    render(<Home />);

    expect(listPlans).not.toHaveBeenCalled();
  });

  describe('the day gate (R3-R6, R10, R15)', () => {
    beforeEach(() => {
      auth.isAuthenticated = true;
      auth.isLoading = false;
      auth.user = { name: 'Kranti' };
    });

    // The whole rule in four rows: the prompt opens `/` only when the gate has
    // not had its turn today AND nothing was touched today (R3). Anything else
    // is the dashboard. "The dashboard is showing" is asserted as the prompt
    // being absent plus the fan-out having started, which is true of the
    // dashboard in every one of its own states — including the failed fetch.
    it.each([
      ['the prompt when neither stamp is set', false, false, 'prompt'],
      ['the dashboard when the gate already fired today', true, false, 'dashboard'],
      ['the dashboard when something was touched today', false, true, 'dashboard'],
      ['the dashboard when both stamps are set', true, true, 'dashboard'],
    ] as const)('should open on %s', async (_case, fired, touched, expected) => {
      if (fired) stampToday(GATE_KEY);
      if (touched) stampToday(TOUCHED_KEY);

      render(<Home />);

      if (expected === 'prompt') {
        expect(promptShowing()).toBeInTheDocument();
        expect(listPlans).not.toHaveBeenCalled();
      } else {
        expect(promptShowing()).toBeNull();
        await waitFor(() => expect(listChecklists).toHaveBeenCalled());
      }
    });

    /**
     * R4, and the reason this test exists rather than trusting the four rows
     * above: a `useEffect` that corrects the state after the first commit
     * passes every assertion about the final DOM while shipping exactly the
     * flash the spec forbids. So this asserts the *phase* the decision is made
     * in. React renders the whole tree before it runs a single effect, and the
     * probe sits first in tree order, so its layout effect precedes any effect
     * of Home's. If the gate's storage read lands after that marker, the
     * decision was made post-paint.
     */
    it('should decide during render, before React runs any effect', async () => {
      stampToday(GATE_KEY);
      const order: string[] = [];
      const read = Storage.prototype.getItem;
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (
        this: Storage,
        key: string,
      ) {
        order.push(`read:${key}`);
        return read.call(this, key);
      });

      function EffectProbe() {
        useLayoutEffect(() => {
          order.push('first-effect');
        }, []);
        return null;
      }

      render(
        <>
          <EffectProbe />
          <Home />
        </>,
      );

      expect(order).toContain('first-effect');
      // Pin the PHASE, not which key is read first. Asserting the operand order
      // means a behaviour-preserving refactor (swapping the conjunction in
      // `dayGateFires`) fails here with a misleading message, while the thing
      // this test exists to catch — the read moving into an effect — is caught
      // either way.
      expect(order[0]).toMatch(/^read:/);
      expect(order.indexOf('first-effect')).toBeGreaterThan(0);
      // The dashboard this rendered has a fan-out in flight; let it land so its
      // state lands inside the test rather than after it.
      await waitFor(() => expect(listChecklists).toHaveBeenCalled());
    });

    it('should record the fire on mount, before the user does anything with the prompt', () => {
      render(<Home />);

      expect(promptShowing()).toBeInTheDocument();
      expect(window.localStorage.getItem(GATE_KEY)).toBe(todayStamp());
    });

    it('should send a second visit the same day straight to the dashboard (R5)', async () => {
      // First visit: the gate fires and the visitor closes the tab on it
      // without skipping and without acting.
      const first = render(<Home />);
      expect(promptShowing()).toBeInTheDocument();
      first.unmount();

      render(<Home />);

      expect(promptShowing()).toBeNull();
      await waitFor(() => expect(listChecklists).toHaveBeenCalled());
    });

    it('should not re-gate when the tab regains focus or becomes visible (R6)', () => {
      render(<Home />);
      expect(promptShowing()).toBeInTheDocument();

      // Midnight passes and another tab records a touch. An open tab keeps the
      // state it has until it reloads; that is the accepted limit, not a bug.
      stampToday(TOUCHED_KEY);
      fireEvent(window, new Event('focus'));
      fireEvent(document, new Event('visibilitychange'));

      expect(promptShowing()).toBeInTheDocument();
      expect(listPlans).not.toHaveBeenCalled();
    });

    it.each([
      ['empty', ''],
      ['yesterday', '2020-01-01'],
      ['garbage', 'yes'],
      ['a full ISO timestamp for today', new Date().toISOString()],
    ])('should show the prompt when the gate stamp reads %s (R15)', (_case, raw) => {
      window.localStorage.setItem(GATE_KEY, raw);

      render(<Home />);

      expect(promptShowing()).toBeInTheDocument();
    });

    it('should show the prompt, not an error, when storage throws on read (R15)', () => {
      // Private mode / blocked site data: the accessor itself throws.
      vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => {
        throw new DOMException('blocked', 'SecurityError');
      });

      expect(() => render(<Home />)).not.toThrow();
      expect(promptShowing()).toBeInTheDocument();
    });

    // R7: the gate decides *whether* the prompt opens and changes nothing about
    // what it does when it does. Asserted here, through Home, because that is
    // the surface the gate put a condition in front of.
    it('should leave the funnel to /create-plan intact when the gate fires (R7)', async () => {
      render(<Home />);

      await userEvent.click(screen.getByRole('button', { name: /project/i }));
      await userEvent.type(
        screen.getByPlaceholderText(/building a movie app/i),
        'ship the gate',
      );
      await userEvent.click(screen.getByRole('button', { name: /start this plan/i }));

      expect(push).toHaveBeenCalledTimes(1);
      const target = new URL(push.mock.calls[0][0] as string, 'http://localhost');
      expect(target.pathname).toBe('/create-plan');
      expect(target.searchParams.get('name')).toBe('ship the gate');
      expect(target.searchParams.get('focus')).toBe('ship the gate');
      expect(target.searchParams.get('planType')).toBe('project');
    });

    it('should leave the stamps alone for a logged-out visitor (R1)', () => {
      auth.isAuthenticated = false;
      auth.user = null;

      render(<Home />);

      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Fireplace');
      expect(window.localStorage.getItem(GATE_KEY)).toBeNull();
    });
  });

  describe('skipping the prompt (R9)', () => {
    beforeEach(() => {
      auth.isAuthenticated = true;
      auth.isLoading = false;
      auth.user = { name: 'Kranti' };
    });

    it('should reveal the dashboard in place, with no navigation', async () => {
      render(<Home />);

      await userEvent.click(screen.getByRole('button', { name: /skip/i }));

      expect(await screen.findByRole('heading', { name: /continue where you left off/i }))
        .toBeInTheDocument();
      expect(screen.queryByText(/what's your focus today/i)).toBeNull();
      // `/` is one route in two states: nothing was pushed and no URL moved.
      expect(push).not.toHaveBeenCalled();
    });

    it('should record the gate as fired but not as a touch (R10)', async () => {
      render(<Home />);

      await userEvent.click(screen.getByRole('button', { name: /skip/i }));

      expect(window.localStorage.getItem(GATE_KEY)).toBe(todayStamp());
      // Skipping is not activity: tomorrow's gate still gets its turn, and
      // nothing else in the app may read this visit as work having happened.
      expect(window.localStorage.getItem(TOUCHED_KEY)).toBeNull();
    });

    it('should no longer send the user to the plans list', () => {
      render(<Home />);

      const skip = screen.getByRole('button', { name: /skip/i });
      expect(skip.textContent).not.toMatch(/plans/i);
      expect(
        screen.queryByRole('link', { name: /skip/i }),
      ).toBeNull();
      expect(
        Array.from(document.querySelectorAll('a')).map((a) => a.getAttribute('href')),
      ).not.toContain('/myplans');
    });
  });

  it('should style the focus prompt from tokens only', () => {
    auth.isAuthenticated = true;
    auth.isLoading = false;

    const { container } = render(<Home />);

    // Assert the prompt is what is actually on screen. Without this the four
    // checks below would happily pass against the dashboard's markup if the
    // gate's default ever flipped for this arrangement, and the test would go
    // on claiming to cover the prompt.
    expect(promptShowing()).toBeInTheDocument();

    const markup = container.innerHTML;
    expect(markup).not.toMatch(/rgba?\(/);
    expect(markup).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(markup).not.toMatch(/-gray-/);
    // The header card that /myplans and /plan/[planId] keep does not come home.
    expect(container.querySelector('.backdrop-blur-sm')).toBeNull();
  });

  describe('the first-run invitation (R43-R44)', () => {
    it('should put the prompt back in place, with no navigation', async () => {
      auth.isAuthenticated = true;
      auth.isLoading = false;
      listPlans.mockResolvedValueOnce([]);
      const user = userEvent.setup();

      render(<Home />);

      // Gate fires on a clean slate, so the prompt opens the day.
      expect(promptShowing()).toBeInTheDocument();

      await user.click(screen.getByText(/skip for now/i));
      expect(
        await screen.findByRole('heading', { name: 'Your first plan' }),
      ).toBeInTheDocument();
      expect(promptShowing()).toBeNull();

      await user.click(screen.getByRole('button', { name: /start a plan/i }));

      // Back to the prompt, on the same route — the invitation hands off, it
      // does not send the user somewhere.
      expect(promptShowing()).toBeInTheDocument();
      expect(push).not.toHaveBeenCalled();
      expect(screen.queryByRole('heading', { name: 'Your first plan' })).toBeNull();
    });

    it('should not spend the day a second time when the prompt is reopened', async () => {
      auth.isAuthenticated = true;
      auth.isLoading = false;
      listPlans.mockResolvedValueOnce([]);
      const user = userEvent.setup();

      render(<Home />);
      await user.click(screen.getByText(/skip for now/i));
      await screen.findByRole('heading', { name: 'Your first plan' });
      await user.click(screen.getByRole('button', { name: /start a plan/i }));

      // The gate's turn was spent on mount; reopening is not a second firing.
      expect(window.localStorage.getItem(GATE_KEY)).toBe(todayStamp());
      expect(window.localStorage.getItem(TOUCHED_KEY)).toBeNull();
    });
  });
});
