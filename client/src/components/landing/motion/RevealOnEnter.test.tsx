import { describe, it, expect } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import {
  mockMatchMedia,
  mockIntersectionObserver,
  removeIntersectionObserver,
  stubElementTop,
  BELOW_FOLD,
} from '@/test/motion';
import RevealOnEnter from './RevealOnEnter';

describe('RevealOnEnter', () => {
  it('should render its children visible when there is no IntersectionObserver', () => {
    mockMatchMedia({ wide: true });
    removeIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <p>Every arc gets a plan.</p>
      </RevealOnEnter>
    );

    expect(screen.getByText('Every arc gets a plan.')).toBeVisible();
    expect(screen.getByTestId('section').className).not.toMatch(/opacity-0/);
  });

  it('should hide below-fold content and reveal it on entry', () => {
    mockMatchMedia({ wide: true });
    const observer = mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <p>Every arc gets a plan.</p>
      </RevealOnEnter>
    );
    expect(screen.getByTestId('section').className).toMatch(/opacity-0/);

    act(() => observer.enter());

    const section = screen.getByTestId('section');
    expect(section.className).not.toMatch(/opacity-0/);
    expect(section.className).toMatch(/animate-riseIn/);
  });

  it('should reveal content flung past without ever intersecting', () => {
    mockMatchMedia({ wide: true });
    const observer = mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <p>Every arc gets a plan.</p>
      </RevealOnEnter>
    );
    expect(screen.getByTestId('section').className).toMatch(/opacity-0/);

    // Fling to the bottom: the section is now ABOVE the viewport and was never
    // reported as intersecting. It must not stay invisible.
    act(() => observer.flungPast());

    expect(screen.getByTestId('section').className).not.toMatch(/opacity-0/);
    expect(screen.getByText('Every arc gets a plan.')).toBeVisible();
  });

  it('should take hidden content out of the tab order and the a11y tree', () => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <a href="/auth">Start your plan</a>
      </RevealOnEnter>
    );

    // `opacity-0` alone would leave this link focusable and announced: a keyboard
    // visitor tabbing before scrolling would land on an invisible button.
    const section = screen.getByTestId('section');
    expect(section).toHaveAttribute('inert');
    expect(section.className).toMatch(/invisible/);
  });

  it('should restore interactivity when revealed', () => {
    mockMatchMedia({ wide: true });
    const observer = mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <a href="/auth">Start your plan</a>
      </RevealOnEnter>
    );
    act(() => observer.enter());

    const section = screen.getByTestId('section');
    expect(section).not.toHaveAttribute('inert');
    expect(section.className).not.toMatch(/invisible/);
  });

  it('should stay revealed once revealed', () => {
    mockMatchMedia({ wide: true });
    const observer = mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <p>Every arc gets a plan.</p>
      </RevealOnEnter>
    );
    act(() => observer.enter());
    act(() => observer.exitBelow());

    expect(screen.getByTestId('section').className).not.toMatch(/opacity-0/);
  });

  it('should never hide content when the user prefers reduced motion', () => {
    mockMatchMedia({ wide: true, reducedMotion: true });
    mockIntersectionObserver();
    stubElementTop(BELOW_FOLD);

    render(
      <RevealOnEnter data-testid="section">
        <p>Every arc gets a plan.</p>
      </RevealOnEnter>
    );

    const section = screen.getByTestId('section');
    expect(section.className).not.toMatch(/opacity-0/);
    expect(section.className).not.toMatch(/animate-riseIn/);
    expect(screen.getByText('Every arc gets a plan.')).toBeVisible();
  });
});
