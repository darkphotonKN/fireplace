import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
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

  it('should honour the mock contract', () => {
    const { container } = render(<PlanSection />);

    expectMockContract(container);
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
