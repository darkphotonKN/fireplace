import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, act, fireEvent } from '@testing-library/react';
import { SIDEBAR_HEAD_SCRIPT } from '@/lib/sidebarState';

/**
 * The sidebar's remembered open/hidden choice, as the app sees it
 * (FS-S6WVY R1–R8).
 *
 * The toggle (`setIsCollapsed`) is the only thing that may write the choice.
 * Everything else, including another tab changing it, only reads.
 */

vi.mock('@/context/AuthContext', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock('@/context/ThemeContext', () => ({
  ThemeProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock('@/components/ui/Toaster', () => ({ default: () => null }));

import LayoutWrapper, { useSidebar } from './LayoutWrapper';

/** Shows the state and exposes the toggle, the way Sidebar uses them. */
function Probe() {
  const { isCollapsed, setIsCollapsed } = useSidebar();
  return (
    <>
      <p>{isCollapsed ? 'hidden' : 'open'}</p>
      <button onClick={() => setIsCollapsed(true)}>hide</button>
      <button onClick={() => setIsCollapsed(false)}>pin</button>
    </>
  );
}

/** A page load: the head script runs, then React renders. */
const load = () => {
  new Function(SIDEBAR_HEAD_SCRIPT)();
  return render(
    <LayoutWrapper>
      <Probe />
    </LayoutWrapper>,
  );
};

const shown = () => screen.getByRole('paragraph').textContent;
const html = document.documentElement;

beforeEach(() => {
  html.removeAttribute('data-sidebar');
  html.removeAttribute('data-sidebar-ready');
});

describe('LayoutWrapper: loading the remembered choice', () => {
  it('should be open with nothing stored (R1)', () => {
    load();

    expect(shown()).toBe('open');
  });

  it('should be hidden when hidden was stored', () => {
    localStorage.setItem('sidebar', 'hidden');

    load();

    expect(shown()).toBe('hidden');
  });

  it('should ignore a legacy sidebarCollapsed=true and load open (R4)', () => {
    localStorage.setItem('sidebarCollapsed', 'true');

    load();

    expect(shown()).toBe('open');
  });

  it('should load open on an unknown stored value', () => {
    localStorage.setItem('sidebar', 'sideways');

    load();

    expect(shown()).toBe('open');
  });

  it('should not write anything just by loading', () => {
    localStorage.setItem('sidebar', 'hidden');
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const removeItem = vi.spyOn(Storage.prototype, 'removeItem');

    load();

    expect(setItem).not.toHaveBeenCalled();
    expect(removeItem).not.toHaveBeenCalled();
  });
});

describe('LayoutWrapper: the toggle is the only writer (R3)', () => {
  it('should remember hidden, and update the painted state', () => {
    load();

    fireEvent.click(screen.getByText('hide'));

    expect(shown()).toBe('hidden');
    expect(localStorage.getItem('sidebar')).toBe('hidden');
    expect(html.getAttribute('data-sidebar')).toBe('hidden');
  });

  it('should remember open when pinned', () => {
    localStorage.setItem('sidebar', 'hidden');
    load();

    fireEvent.click(screen.getByText('pin'));

    expect(shown()).toBe('open');
    expect(localStorage.getItem('sidebar')).toBe('open');
    expect(html.getAttribute('data-sidebar')).toBe('open');
  });

  it('should remove the legacy key on the first toggle (R4)', () => {
    localStorage.setItem('sidebarCollapsed', 'true');
    load();

    fireEvent.click(screen.getByText('hide'));

    expect(localStorage.getItem('sidebarCollapsed')).toBeNull();
  });

  it('should still toggle in-page when storage throws on read and write (R8)', () => {
    const blocked = () => {
      throw new Error('SecurityError');
    };
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(blocked);
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(blocked);
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(blocked);

    load();
    expect(shown()).toBe('open');

    fireEvent.click(screen.getByText('hide'));
    expect(shown()).toBe('hidden');

    fireEvent.click(screen.getByText('pin'));
    expect(shown()).toBe('open');
  });
});

describe('LayoutWrapper: other tabs (R7)', () => {
  const fromAnotherTab = (key: string | null, newValue: string | null) =>
    act(() => {
      window.dispatchEvent(new StorageEvent('storage', { key, newValue }));
    });

  it('should follow another tab hiding the sidebar, without writing', () => {
    load();
    const setItem = vi.spyOn(Storage.prototype, 'setItem');
    const removeItem = vi.spyOn(Storage.prototype, 'removeItem');

    fromAnotherTab('sidebar', 'hidden');

    expect(shown()).toBe('hidden');
    expect(html.getAttribute('data-sidebar')).toBe('hidden');
    expect(setItem).not.toHaveBeenCalled();
    expect(removeItem).not.toHaveBeenCalled();
  });

  it('should follow another tab pinning it open', () => {
    localStorage.setItem('sidebar', 'hidden');
    load();

    fromAnotherTab('sidebar', 'open');

    expect(shown()).toBe('open');
  });

  it('should ignore changes to other keys', () => {
    localStorage.setItem('sidebar', 'hidden');
    load();

    fromAnotherTab('theme', 'light');

    expect(shown()).toBe('hidden');
  });
});

describe('LayoutWrapper: no animation on load (R6)', () => {
  it('should enable sidebar transitions only after the first frame', async () => {
    load();

    expect(html.hasAttribute('data-sidebar-ready')).toBe(false);

    await act(() => new Promise((resolve) => requestAnimationFrame(() => resolve(undefined))));

    expect(html.hasAttribute('data-sidebar-ready')).toBe(true);
  });
});
