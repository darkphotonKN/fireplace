import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import HeroSection from './HeroSection';

const squash = (s: string | null) => (s ?? '').replace(/\s+/g, ' ').trim();

describe('HeroSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should carry the anchor phrase verbatim', () => {
    const { container } = render(<HeroSection />);

    expect(squash(container.textContent)).toContain('Start your plan now. Sit down by the fire.');
  });

  it('should point its CTAs at the unchanged auth targets', () => {
    render(<HeroSection />);

    expect(screen.getByRole('link', { name: /start your plan/i })).toHaveAttribute(
      'href',
      '/auth?tab=signup'
    );
    expect(screen.getByRole('link', { name: /sign in/i })).toHaveAttribute('href', '/auth');
  });

  it('should render exactly one heading, so the coral moment stays singular', () => {
    render(<HeroSection />);

    const headings = screen.getAllByRole('heading');
    expect(headings).toHaveLength(1);
    expect(headings[0].tagName).toBe('H1');
  });

  it('should hide the decorative glow from assistive tech', () => {
    const { container } = render(<HeroSection />);

    expect(container.querySelector('[aria-hidden="true"]')).not.toBeNull();
  });
});
