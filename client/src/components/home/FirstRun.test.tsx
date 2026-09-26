import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import FirstRun from './FirstRun';

/**
 * The first-run invitation (FS-KSJFR R43–R44, D13).
 *
 * What is pinned here is what the screen must *not* become: a navigation, a
 * second copy of the logged-out tour, or a heading with nothing under it.
 */
describe('FirstRun', () => {
  it('should say what a plan is, because the user has no way to know yet', () => {
    render(<FirstRun onStart={() => {}} />);

    expect(
      screen.getByText(/one thing you're building or learning/i),
    ).toBeInTheDocument();
  });

  it('should offer exactly one action', () => {
    render(<FirstRun onStart={() => {}} />);

    expect(screen.getAllByRole('button')).toHaveLength(1);
  });

  it('should hand back to the prompt rather than navigating (R44)', async () => {
    const onStart = vi.fn();
    const user = userEvent.setup();

    render(<FirstRun onStart={onStart} />);
    await user.click(screen.getByRole('button', { name: /start a plan/i }));

    expect(onStart).toHaveBeenCalledTimes(1);
    // A link would be a navigation; the shape of this screen is a state swap.
    expect(screen.queryAllByRole('link')).toHaveLength(0);
  });

  it('should carry the page its one h1, since no other block renders here', () => {
    const { container } = render(<FirstRun onStart={() => {}} />);

    expect(container.querySelectorAll('h1')).toHaveLength(1);
    expect(screen.getByRole('region')).toHaveAccessibleName('Your first plan');
  });

  it('should not restate the logged-out tour (R44)', () => {
    const { container } = render(<FirstRun onStart={() => {}} />);

    // The tour's anchor phrase and its sign-up framing belong to a surface for
    // people who have not signed up. This user already did.
    expect(container.textContent).not.toMatch(/sit down by the fire/i);
    expect(container.textContent).not.toMatch(/sign up|sign in/i);
  });

  it('should style from tokens only', () => {
    const { container } = render(<FirstRun onStart={() => {}} />);

    const markup = container.innerHTML;
    expect(markup).not.toMatch(/rgba?\(/);
    expect(markup).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(markup).not.toMatch(/-gray-/);
    expect(markup).not.toMatch(/dark:/);
  });
});
