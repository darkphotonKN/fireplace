import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
import BentoGrid from './BentoGrid';

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

  it('should put TODAY second, with the id the hero badge points at', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const today = screen.getByRole('link', { name: 'Today Try it' });

    expect(tiles.indexOf(today)).toBe(1);
    expect(today).toHaveAttribute('id', 'today');
    expect(today).toHaveAttribute('href', '/auth?tab=signup');
  });

  it('should put REVIEW third', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const review = screen.getByRole('link', { name: 'Review Try it' });

    expect(tiles.indexOf(review)).toBe(2);
    expect(review).toHaveAttribute('href', '/auth?tab=signup');
  });

  it('should put MILESTONES fourth, with all fourteen segments', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const milestones = screen.getByRole('link', { name: 'Milestones Try it' });

    expect(tiles.indexOf(milestones)).toBe(3);
    expect(milestones).toHaveAttribute('href', '/auth?tab=signup');
    expect(milestones.querySelectorAll('[data-milestone-segment]')).toHaveLength(14);
  });

  it('should put CALENDAR fifth and last, with a whole month of days', () => {
    const { container } = render(<BentoGrid />);
    const tiles = Array.from(container.querySelectorAll<HTMLElement>('[data-bento-grid] > a'));
    const calendar = screen.getByRole('link', { name: 'Calendar Try it' });

    expect(tiles).toHaveLength(5);
    expect(tiles.indexOf(calendar)).toBe(4);
    expect(calendar).toHaveAttribute('href', '/auth?tab=signup');
    expect(calendar.querySelectorAll('[data-calendar-day]').length).toBeGreaterThanOrEqual(28);
  });

  it('should honour the mock contract on every tile', () => {
    const { container } = render(<BentoGrid />);

    expectMockContract(container);
  });

  it('should keep visual order equal to DOM order', () => {
    const { container } = render(<BentoGrid />);
    const grid = container.querySelector<HTMLElement>('[data-bento-grid]')!;

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
