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
  });

  it('should appear and become interactive once the hero is scrolled past', async () => {
    render(<ChromeBar />);

    await scrollTo(window.innerHeight);

    const bar = screen.getByTestId('landing-chrome');
    expect(bar).not.toHaveAttribute('inert');
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
    expect(start).toHaveAttribute('href', '/auth?tab=signup');

    // Logged-in visitors never see the landing, so there is no account menu.
    expect(within(bar).queryByRole('menu')).toBeNull();
    expect(within(bar).queryByRole('button', { name: /profile|account|menu/i })).toBeNull();
  });
});
