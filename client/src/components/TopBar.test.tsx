import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { mockMatchMedia } from '@/test/motion';

/**
 * The top bar at phone widths (FS-49CQ2 R16–R20). Title plus four controls
 * don't fit 375px, so below `sm` the title text and the profile name go
 * visual-only-hidden (still named for assistive tech), and below `md` every
 * control is a 40px tap target with a left gutter matching the right.
 *
 * jsdom does no layout, so widths are asserted through the responsive classes
 * that produce them; the 375px fit itself is checked in a browser (R22).
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
const classesOf = (el: Element) => el.getAttribute('class')?.split(/\s+/) ?? [];

beforeEach(() => {
  mockMatchMedia({ wide: false });
  pathname = '/plan/abc';
});

describe('top bar — the title below sm (R16)', () => {
  it('should hide the "Fireplace" text visually below sm and show it from sm', async () => {
    await shell();

    const title = within(topBar()).getByRole('heading', { name: 'Fireplace' });
    expect(classesOf(title)).toEqual(expect.arrayContaining(['sr-only', 'sm:not-sr-only']));
  });

  it('should keep the logo linking home, with an accessible name', async () => {
    await shell();

    const home = within(topBar()).getByRole('link', { name: 'Fireplace' });
    expect(home.getAttribute('href')).toBe('/');
    expect(within(home).getByTestId('logo')).toBeTruthy();
  });
});

describe('top bar — the profile control below sm (R18)', () => {
  it('should hide the name text below sm and keep it as the accessible name', async () => {
    await shell();

    const profile = within(topBar()).getByRole('button', { name: LONG_NAME });
    const name = within(profile).getByText(LONG_NAME);
    expect(classesOf(name)).toEqual(expect.arrayContaining(['sr-only', 'sm:not-sr-only']));
  });

  it('should cut a long name short from sm rather than squeeze the other controls', async () => {
    await shell();

    expect(classesOf(screen.getByText(LONG_NAME))).toContain('sm:truncate');
    for (const fixed of [
      screen.getByRole('link', { name: 'Home' }),
      screen.getByRole('button', { name: /switch to dark mode/i }),
    ]) {
      expect(classesOf(fixed)).toContain('shrink-0');
    }
  });
});

describe('top bar — 40px tap targets below md (R17)', () => {
  it.each([
    ['menu', () => screen.getByRole('button', { name: 'Open plans panel' })],
    ['home', () => screen.getByRole('link', { name: 'Home' })],
    ['theme', () => screen.getByRole('button', { name: /switch to dark mode/i })],
    ['profile', () => screen.getByRole('button', { name: LONG_NAME })],
  ])('should make the %s control at least 40×40px', async (_, control) => {
    await shell();

    const classes = classesOf(control());
    expect(classes).toContain('h-10');
    expect(classes.some((c) => c === 'w-10' || c === 'min-w-10')).toBe(true);
  });
});

describe('top bar — gutters (R19)', () => {
  it('should match the right padding below md and centre the logo on the rail from md', async () => {
    await shell();

    const row = topBar().firstElementChild!;
    expect(classesOf(row)).toEqual(expect.arrayContaining(['px-4', 'md:pl-2.5']));
  });
});

describe('top bar — stays on top of the page (R22)', () => {
  it('should stack above page cards that lift themselves to z-10, so its menu is never covered', async () => {
    await shell();

    // The plan title card is `relative z-10`; a tie lets it, later in the DOM,
    // paint over the bar on scroll and over the profile menu's "Sign Out".
    expect(classesOf(topBar())).toContain('z-30');
  });
});

describe('top bar — tokens only (R20)', () => {
  it('should carry no raw colour, gray/amber or black/white utilities', async () => {
    await shell();
    await within(topBar()).findByText(LONG_NAME);

    const offenders = [topBar(), ...topBar().querySelectorAll('[class]')]
      .flatMap(classesOf)
      .filter((c) => /#|rgb|(^|:)(bg|text|border|ring)-(gray|amber|black|white)\b/.test(c));
    expect(offenders).toEqual([]);
  });

  it('should give every control the coral focus ring', async () => {
    await shell();

    const bar = within(topBar());
    const controls = [...bar.getAllByRole('button'), ...bar.getAllByRole('link')];
    expect(controls.length).toBe(5); // menu, logo/title, home, theme, profile
    for (const control of controls) {
      expect(classesOf(control)).toContain('focus-visible:ring-ring');
    }
  });
});
