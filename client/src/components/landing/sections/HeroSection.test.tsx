import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockMatchMedia, mockIntersectionObserver } from '@/test/motion';
import userEvent from '@testing-library/user-event';
import HeroSection from './HeroSection';

const squash = (s: string | null) => (s ?? '').replace(/\s+/g, ' ').trim();

describe('HeroSection', () => {
  beforeEach(() => {
    mockMatchMedia({ wide: true });
    mockIntersectionObserver();
  });

  it('should point its CTAs at the unchanged auth targets', () => {
    render(<HeroSection />);

    expect(screen.getByRole('link', { name: /start your plan/i })).toHaveAttribute(
      'href',
      '/auth?tab=signup'
    );
    expect(screen.getByRole('link', { name: /sign in/i })).toHaveAttribute('href', '/auth');
  });

  it('should set the anchor phrase verbatim as the h1', () => {
    render(<HeroSection />);

    const h1 = screen.getByRole('heading', { level: 1 });
    // The space before the break is the spoken pause between the halves.
    expect(h1.textContent).toBe('Start your plan now. Sit down by the fire.');
  });

  it('should link the badge, verbatim, to the Today tile and keep it in the tab order', async () => {
    const user = userEvent.setup();
    render(<HeroSection />);

    const badge = screen.getByRole('link', { name: 'New · Day plan with AI moves ↗' });
    expect(badge).toHaveAttribute('href', '#today');

    await user.tab();
    expect(badge).toHaveFocus();
  });

  it('should run eyebrow, badge, headline, subhead, then the two CTAs, with the badge tabbed first', async () => {
    const user = userEvent.setup();
    const { container } = render(<HeroSection />);

    const text = squash(container.textContent);
    const order = [
      'Fireplace',
      'New · Day plan with AI moves ↗',
      'Start your plan now. Sit down by the fire.',
      "The whole shape of a project, and the four things you're doing about it today. When the day closes, you write down how it actually went.",
      'Start your plan',
      'Sign in',
    ].map((part, i, all) => text.indexOf(part, i === 0 ? 0 : text.indexOf(all[i - 1]) + all[i - 1].length));
    order.forEach((at) => expect(at).toBeGreaterThanOrEqual(0));
    expect([...order].sort((a, b) => a - b)).toEqual(order);

    await user.tab();
    await user.tab();
    expect(screen.getByRole('link', { name: 'Start your plan' })).toHaveFocus();
    await user.tab();
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveFocus();
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
