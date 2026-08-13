import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import PlanSection from './PlanSection';

describe('PlanSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should lead with a non-coral section heading', () => {
    render(<PlanSection />);

    const heading = screen.getByRole('heading');
    expect(heading).toHaveTextContent('Every arc gets a plan.');
    // h2, not h1: the hero owns the page's only coral heading (R14).
    expect(heading.tagName).toBe('H2');
  });

  it('should hide its mock art from assistive tech', () => {
    const { container } = render(<PlanSection />);

    const mock = container.querySelector('[data-landing-mock]');
    expect(mock).not.toBeNull();
    expect(mock).toHaveAttribute('aria-hidden', 'true');
  });

  it('should expose nothing interactive inside the mock', () => {
    const { container } = render(<PlanSection />);
    const mock = container.querySelector('[data-landing-mock]') as HTMLElement;

    // Fake checkboxes must not be real inputs, and nothing decorative may be
    // focusable or announced as a control — a screen reader should hear the
    // section's copy, not a recital of invented task labels.
    expect(mock.querySelectorAll('input, button, a, [tabindex], [role]')).toHaveLength(0);
  });

  it('should drift the mock art while leaving the copy at native speed', () => {
    const { container } = render(<PlanSection />);

    const artLayer = container.querySelector('[data-landing-mock]')!.parentElement!;
    expect(artLayer.style.transform).toMatch(/translate3d/);

    const copy = screen.getByRole('heading').parentElement!;
    expect(copy.style.transform).toBe('');
  });

  it('should render with no app context providers at all', () => {
    // Rendering bare is the assertion: a mock that reached for AuthContext, the
    // theme, or the service layer would throw here.
    expect(() => render(<PlanSection />)).not.toThrow();
  });
});
