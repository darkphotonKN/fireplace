import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, within, fireEvent, act } from '@testing-library/react';

/**
 * FS-12ZMF R26: the sidebar lists shared plans beside owned ones, and only a
 * plan the caller is a member of carries the quiet "shared" mark.
 */

const listPlans = vi.fn();
const setIsCollapsed = vi.fn();
let isCollapsed = false;
let pathname = '/';

vi.mock('@/api/plans', () => ({ listPlans: (...a: unknown[]) => listPlans(...a) }));
vi.mock('next/navigation', () => ({ usePathname: () => pathname }));
vi.mock('./LayoutWrapper', () => ({
  useSidebar: () => ({ isCollapsed, setIsCollapsed }),
}));

import Sidebar from './Sidebar';

describe('Sidebar', () => {
  beforeEach(() => {
    isCollapsed = false;
    pathname = '/';
    setIsCollapsed.mockReset();
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

  describe('open (FS-S6WVY §Docked shell, §Look)', () => {
    it('should offer to hide the panel it controls', async () => {
      render(<Sidebar />);
      await screen.findByRole('link', { name: /My garden/ });

      const toggle = screen.getByRole('button', { name: 'Hide plans panel' });
      expect(toggle).toHaveAttribute('aria-expanded', 'true');
      const panelId = toggle.getAttribute('aria-controls');
      expect(panelId).toBeTruthy();
      expect(document.getElementById(panelId!)).toContainElement(
        screen.getByRole('link', { name: /My garden/ }),
      );
    });

    it('should hide by writing the choice through the toggle', async () => {
      render(<Sidebar />);
      await screen.findByRole('link', { name: /My garden/ });
      fireEvent.click(screen.getByRole('button', { name: 'Hide plans panel' }));
      expect(setIsCollapsed).toHaveBeenCalledWith(true);
    });

    it('should mark the plan being viewed as the current page, and only that one', async () => {
      pathname = '/plan/mine';
      render(<Sidebar />);

      const active = await screen.findByRole('link', { name: /My garden/ });
      expect(active).toHaveAttribute('aria-current', 'page');
      expect(screen.getByRole('link', { name: /Their kitchen/ })).not.toHaveAttribute(
        'aria-current',
      );
    });

    it('should expose which plan type is selected, and switch it', async () => {
      listPlans.mockResolvedValue([
        { id: 'p', name: 'A project', planType: 'project' },
        { id: 'l', name: 'A course', planType: 'learning' },
      ]);
      render(<Sidebar />);
      await screen.findByRole('link', { name: 'A project' });

      const projects = screen.getByRole('button', { name: 'Projects' });
      const learning = screen.getByRole('button', { name: 'Learning' });
      expect(projects).toHaveAttribute('aria-pressed', 'true');
      expect(learning).toHaveAttribute('aria-pressed', 'false');

      fireEvent.click(learning);
      expect(learning).toHaveAttribute('aria-pressed', 'true');
      expect(projects).toHaveAttribute('aria-pressed', 'false');
      expect(screen.getByRole('link', { name: 'A course' })).toBeInTheDocument();
      expect(screen.queryByRole('link', { name: 'A project' })).toBeNull();
    });

    it('should offer a quiet "New plan" link to create one', async () => {
      render(<Sidebar />);
      const newPlan = await screen.findByRole('link', { name: 'New plan' });
      expect(newPlan).toHaveAttribute('href', '/create-plan');
    });
  });

  describe('hidden, and the peek (FS-S6WVY §Docked shell, §Peek)', () => {
    /** The plans panel, found by the id the toggle says it controls. */
    const panel = () =>
      document.getElementById(
        screen.getByRole('button', { name: /plans panel/ }).getAttribute('aria-controls')!,
      )!;
    /** The rail: the docked column the panel and toggle live in. */
    const rail = () => panel().parentElement!;
    const wait = (ms: number) =>
      act(() => {
        vi.advanceTimersByTime(ms);
      });

    const renderHidden = async () => {
      isCollapsed = true;
      render(<Sidebar />);
      await act(async () => {}); // let the plan list settle
    };

    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it('should offer to show the panel, and keep its contents out of reach', async () => {
      await renderHidden();

      const toggle = screen.getByRole('button', { name: 'Show plans panel' });
      expect(toggle).toHaveAttribute('aria-expanded', 'false');
      expect(panel()).toHaveAttribute('inert');
    });

    it('should leave the open panel reachable', async () => {
      render(<Sidebar />);
      await act(async () => {});
      expect(panel()).not.toHaveAttribute('inert');
    });

    it('should peek after a short hover on the rail, without writing the choice', async () => {
      const setItem = vi.spyOn(Storage.prototype, 'setItem');
      await renderHidden();

      fireEvent.mouseEnter(rail());
      await wait(200);

      expect(panel()).not.toHaveAttribute('inert');
      expect(setIsCollapsed).not.toHaveBeenCalled();
      expect(setItem).not.toHaveBeenCalledWith('sidebar', expect.anything());
    });

    it('should not peek when the pointer only crosses the rail', async () => {
      await renderHidden();

      fireEvent.mouseEnter(rail());
      await wait(50);
      expect(panel()).toHaveAttribute('inert');
      fireEvent.mouseLeave(rail());
      await wait(300);

      expect(panel()).toHaveAttribute('inert');
    });

    it('should close the peek when the pointer leaves', async () => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);

      fireEvent.mouseLeave(rail());
      expect(panel()).toHaveAttribute('inert');
    });

    it('should close the peek on Escape', async () => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);

      fireEvent.keyDown(document, { key: 'Escape' });
      expect(panel()).toHaveAttribute('inert');
    });

    it('should close the peek when focus leaves it', async () => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);
      const link = screen.getByRole('link', { name: /My garden/ });

      fireEvent.focus(link);
      fireEvent.blur(link, { relatedTarget: document.body });
      expect(panel()).toHaveAttribute('inert');
    });

    it('should keep the peek while focus moves within it', async () => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);

      fireEvent.blur(screen.getByRole('link', { name: /My garden/ }), {
        relatedTarget: screen.getByRole('link', { name: /Their kitchen/ }),
      });
      expect(panel()).not.toHaveAttribute('inert');
    });

    it('should close the peek on navigation', async () => {
      isCollapsed = true;
      const { rerender } = render(<Sidebar />);
      await act(async () => {});
      fireEvent.mouseEnter(rail());
      await wait(200);

      pathname = '/plan/theirs';
      rerender(<Sidebar />);
      expect(panel()).toHaveAttribute('inert');
    });

    it('should not revive a peek after the panel was opened and hidden elsewhere', async () => {
      isCollapsed = true;
      const { rerender } = render(<Sidebar />);
      await act(async () => {});
      fireEvent.mouseEnter(rail());
      await wait(200);

      isCollapsed = false; // another tab pinned it open...
      rerender(<Sidebar />);
      isCollapsed = true; // ...then hid it again
      rerender(<Sidebar />);
      expect(panel()).toHaveAttribute('inert');
    });

    it('should still peek and pin when storage is blocked (R8)', async () => {
      vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new Error('SecurityError');
      });
      await renderHidden();

      fireEvent.mouseEnter(rail());
      await wait(200);
      expect(panel()).not.toHaveAttribute('inert');

      fireEvent.click(screen.getByRole('button', { name: 'Show plans panel' }));
      expect(setIsCollapsed).toHaveBeenCalledWith(false);
    });

    it.each([
      ['Escape', () => fireEvent.keyDown(document, { key: 'Escape' })],
      ['the pointer leaving', () => fireEvent.mouseLeave(rail())],
    ])('should hand focus to the toggle when %s closes a peek with focus inside', async (_how, close) => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);
      act(() => screen.getByRole('link', { name: /My garden/ }).focus());

      act(close);

      expect(panel()).toHaveAttribute('inert');
      expect(screen.getByRole('button', { name: 'Show plans panel' })).toHaveFocus();
    });

    it('should leave focus alone when a peek closes with focus elsewhere', async () => {
      const outside = document.createElement('button');
      document.body.appendChild(outside);
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);
      outside.focus();

      fireEvent.mouseLeave(rail());

      expect(outside).toHaveFocus();
      outside.remove();
    });

    it('should pin open when the toggle is clicked during a peek', async () => {
      await renderHidden();
      fireEvent.mouseEnter(rail());
      await wait(200);

      fireEvent.click(screen.getByRole('button', { name: 'Show plans panel' }));
      expect(setIsCollapsed).toHaveBeenCalledWith(false);
    });
  });
});
