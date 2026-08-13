import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
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

  it('should carry a theme toggle and a primary CTA', async () => {
    render(<ChromeBar />);
    await scrollTo(window.innerHeight);

    expect(screen.getByRole('link', { name: /start your plan/i })).toHaveAttribute(
      'href',
      '/auth?tab=signup'
    );
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should be fixed, so its arrival never shifts the content behind it', () => {
    render(<ChromeBar />);

    expect(screen.getByTestId('landing-chrome').className).toMatch(/fixed/);
  });
});
