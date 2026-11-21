import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver, stubElementTop, BELOW_FOLD } from '@/test/motion';
import ReturnSection from './ReturnSection';

describe('ReturnSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should keep its copy under one h2', () => {
    render(<ReturnSection />);

    const heading = screen.getByRole('heading', { level: 2 });
    expect(screen.getAllByRole('heading')).toEqual([heading]);
    expect(heading).toHaveTextContent(/^Tomorrow, the same fire\.$/);

    const body = screen.getByText(/^Plan it, work it, close it honestly\./);
    expect(body).toHaveTextContent(
      "Plan it, work it, close it honestly. What you wrote down is where tomorrow's list starts. Then you sit down and do it again."
    );
  });

  it('should offer two CTAs: Start your plan and Sign in', () => {
    render(<ReturnSection />);

    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(2);

    const primary = screen.getByRole('link', { name: /start your plan/i });
    const quiet = screen.getByRole('link', { name: /sign in/i });

    expect(primary).toHaveAttribute('href', '/auth?tab=signup');
    expect(quiet).toHaveAttribute('href', '/auth');
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
