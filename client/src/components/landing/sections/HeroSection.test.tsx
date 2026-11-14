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

  it('should carry the anchor phrase verbatim', () => {
    const { container } = render(<HeroSection />);

    expect(squash(container.textContent)).toContain('Start your plan now. Sit down by the fire.');
  });

  it('should point its CTAs at the unchanged auth targets', () => {
    render(<HeroSection />);

    expect(screen.getByRole('link', { name: /start your plan/i })).toHaveAttribute(
      'href',
      '/auth?tab=signup'
    );
    expect(screen.getByRole('link', { name: /sign in/i })).toHaveAttribute('href', '/auth');
  });

  it('should set the anchor phrase as the one h1, at foreground and x.ai scale', () => {
    render(<HeroSection />);

    const h1 = screen.getByRole('heading', { level: 1 });
    // The space before the break is the spoken pause between the halves.
    expect(h1.textContent).toBe('Start your plan now. Sit down by the fire.');
    expect(h1.className).toMatch(/\btext-foreground\b/);
    expect(h1.className).toMatch(/(^|\s)text-(5xl|6xl|7xl|8xl)(\s|$)/);
    expect(h1.className).toMatch(/\blg:text-(7xl|8xl)\b/);
  });

  it('should underline "fire" in coral, the page\'s one coral moment', () => {
    render(<HeroSection />);

    const h1 = screen.getByRole('heading', { level: 1 });
    const underlined = [...h1.querySelectorAll('*')].filter((el) =>
      /\bdecoration-primary\b/.test(el.className)
    );
    expect(underlined).toHaveLength(1);
    expect(underlined[0].textContent).toBe('fire');
    expect(underlined[0].className).toMatch(/(^|\s)underline(\s|$)/);
  });

  it('should link the badge, verbatim, to the Today tile and keep it in the tab order', async () => {
    const user = userEvent.setup();
    render(<HeroSection />);

    const badge = screen.getByRole('link', { name: 'New · Day plan with AI moves ↗' });
    expect(badge).toHaveAttribute('href', '#today');

    await user.tab();
    expect(badge).toHaveFocus();
  });

  it('should make "Start your plan" the solid action pill and "Sign in" the grey pill', () => {
    render(<HeroSection />);

    const start = screen.getByRole('link', { name: 'Start your plan' });
    const signIn = screen.getByRole('link', { name: 'Sign in' });
    expect(start.className).toMatch(/\bbg-action\b/);
    expect(start.className).toMatch(/\btext-action-foreground\b/);
    expect(signIn.className).toMatch(/\bbg-muted\b/);
    expect(signIn.className).toMatch(/\btext-foreground\b/);
    for (const pill of [start, signIn]) expect(pill.className).toMatch(/\brounded-control\b/);
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

    // The subhead sits at x.ai's subhead scale, not body size.
    const subhead = screen.getByText(/The whole shape of a project/);
    expect(subhead.className).toMatch(/\btext-(xl|2xl)\b/);

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
