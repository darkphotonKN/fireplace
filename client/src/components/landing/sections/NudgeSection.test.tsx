import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
import { GATEWAY_TASK } from '../mocks/content';
import NudgeSection from './NudgeSection';

describe('NudgeSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should lead with a non-coral section heading', () => {
    render(<NudgeSection />);

    const heading = screen.getByRole('heading');
    expect(heading).toHaveTextContent('It notices what you keep skipping.');
    expect(heading.tagName).toBe('H2');
  });

  it('should render compact, so the beat lands as a grace note', () => {
    const { container } = render(<NudgeSection />);

    expect(container.querySelector('[data-section-size]')).toHaveAttribute(
      'data-section-size',
      'compact'
    );
  });

  it('should show exactly one suggestion, grounded in the plan already shown', () => {
    const { container } = render(<NudgeSection />);

    const mocks = container.querySelectorAll('[data-landing-mock]');
    expect(mocks).toHaveLength(1);
    // A generic nudge would not name the task the earlier mocks showed.
    expect(mocks[0].textContent).toContain(GATEWAY_TASK);
  });

  it('should honour the mock contract', () => {
    const { container } = render(<NudgeSection />);

    expectMockContract(container);
  });
});
