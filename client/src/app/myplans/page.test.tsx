import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';

/**
 * FS-12ZMF R26, R27: My Plans lists shared plans beside owned ones, marks the
 * shared ones, and offers delete only on plans the caller owns.
 */

const listPlans = vi.fn();

vi.mock('@/api/plans', () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
  searchPlans: vi.fn(),
  deletePlan: vi.fn(),
}));
vi.mock('@/context/ThemeContext', () => ({ useTheme: () => ({ theme: 'light' }) }));

import MyPlans from './page';

/** The card wrapper around a plan's link: where its delete control lives. */
const cardOf = (name: RegExp) =>
  screen.getByRole('link', { name }).parentElement as HTMLElement;

describe('My Plans', () => {
  beforeEach(() => {
    listPlans.mockReset().mockResolvedValue([
      { id: 'theirs', name: 'Their kitchen', planType: 'project', relation: 'member' },
      { id: 'mine', name: 'My garden', planType: 'project', relation: 'owner' },
    ]);
  });

  it('should mark a shared plan and leave an owned plan unmarked', async () => {
    render(<MyPlans />);
    await screen.findByRole('link', { name: /their kitchen/i });

    expect(within(cardOf(/their kitchen/i)).getByText('shared')).toBeInTheDocument();
    expect(within(cardOf(/my garden/i)).queryByText('shared')).toBeNull();
  });

  it('should offer delete on an owned plan only (R27)', async () => {
    render(<MyPlans />);
    await screen.findByRole('link', { name: /their kitchen/i });

    expect(
      within(cardOf(/their kitchen/i)).queryByRole('button', { name: 'Delete plan' }),
    ).toBeNull();
    expect(
      within(cardOf(/my garden/i)).getByRole('button', { name: 'Delete plan' }),
    ).toBeInTheDocument();
  });
});
