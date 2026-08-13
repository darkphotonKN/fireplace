import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
import DailySection from './DailySection';

describe('DailySection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should lead with a non-coral section heading', () => {
    render(<DailySection />);

    const heading = screen.getByRole('heading');
    expect(heading).toHaveTextContent('Today is a short list.');
    expect(heading.tagName).toBe('H2');
  });

  it('should show the two scopes as distinct groups', () => {
    const { container } = render(<DailySection />);
    const mock = container.querySelector('[data-landing-mock]') as HTMLElement;

    // The dual-scope idea has to be legible at a glance — that is the whole
    // point of this beat, so the mock must actually separate the two.
    expect(mock.textContent).toMatch(/today/i);
    expect(mock.textContent).toMatch(/long.?term/i);
  });

  it('should honour the mock contract', () => {
    const { container } = render(<DailySection />);

    expectMockContract(container);
  });
});
