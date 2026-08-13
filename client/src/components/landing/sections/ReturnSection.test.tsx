import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver, stubElementTop, BELOW_FOLD } from '@/test/motion';
import ReturnSection from './ReturnSection';

describe('ReturnSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should close on a non-coral heading', () => {
    render(<ReturnSection />);

    const heading = screen.getByRole('heading');
    expect(heading.tagName).toBe('H2');
  });

  it('should offer one primary action and a quiet sign-in, not a matched pair', () => {
    render(<ReturnSection />);

    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(2);

    const primary = screen.getByRole('link', { name: /start your plan/i });
    const quiet = screen.getByRole('link', { name: /sign in/i });

    expect(primary).toHaveAttribute('href', '/auth?tab=signup');
    expect(quiet).toHaveAttribute('href', '/auth');

    // The primary is a filled button; the sign-in is a text link. By this point
    // the visitor is warm, and two equal-weight buttons read as hesitation.
    expect(primary.className).toMatch(/bg-primary/);
    expect(quiet.className).not.toMatch(/bg-primary/);
  });

  it('should put on-coral text on the primary via the token, never hardcoded white', () => {
    render(<ReturnSection />);

    const primary = screen.getByRole('link', { name: /start your plan/i });
    expect(primary.className).toMatch(/text-primary-foreground/);
    expect(primary.className).not.toMatch(/text-white/);
  });

  it('should reprise the hearth glow from the hero', () => {
    const { container } = render(<ReturnSection />);

    expect(container.querySelector('[aria-hidden="true"]')).not.toBeNull();
  });

  it('should keep its CTAs out of reach until the section is revealed', () => {
    // This section sits below the fold and carries the page's closing CTAs, so
    // it is the one place where hiding-by-opacity would strand a keyboard
    // visitor on an invisible button.
    const observer = mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);
    const { container } = render(<ReturnSection />);

    const revealed = container.querySelector('[data-landing-reveal]') as HTMLElement;
    expect(revealed).toHaveAttribute('inert');

    act(() => observer.enter());
    expect(revealed).not.toHaveAttribute('inert');
  });
});
