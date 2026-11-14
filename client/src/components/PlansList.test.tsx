import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within, fireEvent } from '@testing-library/react';

/**
 * The dock and the narrow-width drawer show one plan list (FS-49CQ2 R4): one
 * fetch for the shell, and one Projects / Learning choice, whichever surface
 * makes it.
 */

const listPlans = vi.fn();

vi.mock('@/api/plans', () => ({ listPlans: (...a: unknown[]) => listPlans(...a) }));
vi.mock('next/navigation', () => ({ usePathname: () => '/' }));

import PlansPanel from './PlansPanel';
import { PlansListProvider } from './PlansList';

function twoSurfaces() {
  render(
    <PlansListProvider>
      <section aria-label="dock">
        <PlansPanel />
      </section>
      <section aria-label="drawer">
        <PlansPanel />
      </section>
    </PlansListProvider>,
  );
  return {
    dock: screen.getByRole('region', { name: 'dock' }),
    drawer: screen.getByRole('region', { name: 'drawer' }),
  };
}

describe('PlansListProvider', () => {
  beforeEach(() => {
    listPlans.mockReset().mockResolvedValue([
      { id: 'p', name: 'A project', planType: 'project' },
      { id: 'l', name: 'A course', planType: 'learning' },
    ]);
  });

  it('should fetch the plans once for every panel it serves', async () => {
    const { dock, drawer } = twoSurfaces();

    expect(await within(dock).findByRole('link', { name: /A project/ })).toBeInTheDocument();
    expect(within(drawer).getByRole('link', { name: /A project/ })).toBeInTheDocument();
    expect(listPlans).toHaveBeenCalledTimes(1);
  });

  it('should share the filter between the panels', async () => {
    const { dock, drawer } = twoSurfaces();
    await within(dock).findByRole('link', { name: /A project/ });

    fireEvent.click(within(dock).getByRole('button', { name: 'Learning' }));

    expect(within(drawer).getByRole('button', { name: 'Learning' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(within(drawer).getByRole('link', { name: /A course/ })).toBeInTheDocument();
  });

  it('should not fetch until a panel shows', () => {
    render(<PlansListProvider>closed drawer</PlansListProvider>);
    expect(listPlans).not.toHaveBeenCalled();
  });

  it('should say the list failed to load, in every panel', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    listPlans.mockRejectedValue(new Error('down'));
    const { dock, drawer } = twoSurfaces();

    expect(await within(dock).findByText('Failed to load plans')).toBeInTheDocument();
    expect(within(drawer).getByText('Failed to load plans')).toBeInTheDocument();
    expect(listPlans).toHaveBeenCalledTimes(1);
  });
});
