import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, act } from '@testing-library/react';

/**
 * The sidebar discovery tip, and where it is allowed to fire (FS-KSJFR R46,
 * FS-S6WVY R10–R13).
 *
 * The tip tells the user their plans live in the side panel. On a plan page that
 * is news. On `/` it stopped being news when home grew Continue cards and Your
 * other plans — pointing at a panel that duplicates what is already on screen
 * reads as an unfinished redesign, so R46 takes the tip off that one route and
 * leaves it untouched everywhere else.
 *
 * It is also not news when the panel is already open (FS-S6WVY R12), so it only
 * fires when the remembered state is hidden on arrival.
 *
 * Home no longer forces the sidebar closed (FS-KSJFR R45, superseded by
 * FS-S6WVY R10): that collapse overwrote the user's remembered choice.
 */

const toast = vi.fn();
let pathname = '/';
let isAuthenticated = true;

vi.mock('@/components/ui/use-toast', () => ({ toast: (...args: unknown[]) => toast(...args) }));
vi.mock('next/navigation', () => ({ usePathname: () => pathname }));
vi.mock('@/context/AuthContext', () => ({ useAuth: () => ({ isAuthenticated }) }));
vi.mock('./Sidebar', () => ({ default: () => <div data-testid="sidebar" /> }));
vi.mock('./UserProfile', () => ({ default: () => null }));
vi.mock('./Logo', () => ({ default: () => null }));
vi.mock('./ThemeToggle', () => ({ default: () => null }));
vi.mock('./AuthGuard', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import LayoutContent from './LayoutContent';

/**
 * The remembered choice, as the head script would have painted it on arrival.
 * The toast reads this, never the sidebar context.
 */
const arriveWithSidebar = (state: 'open' | 'hidden') => {
  localStorage.setItem('sidebar', state);
  document.documentElement.setAttribute('data-sidebar', state);
};

/** The first-timer hint is behind a 1.5s timer; let it come due. */
const letTheHintFire = () =>
  act(() => {
    vi.advanceTimersByTime(2000);
  });

beforeEach(() => {
  vi.useFakeTimers();
  toast.mockReset();
  localStorage.clear();
  document.documentElement.removeAttribute('data-sidebar');
  isAuthenticated = true;
  arriveWithSidebar('hidden');
});
afterEach(() => vi.useRealTimers());

describe('LayoutContent — the sidebar discovery tip (R46)', () => {
  it('should not fire on the authenticated home, even for a first-timer', () => {
    pathname = '/';

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
  });

  it('should still fire on a plan page for a first-timer', () => {
    pathname = '/plan/abc';

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).toHaveBeenCalledTimes(1);
    expect(toast.mock.calls[0][0]).toMatchObject({
      title: expect.stringMatching(/side panel/i),
    });
  });

  it('should still fire the 24h reminder on a plan page', () => {
    pathname = '/plan/abc';
    localStorage.setItem('hasSeenSidebarHint', 'true');

    render(<LayoutContent>page</LayoutContent>);

    // No timer on this branch — it toasts during the effect.
    expect(toast).toHaveBeenCalledTimes(1);
  });

  it('should not fire on home even once the first-timer hint is spent', () => {
    pathname = '/';
    localStorage.setItem('hasSeenSidebarHint', 'true');

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
  });

  it('should leave the home first-timer flag unspent, so a plan page still gets it', () => {
    pathname = '/';
    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    // Home must not quietly burn the one-time hint on the user's behalf.
    expect(localStorage.getItem('hasSeenSidebarHint')).toBeNull();
  });

  it('should not fire anywhere when signed out', () => {
    pathname = '/plan/abc';
    isAuthenticated = false;

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
  });
});

describe('LayoutContent — the tip only when the sidebar is hidden (FS-S6WVY R12–R13)', () => {
  it('should not fire on a plan page when the sidebar is open, and leave the first-timer flag unspent', () => {
    pathname = '/plan/abc';
    arriveWithSidebar('open');

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
    expect(localStorage.getItem('hasSeenSidebarHint')).toBeNull();
  });

  it('should not fire the 24h reminder on a plan page when the sidebar is open', () => {
    pathname = '/plan/abc';
    arriveWithSidebar('open');
    localStorage.setItem('hasSeenSidebarHint', 'true');

    render(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
  });

  it('should not fire as a reaction to hiding the sidebar on a plan page', () => {
    pathname = '/plan/abc';
    arriveWithSidebar('open');
    const { rerender } = render(<LayoutContent>page</LayoutContent>);

    arriveWithSidebar('hidden');
    rerender(<LayoutContent>page</LayoutContent>);
    letTheHintFire();

    expect(toast).not.toHaveBeenCalled();
  });
});

describe('LayoutContent — blocked storage (FS-S6WVY R8)', () => {
  it.each(['getItem', 'setItem'] as const)(
    'should not throw on a plan page when storage %s throws',
    (method) => {
      pathname = '/plan/abc';
      arriveWithSidebar('hidden');
      vi.spyOn(Storage.prototype, method).mockImplementation(() => {
        throw new Error('SecurityError');
      });

      expect(() => {
        render(<LayoutContent>page</LayoutContent>);
        letTheHintFire();
      }).not.toThrow();
    },
  );
});

describe('LayoutContent — home respects the remembered choice (FS-S6WVY R10)', () => {
  it.each(['open', 'hidden'] as const)(
    'should neither collapse nor write on home with the sidebar %s',
    (state) => {
      pathname = '/';
      arriveWithSidebar(state);
      const setItem = vi.spyOn(Storage.prototype, 'setItem');
      const removeItem = vi.spyOn(Storage.prototype, 'removeItem');

      render(<LayoutContent>page</LayoutContent>);
      letTheHintFire();

      expect(setItem).not.toHaveBeenCalledWith('sidebar', expect.anything());
      expect(removeItem).not.toHaveBeenCalled();
    },
  );

  it('should never write the choice while navigating between home, a plan and My Plans', () => {
    arriveWithSidebar('open');
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    pathname = '/';
    const { rerender } = render(<LayoutContent>page</LayoutContent>);

    for (const next of ['/plan/abc', '/myplans', '/', '/plan/xyz', '/']) {
      pathname = next;
      rerender(<LayoutContent>page</LayoutContent>);
      letTheHintFire();
    }

    expect(setItem).not.toHaveBeenCalledWith('sidebar', expect.anything());
    expect(localStorage.getItem('sidebar')).toBe('open');
  });

  it('should give home the same content offset as every other route', () => {
    arriveWithSidebar('open');
    pathname = '/';
    const { container: home } = render(<LayoutContent>page</LayoutContent>);
    const homeOffset = home.querySelector('main')?.parentElement?.className;

    pathname = '/plan/abc';
    const { container: plan } = render(<LayoutContent>page</LayoutContent>);
    const planOffset = plan.querySelector('main')?.parentElement?.className;

    expect(homeOffset).toBe(planOffset);
  });
});
