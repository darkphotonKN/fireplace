import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';

/**
 * FS-12ZMF R26: the sidebar lists shared plans beside owned ones, and only a
 * plan the caller is a member of carries the quiet "shared" mark.
 */

const listPlans = vi.fn();

vi.mock('@/api/plans', () => ({ listPlans: (...a: unknown[]) => listPlans(...a) }));
vi.mock('next/navigation', () => ({ usePathname: () => '/' }));
vi.mock('./LayoutWrapper', () => ({
  useSidebar: () => ({ isCollapsed: false, setIsCollapsed: () => {} }),
}));
vi.mock('@/context/ThemeContext', () => ({ useTheme: () => ({ theme: 'light' }) }));

import Sidebar from './Sidebar';

describe('Sidebar', () => {
  beforeEach(() => {
    listPlans.mockReset().mockResolvedValue([
      { id: 'mine', name: 'My garden', planType: 'project', relation: 'owner' },
      { id: 'theirs', name: 'Their kitchen', planType: 'project', relation: 'member' },
    ]);
  });

  it('should mark a shared plan and leave an owned plan unmarked', async () => {
    render(<Sidebar />);

    const shared = await screen.findByRole('link', { name: /Their kitchen/ });
    const owned = screen.getByRole('link', { name: /My garden/ });
    expect(within(shared).getByText('shared')).toBeInTheDocument();
    expect(within(owned).queryByText('shared')).toBeNull();
  });
});
