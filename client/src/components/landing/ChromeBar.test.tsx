import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, act, within } from '@testing-library/react';
import { mockMatchMedia } from '@/test/motion';

vi.mock('@/context/ThemeContext', () => ({
  useTheme: () => ({ theme: 'dark', toggleTheme: vi.fn() }),
}));

import ChromeBar from './ChromeBar';

/** Scroll and let the queued animation frame actually run. */
async function scrollTo(y: number) {
  window.scrollY = y;
  await act(async () => {
    window.dispatchEvent(new Event('scroll'));
    await new Promise((resolve) => setTimeout(resolve, 32));
  });
}

describe('ChromeBar', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    window.scrollY = 0;
  });

  it('should stay out of the way over the hero', () => {
    render(<ChromeBar />);

    const bar = screen.getByTestId('landing-chrome');
    expect(bar).toHaveAttribute('inert');
    expect(bar.className).toMatch(/opacity-0/);
  });

  it('should appear and become interactive once the hero is scrolled past', async () => {
    render(<ChromeBar />);

    await scrollTo(window.innerHeight);

    const bar = screen.getByTestId('landing-chrome');
    expect(bar).not.toHaveAttribute('inert');
    expect(bar.className).not.toMatch(/opacity-0/);
  });

  it('should hold logo, theme toggle, Sign in, then Start your plan, and no profile menu', async () => {
    render(<ChromeBar />);
    await scrollTo(window.innerHeight);

    const bar = screen.getByTestId('landing-chrome');
    expect(within(bar).getByText('Fireplace')).toBeInTheDocument();

    const controls = [...bar.querySelectorAll('a, button')];
    const toggle = within(bar).getByRole('button');
    const signIn = within(bar).getByRole('link', { name: 'Sign in' });
    const start = within(bar).getByRole('link', { name: 'Start your plan' });
    expect(controls).toEqual([toggle, signIn, start]);

    expect(signIn).toHaveAttribute('href', '/auth');
    expect(signIn.className).toMatch(/\bbg-muted\b/);
    expect(start).toHaveAttribute('href', '/auth?tab=signup');
    expect(start.className).toMatch(/\bbg-action\b/);
    for (const pill of [signIn, start]) {
      expect(pill.className).toMatch(/\brounded-control\b/);
      expect(pill.className).toMatch(/\btext-sm\b/);
      expect(pill.className).toMatch(/\bpx-4\b/);
      expect(pill.className).toMatch(/\bpy-2\b/);
      expect(pill.className).not.toMatch(/\b(px-8|py-3)\b/);
    }

    // Logged-in visitors never see the landing, so there is no account menu.
    expect(within(bar).queryByRole('menu')).toBeNull();
    expect(within(bar).queryByRole('button', { name: /profile|account|menu/i })).toBeNull();
  });

  // jsdom has no layout, so these pin the classes that keep the bar one line
  // at 320px; the real widths are checked in a browser.
  it('should keep each pill on one line rather than wrapping or squeezing it', () => {
    render(<ChromeBar />);

    const bar = screen.getByTestId('landing-chrome');
    for (const name of ['Sign in', 'Start your plan']) {
      const pill = within(bar).getByRole('link', { name });
      expect(pill.className).toMatch(/\bwhitespace-nowrap\b/);
      expect(pill.className).toMatch(/\bshrink-0\b/);
    }
  });

  it('should drop the wordmark below sm and keep the logo, so both pills fit a phone', () => {
    render(<ChromeBar />);

    const wordmark = within(screen.getByTestId('landing-chrome')).getByText('Fireplace');
    expect(wordmark.className).toMatch(/(^|\s)hidden(\s|$)/);
    expect(wordmark.className).toMatch(/(^|\s)sm:inline(\s|$)/);
  });

  it('should be fixed, so its arrival never shifts the content behind it', () => {
    render(<ChromeBar />);

    expect(screen.getByTestId('landing-chrome').className).toMatch(/fixed/);
  });
});
