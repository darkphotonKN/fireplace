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
    // The hero's underline on "fire" is the page's one coral moment.
    expect(heading.className).toMatch(/\btext-foreground\b/);
    expect(heading.className).not.toMatch(/\b(?:text|decoration)-primary\b/);
  });

  it('should keep its copy, scaled up to sit with the hero', () => {
    render(<ReturnSection />);

    const heading = screen.getByRole('heading', { level: 2 });
    expect(heading).toHaveTextContent(/^Tomorrow, the same fire\.$/);
    // Starts at text-4xl and grows to text-6xl: the hero's x.ai scale, a step
    // below its own headline so the close does not compete with the h1.
    expect(heading.className).toMatch(/(?:^|\s)text-4xl\b/);
    expect(heading.className).toMatch(/\blg:text-6xl\b/);

    const body = screen.getByText(/^Plan it, work it, close it honestly\./);
    expect(body).toHaveTextContent(
      "Plan it, work it, close it honestly. What you wrote down is where tomorrow's list starts. Then you sit down and do it again."
    );
    expect(body.className).toMatch(/(?:^|\s)text-lg\b/);
    expect(body.className).toMatch(/\bmd:text-xl\b/);
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
    expect(primary.className).toMatch(/\bbg-action\b/);
    expect(quiet.className).not.toMatch(/\bbg-/);
  });

  it('should put the primary\'s text on its paired token, never hardcoded white', () => {
    render(<ReturnSection />);

    const primary = screen.getByRole('link', { name: /start your plan/i });
    expect(primary.className).toMatch(/\btext-action-foreground\b/);
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
