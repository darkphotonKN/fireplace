import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';

// The chrome bar reuses the app's ThemeToggle; the rest renders provider-free.
vi.mock('@/context/ThemeContext', () => ({
  useTheme: () => ({ theme: 'dark', toggleTheme: vi.fn() }),
}));

import LandingTour from './LandingTour';

const TILE_NAMES = ['Plan Try it', 'Today Try it', 'Review Try it', 'Milestones Try it', 'Calendar Try it'];

/**
 * What Tab reaches, in sequential order: links and buttons in DOM order, minus
 * anything inside an `inert` subtree (the chrome bar while it is hidden).
 * jsdom does not implement inert, so the filter stands in for the browser.
 */
function tabStops(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>('a[href], button')).filter(
    (el) => !el.closest('[inert]')
  );
}

describe('LandingTour', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should render exactly chrome bar, hero, grid and close, in that order', () => {
    const { container } = render(<LandingTour />);
    const beats = Array.from(container.querySelector('main')!.children);

    expect(beats).toHaveLength(4);
    expect(beats[0]).toHaveAttribute('data-testid', 'landing-chrome');
    expect(beats[1]).toContainElement(screen.getByRole('heading', { level: 1 }));
    expect(beats[2].querySelector('[data-bento-grid]')).not.toBeNull();
    expect(beats[3]).toContainElement(screen.getByRole('heading', { name: 'Tomorrow, the same fire.' }));
  });

  it('should hold the five tiles in story order and nothing from the old sections', () => {
    const { container } = render(<LandingTour />);
    const grid = container.querySelector<HTMLElement>('[data-bento-grid]')!;

    const tiles = within(grid).getAllByRole('link');
    expect(tiles).toHaveLength(5);
    TILE_NAMES.forEach((name, i) => expect(tiles[i]).toBe(within(grid).getByRole('link', { name })));

    // The close is the only h2 left: the four story sections each brought one.
    expect(screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)).toEqual([
      'Tomorrow, the same fire.',
    ]);
  });

  it('should tab from the badge through the hero CTAs and the five tiles to the close CTAs', () => {
    const { container } = render(<LandingTour />);
    const [, hero, grid, close] = Array.from(container.querySelector('main')!.children) as HTMLElement[];

    const expected = [
      within(hero).getByRole('link', { name: 'New · Day plan with AI moves ↗' }),
      within(hero).getByRole('link', { name: 'Start your plan' }),
      within(hero).getByRole('link', { name: 'Sign in' }),
      ...TILE_NAMES.map((name) => within(grid).getByRole('link', { name })),
      within(close).getByRole('link', { name: 'Start your plan' }),
      within(close).getByRole('link', { name: 'Sign in' }),
    ];

    const stops = tabStops(container.querySelector('main')!);
    expect(stops).toHaveLength(expected.length);
    stops.forEach((stop, i) => expect(stop).toBe(expected[i]));
  });

  it('should spend coral on the underline under "fire" and nowhere else', () => {
    const { container } = render(<LandingTour />);

    // CTAs are on `--action`, and the glow is a backdrop at low alpha in an
    // inline style. Mocks are pictures of production screens (R13), whose bars
    // and ticks are coral there too; they are art, not a coral moment of the
    // page's own. Everything else counts.
    const coral = Array.from(container.querySelectorAll<HTMLElement>('[class]')).filter(
      (el) =>
        !el.closest('[data-landing-mock]') &&
        /\b(?:text|decoration|bg|border|ring|fill|stroke)-primary(?:\/\d+)?\b/.test(el.getAttribute('class')!)
    );

    expect(coral).toHaveLength(1);
    expect(coral[0]).toHaveTextContent(/^fire$/);
    expect(screen.getByRole('heading', { level: 1 })).toContainElement(coral[0]);
  });
});
