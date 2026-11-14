import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act, within } from '@testing-library/react';

/**
 * Below md the plans panel is a drawer behind a menu button in the top bar,
 * not a dock (FS-49CQ2 R1–R10). Its open state is local and never remembered
 * (R6): nothing here may write the `sidebar` key that the docked widths own
 * (FS-S6WVY R3).
 *
 * The shell is rendered whole, with the dock stubbed out, because the drawer's
 * contract spans the top bar (the menu button), the overlay and main (inert
 * while open).
 */

const listPlans = vi.fn();
let pathname = '/plan/mine';

vi.mock('@/api/plans', () => ({ listPlans: (...a: unknown[]) => listPlans(...a) }));
vi.mock('@/components/ui/use-toast', () => ({ toast: vi.fn() }));
vi.mock('next/navigation', () => ({ usePathname: () => pathname }));
vi.mock('@/context/AuthContext', () => ({ useAuth: () => ({ isAuthenticated: true }) }));
vi.mock('./Sidebar', () => ({ default: () => <div data-testid="dock" /> }));
vi.mock('./UserProfile', () => ({ default: () => null }));
vi.mock('./Logo', () => ({ default: () => null }));
vi.mock('./ThemeToggle', () => ({ default: () => null }));
vi.mock('./AuthGuard', () => ({
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import LayoutContent from './LayoutContent';

/**
 * A `matchMedia` whose `(min-width: 768px)` answer a test can flip, as a
 * resize or rotation would.
 */
function stubViewport(initiallyDocked: boolean) {
  let docked = initiallyDocked;
  const listeners = new Set<() => void>();
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    get matches() {
      return query.includes('min-width: 768px') ? docked : false;
    },
    media: query,
    onchange: null,
    addEventListener: (_: string, cb: () => void) => listeners.add(cb),
    removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
    addListener: (cb: () => void) => listeners.add(cb),
    removeListener: (cb: () => void) => listeners.delete(cb),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia;
  return (next: boolean) =>
    act(() => {
      docked = next;
      listeners.forEach((cb) => cb());
    });
}

const shell = () => render(<LayoutContent>page body</LayoutContent>);
const menuButton = () => screen.getByRole('button', { name: /plans panel/ });
const drawer = () => screen.queryByRole('complementary', { name: 'Plans panel' });

async function openDrawer() {
  fireEvent.click(menuButton());
  await screen.findByRole('link', { name: /My garden/ });
}

const sidebarWrites = (setItem: ReturnType<typeof vi.spyOn>) =>
  setItem.mock.calls.filter(([key]) => key === 'sidebar' || key === 'sidebarCollapsed');

const originalMatchMedia = window.matchMedia;

/** jsdom can't follow a link; keep the tap, drop the navigation it would start. */
const stayOnPage = (event: Event) => event.preventDefault();

beforeEach(() => {
  document.addEventListener('click', stayOnPage);
  pathname = '/plan/mine';
  localStorage.clear();
  document.documentElement.removeAttribute('data-sidebar');
  document.body.style.overflow = '';
  stubViewport(false);
  listPlans.mockReset().mockResolvedValue([
    { id: 'mine', name: 'My garden', planType: 'project', relation: 'owner' },
    { id: 'theirs', name: 'Their kitchen', planType: 'project', relation: 'member' },
  ]);
});
afterEach(() => {
  document.removeEventListener('click', stayOnPage);
  window.matchMedia = originalMatchMedia;
});

describe('PlansDrawer — the menu button (R3, R8, R10)', () => {
  it.each([['open'], ['hidden'], [null]])(
    'should start closed whatever `sidebar` holds (%s)',
    (stored) => {
      if (stored) {
        localStorage.setItem('sidebar', stored);
        document.documentElement.setAttribute('data-sidebar', stored);
      }
      shell();

      const button = screen.getByRole('button', { name: 'Open plans panel' });
      expect(button).toHaveAttribute('aria-expanded', 'false');
      expect(drawer()).toBeNull();
    },
  );

  it('should point at the drawer it controls, and show only below md', () => {
    shell();
    const button = menuButton();
    const id = button.getAttribute('aria-controls');
    expect(id).toBeTruthy();
    expect(document.getElementById(id!)).not.toBeNull();
    expect(button).toHaveClass('md:hidden');
  });
});

describe('PlansDrawer — open (R4, R8)', () => {
  it('should show the shared panel over a scrim and move focus into it', async () => {
    shell();
    await openDrawer();

    const button = screen.getByRole('button', { name: 'Close plans panel' });
    expect(button).toHaveAttribute('aria-expanded', 'true');
    const panel = drawer()!;
    expect(document.getElementById(button.getAttribute('aria-controls')!)).toContainElement(panel);
    expect(within(panel).getByRole('button', { name: 'Projects' })).toHaveFocus();
    expect(within(panel).getByRole('button', { name: 'Learning' })).toBeInTheDocument();
    expect(
      within(within(panel).getByRole('link', { name: /Their kitchen/ })).getByText('shared'),
    ).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: /New plan/ })).toHaveAttribute(
      'href',
      '/create-plan',
    );
    expect(screen.getByTestId('plans-drawer-scrim')).toBeInTheDocument();
  });

  it('should make main unreachable and stop the page scrolling while open', async () => {
    shell();
    const main = screen.getByRole('main');
    expect(main.closest('[inert]')).toBeNull();

    await openDrawer();
    expect(main.closest('[inert]')).not.toBeNull();
    expect(document.body.style.overflow).toBe('hidden');

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(main.closest('[inert]')).toBeNull();
    expect(document.body.style.overflow).toBe('');
  });
});

