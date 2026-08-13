import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import { expectMockContract } from '@/test/mockContract';
import ReviewSection from './ReviewSection';

describe('ReviewSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should lead with a non-coral section heading', () => {
    render(<ReviewSection />);

    const heading = screen.getByRole('heading');
    expect(heading).toHaveTextContent('Close the day honestly.');
    expect(heading.tagName).toBe('H2');
  });

  it('should honour the mock contract', () => {
    const { container } = render(<ReviewSection />);

    expectMockContract(container);
  });

  it('should render at full size', () => {
    const { container } = render(<ReviewSection />);

    expect(container.querySelector('[data-section-size]')).toHaveAttribute(
      'data-section-size',
      'default'
    );
  });
});
