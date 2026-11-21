import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { mockMatchMedia } from '@/test/motion';

/**
 * The top bar at phone widths (FS-49CQ2 R16–R20): what stays reachable and
 * named once the title and the profile name go visual-only below `sm`. The
 * layout itself (tap targets, gutters) is checked in a browser (R22).
 */

let pathname = '/plan/abc';
const LONG_NAME = 'Bartholomew Montgomery-Fitzwilliam the Third';

vi.mock('next/navigation', () => ({ usePathname: () => pathname }));
vi.mock('@/context/AuthContext', () => ({
  useAuth: () => ({ isAuthenticated: true, signOut: vi.fn() }),
}));
vi.mock('@/context/ThemeContext', () => ({
  useTheme: () => ({ theme: 'light', toggleTheme: vi.fn() }),
}));
vi.mock('@/services/api', () => ({
  getProfile: () => Promise.resolve({ displayName: LONG_NAME, name: 'bart' }),
}));
vi.mock('@/components/ui/use-toast', () => ({ toast: vi.fn() }));
vi.mock('./Sidebar', () => ({ default: () => null }));
vi.mock('./Logo', () => ({ default: () => <svg data-testid="logo" /> }));
vi.mock('./AuthGuard', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import LayoutContent from './LayoutContent';

const topBar = () => screen.getByRole('banner');

/** Render the shell and let the profile's name arrive, so nothing updates after the test. */
async function shell() {
  render(<LayoutContent>page</LayoutContent>);
  await screen.findByText(LONG_NAME);
}

beforeEach(() => {
  mockMatchMedia({ wide: false });
  pathname = '/plan/abc';
});

describe('top bar — the logo (R16)', () => {
  it('should keep the logo linking home, with an accessible name', async () => {
    await shell();

    const home = within(topBar()).getByRole('link', { name: 'Fireplace' });
    expect(home.getAttribute('href')).toBe('/');
    expect(within(home).getByTestId('logo')).toBeTruthy();
  });
});

describe('top bar — the profile control (R18)', () => {
  it('should keep the name as the profile control\'s accessible name', async () => {
    await shell();

    const profile = within(topBar()).getByRole('button', { name: LONG_NAME });
    expect(within(profile).getByText(LONG_NAME)).toBeTruthy();
  });
});

describe('top bar — stacking', () => {
  it('should stack above page cards that lift themselves to z-10, so its menu is never covered', async () => {
    await shell();

    // The plan title card is `relative z-10`; a tie lets it, later in the DOM,
    // paint over the bar on scroll and over the profile menu's "Sign Out".
    expect(topBar().className.split(/\s+/)).toContain('z-30');
  });
});