describe('PlansDrawer — closing (R5, R7, R8)', () => {
  it('should close on Escape and return focus to the menu button', async () => {
    shell();
    await openDrawer();
    fireEvent.keyDown(document, { key: 'Escape' });

    expect(drawer()).toBeNull();
    expect(screen.getByRole('button', { name: 'Open plans panel' })).toHaveFocus();
  });

  it('should close on a scrim tap and return focus to the menu button', async () => {
    shell();
    await openDrawer();
    fireEvent.click(screen.getByTestId('plans-drawer-scrim'));

    expect(drawer()).toBeNull();
    expect(menuButton()).toHaveFocus();
  });

  it('should close when the menu button is pressed again, keeping focus on it', async () => {
    shell();
    await openDrawer();
    menuButton().focus();
    fireEvent.click(menuButton());

    expect(drawer()).toBeNull();
    expect(menuButton()).toHaveFocus();
  });

  it('should close on a tap on the current plan, though the route does not change', async () => {
    shell();
    await openDrawer();
    fireEvent.click(within(drawer()!).getByRole('link', { name: /My garden/ }));

    expect(drawer()).toBeNull();
  });

  it('should close on a tap on Home in the top bar, already on home', async () => {
    pathname = '/';
    shell();
    await openDrawer();
    fireEvent.click(screen.getByRole('link', { name: /Home/ }));

    expect(drawer()).toBeNull();
  });

  it('should close on a route change', async () => {
    const { rerender } = shell();
    await openDrawer();

    pathname = '/plan/theirs';
    rerender(<LayoutContent>page body</LayoutContent>);
    expect(drawer()).toBeNull();
  });

  it('should close when the viewport grows to docked width, and stay closed on shrinking back', async () => {
    const resize = stubViewport(false);
    shell();
    await openDrawer();

    resize(true);
    expect(drawer()).toBeNull();
    resize(false);
    expect(drawer()).toBeNull();
    expect(menuButton()).toHaveAttribute('aria-expanded', 'false');
  });
});

describe('PlansDrawer — one list for the page (R4)', () => {
  it('should reopen with the list already there and the filter kept, without fetching again', async () => {
    listPlans.mockResolvedValue([
      { id: 'mine', name: 'My garden', planType: 'project', relation: 'owner' },
      { id: 'notes', name: 'Go notes', planType: 'learning', relation: 'owner' },
    ]);
    shell();
    await openDrawer();
    fireEvent.click(within(drawer()!).getByRole('button', { name: 'Learning' }));
    fireEvent.keyDown(document, { key: 'Escape' });

    fireEvent.click(menuButton());
    const panel = drawer()!;
    expect(within(panel).queryByText('Loading…')).toBeNull();
    expect(within(panel).getByRole('link', { name: /Go notes/ })).toBeInTheDocument();
    expect(within(panel).getByRole('button', { name: 'Learning' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(listPlans).toHaveBeenCalledTimes(1);
  });
});

describe('PlansDrawer — never remembered (R6)', () => {
  it('should never write the remembered choice while opening, closing and navigating', async () => {
    localStorage.setItem('sidebar', 'hidden');
    document.documentElement.setAttribute('data-sidebar', 'hidden');
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const { rerender } = shell();

    await openDrawer();
    fireEvent.keyDown(document, { key: 'Escape' });
    await openDrawer();
    fireEvent.click(screen.getByTestId('plans-drawer-scrim'));
    await openDrawer();
    pathname = '/plan/theirs';
    rerender(<LayoutContent>page body</LayoutContent>);

    expect(sidebarWrites(setItem)).toEqual([]);
    expect(localStorage.getItem('sidebar')).toBe('hidden');
    expect(document.documentElement.getAttribute('data-sidebar')).toBe('hidden');
  });
});

describe('Shell gutters (R15)', () => {
  it('should pad main p-4 below md and p-8 at md and up, under the top bar', () => {
    shell();
    expect(screen.getByRole('main')).toHaveClass('pt-20', 'px-4', 'pb-4', 'md:px-8', 'md:pb-8');
  });
});
