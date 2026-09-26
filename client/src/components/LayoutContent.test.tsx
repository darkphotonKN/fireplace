import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, act } from '@testing-library/react';

/**
 * The sidebar discovery tip, and where it is allowed to fire (FS-KSJFR R45–R46).
 *
 * The tip tells the user their plans live in the side panel. On a plan page that
 * is news. On `/` it stopped being news when home grew Continue cards and Your
 * other plans — pointing at a panel that duplicates what is already on screen
 * reads as an unfinished redesign, so R46 takes the tip off that one route and
 * leaves it untouched everywhere else.
 *
 * The collapse (R45) is a separate effect and deliberately survives: home keeps
 * its clean full-width entry.
 */

const toast = vi.fn();
const setIsCollapsed = vi.fn();
let pathname = '/';
let isAuthenticated = true;

vi.mock('@/components/ui/use-toast', () => ({ toast: (...args: unknown[]) => toast(...args) }));
vi.mock('next/navigation', () => ({ usePathname: () => pathname }));
vi.mock('@/context/AuthContext', () => ({ useAuth: () => ({ isAuthenticated }) }));
vi.mock('./LayoutWrapper', () => ({
  useSidebar: () => ({ isCollapsed: true, setIsCollapsed }),
}));
vi.mock('./Sidebar', () => ({ default: () => <div data-testid="sidebar" /> }));
vi.mock('./UserProfile', () => ({ default: () => null }));
vi.mock('./Logo', () => ({ default: () => null }));
vi.mock('./ThemeToggle', () => ({ default: () => null }));
vi.mock('./AuthGuard', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import LayoutContent from './LayoutContent';

/** The first-timer hint is behind a 1.5s timer; let it come due. */
const letTheHintFire = () =>
  act(() => {
    vi.advanceTimersByTime(2000);
  });

beforeEach(() => {
  vi.useFakeTimers();
  toast.mockReset();
  setIsCollapsed.mockReset();
  localStorage.clear();
  isAuthenticated = true;
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

describe('LayoutContent — the sidebar collapse on home (R45)', () => {
  it('should still collapse the sidebar on entry to home', () => {
    pathname = '/';

    render(<LayoutContent>page</LayoutContent>);

    expect(setIsCollapsed).toHaveBeenCalledWith(true);
  });

  it('should not force the collapse on a plan page', () => {
    pathname = '/plan/abc';

    render(<LayoutContent>page</LayoutContent>);

    expect(setIsCollapsed).not.toHaveBeenCalled();
  });
});
