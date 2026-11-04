import { describe, it, expect, beforeEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
import BentoGrid from './BentoGrid';

const here = (file: string) => readFileSync(resolve(__dirname, file), 'utf8');

describe('BentoGrid', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should hold the PLAN tile as a link to sign-up', () => {
    render(<BentoGrid />);

    const plan = screen.getByRole('link', { name: 'Plan Try it' });
    expect(plan).toHaveAttribute('href', '/auth?tab=signup');
  });

  it('should put TODAY second, inverted, with the id the hero badge points at', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const today = screen.getByRole('link', { name: 'Today Try it' });

    expect(tiles.indexOf(today)).toBe(1);
    expect(today).toHaveAttribute('id', 'today');
    expect(today).toHaveAttribute('href', '/auth?tab=signup');
    expect(today).toHaveClass('bg-secondary', 'text-secondary-foreground');
  });

  it('should put REVIEW third, a standard tile sharing row 1 equally at lg', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const review = screen.getByRole('link', { name: 'Review Try it' });

    expect(tiles.indexOf(review)).toBe(2);
    expect(review).toHaveAttribute('href', '/auth?tab=signup');
    expect(review).toHaveClass('bg-card', 'border-border');
    expect(review).not.toHaveClass('bg-secondary');
    // PLAN, TODAY, REVIEW: one column each of the lg grid's three.
    tiles.slice(0, 3).forEach((tile) => {
      expect(tile.className).not.toMatch(/(^|\s)(\w+:)?col-span-/);
    });
  });

  it('should put MILESTONES fourth, spanning two columns at lg only', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const milestones = screen.getByRole('link', { name: 'Milestones Try it' });

    expect(tiles.indexOf(milestones)).toBe(3);
    expect(milestones).toHaveAttribute('href', '/auth?tab=signup');
    expect(milestones).toHaveClass('lg:col-span-2', 'bg-card', 'border-border');
    expect(milestones.className).not.toMatch(/(^|\s)md:col-span-/);
    expect(milestones.querySelectorAll('[data-milestone-segment]')).toHaveLength(14);
  });

  it('should put CALENDAR fifth and last, across both columns at md and one of three at lg', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const calendar = screen.getByRole('link', { name: 'Calendar Try it' });

    expect(tiles).toHaveLength(5);
    expect(tiles.indexOf(calendar)).toBe(4);
    expect(calendar).toHaveAttribute('href', '/auth?tab=signup');
    expect(calendar).toHaveClass('md:col-span-2', 'lg:col-span-1', 'bg-card', 'border-border');
    expect(calendar.querySelectorAll('[data-calendar-day]').length).toBeGreaterThanOrEqual(28);
  });

  it('should draw nothing inside TODAY from the page tokens (R12)', () => {
    render(<BentoGrid />);
    const today = screen.getByRole('link', { name: 'Today Try it' });
    // Page tokens that would not read on the inverted fill.
    const page = /^(text|bg|border)-(foreground|muted-foreground|card|card-foreground|muted|border|background)(\/|$)/;

    today.querySelectorAll('*').forEach((el) => {
      const tokens = String(el.getAttribute('class') ?? '').split(/\s+/).filter(Boolean);
      expect(tokens.filter((t) => page.test(t))).toEqual([]);
    });
  });

  it('should honour the mock contract on every tile', () => {
    const { container } = render(<BentoGrid />);

    expectMockContract(container);
  });

  it('should step from one column to two at md and three at lg, in DOM order', () => {
    const { container } = render(<BentoGrid />);
    const grid = container.querySelector<HTMLElement>('[data-bento-grid]')!;

    expect(grid).toHaveClass('grid', 'grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3');
    // Visual order is DOM order, so tab order is what the eye sees (R10).
    expect(grid.className).not.toMatch(/grid-flow-dense|\border-/);
    grid.querySelectorAll('*').forEach((el) => {
      expect(String(el.getAttribute('class'))).not.toMatch(/(^|\s)(\w+:)?order-/);
    });
  });

  it('should rise in through Section and never be parallaxed', () => {
    const { container } = render(<BentoGrid />);
    const grid = container.querySelector<HTMLElement>('[data-bento-grid]')!;

    expect(grid.closest('[data-landing-reveal]')).not.toBeNull();
    container.querySelectorAll<HTMLElement>('*').forEach((el) => {
      expect(el.style.transform).toBe('');
    });
  });
});

describe('bento.module.css', () => {
  // Comments describe the triggers in prose; only the rules count.
  const css = here('bento.module.css').replace(/\/\*[\s\S]*?\*\//g, '');

  it('should gate hover behind (hover: hover) and reveal on :focus-visible', () => {
    const hoverBlock = /@media \(hover: hover\) \{([\s\S]*?)\n\}/.exec(css)?.[1] ?? '';
    const outside = css.replace(/@media \(hover: hover\) \{[\s\S]*?\n\}/, '');

    expect(hoverBlock).toMatch(/\.tile:hover/);
    expect(outside).not.toMatch(/:hover/);
    expect(css).toMatch(/\.tile:focus-visible/);
  });

  it('should animate only transform and opacity', () => {
    const transitions = Array.from(css.matchAll(/transition(?:-property)?:\s*([^;]+);/g)).map(
      (m) => m[1]
    );
    expect(transitions.length).toBeGreaterThan(0);

    for (const value of transitions) {
      if (value.trim() === 'none') continue;
      // Split on top-level commas only: `var(--reveal-delay, 0ms)` has its own.
      for (const part of value.replace(/\([^()]*\)/g, '()').split(',')) {
        expect(part.trim().split(/\s+/)[0]).toMatch(/^(transform|opacity)$/);
      }
    }
    expect(css).not.toMatch(/@keyframes/);
  });
});

describe('landing colour stays token-only (R24)', () => {
  // Every file this slice adds or reworks.
  const files = [
    './bento.module.css',
    './BentoGrid.tsx',
    './BentoTile.tsx',
    './reveal.tsx',
    '../mocks/PlanTileMock.tsx',
    '../mocks/TodayTileMock.tsx',
    '../mocks/ReviewTileMock.tsx',
    '../mocks/MilestonesTileMock.tsx',
    '../mocks/CalendarTileMock.tsx',
    '../mocks/content.ts',
  ];

  it.each(files)('%s has no raw colour', (file) => {
    const source = here(file);

    expect(source).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(source).not.toMatch(/\b(rgb|hsl)a?\((?!\s*var\()/);
    expect(source).not.toMatch(/-gray-|\bbg-white\b|\btext-white\b|\bbg-black\b/);
  });
});
