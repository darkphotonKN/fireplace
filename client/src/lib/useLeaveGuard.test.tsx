import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import Link from 'next/link';
import { createRoot } from 'react-dom/client';

/**
 * The note page's leave guard (FS-PGG2N R27, R32): in-app links are held
 * while the page decides, and nothing else is touched.
 */

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, replace: vi.fn(), refresh: vi.fn() }),
}));

import { useLeaveGuard } from './useLeaveGuard';

let decide: ReturnType<typeof vi.fn>;
// What reached the link's own click handler, i.e. what was not held.
const reached = vi.fn();

function Guarded() {
  useLeaveGuard(decide as unknown as (href: string) => Promise<boolean>);
  return (
    <div>
      <Link href="/plan/p1" onClick={(e) => { e.preventDefault(); reached('/plan/p1'); }}>
        Back to plan
      </Link>
      <a href="https://go.dev/ref/mem" target="_blank" rel="noopener noreferrer" onClick={(e) => { e.preventDefault(); reached('external'); }}>
        external
      </a>
      <Link href="/plans" target="_blank" onClick={(e) => { e.preventDefault(); reached('new tab'); }}>
        new tab
      </Link>
      <a href="#here" onClick={(e) => { e.preventDefault(); reached('hash'); }}>
        hash
      </a>
      <button type="button" onClick={() => reached('button')}>
        not a link
      </button>
    </div>
  );
}

beforeEach(() => {
  push.mockReset();
  reached.mockReset();
  decide = vi.fn();
});

describe('useLeaveGuard', () => {
  it('should hold a same-origin link until the guard resolves, then go there', async () => {
    let answer!: (go: boolean) => void;
    decide.mockReturnValue(new Promise<boolean>((res) => (answer = res)));
    render(<Guarded />);
    const click = fireEvent.click(screen.getByText('Back to plan'));
    expect(click).toBe(false); // default prevented: the browser does not follow it
    expect(reached).not.toHaveBeenCalled(); // nor does Next's Link
    expect(decide).toHaveBeenCalledWith('/plan/p1');
    expect(push).not.toHaveBeenCalled();
    await act(async () => answer(true));
    expect(push).toHaveBeenCalledWith('/plan/p1');
  });

  it('should drop the navigation when the guard says stay', async () => {
    decide.mockResolvedValue(false);
    render(<Guarded />);
    await act(async () => fireEvent.click(screen.getByText('Back to plan')));
    expect(push).not.toHaveBeenCalled();
  });

  it('should swallow a second click while the first is still being decided', async () => {
    let answer!: (go: boolean) => void;
    decide.mockReturnValue(new Promise<boolean>((res) => (answer = res)));
    render(<Guarded />);
    fireEvent.click(screen.getByText('Back to plan'));
    fireEvent.click(screen.getByText('Back to plan'));
    expect(decide).toHaveBeenCalledTimes(1);
    await act(async () => answer(true));
    expect(push).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['an external link', 'external', {}],
    ['a link that opens a new tab', 'new tab', {}],
    ['a hash on this page', 'hash', {}],
    ['a modified click (new tab)', 'Back to plan', { metaKey: true }],
    ['a ctrl-click', 'Back to plan', { ctrlKey: true }],
    ['a middle click', 'Back to plan', { button: 1 }],
    ['a button', 'not a link', {}],
  ])('should leave %s alone', (_why, text, init) => {
    render(<Guarded />);
    fireEvent.click(screen.getByText(text), init);
    expect(decide).not.toHaveBeenCalled();
    const label = text === 'Back to plan' ? '/plan/p1' : text === 'not a link' ? 'button' : text;
    expect(reached).toHaveBeenCalledWith(label);
  });

  it('should hold a click that lands as soon as the page is on screen', async () => {
    // Rendered the way the app renders, outside act: React commits the page,
    // then runs its passive effects in a later task. A click in between must
    // already be held, or it goes past the guard unflushed and unasked.
    decide.mockResolvedValue(false);
    const env = globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean };
    const actEnv = env.IS_REACT_ACT_ENVIRONMENT;
    env.IS_REACT_ACT_ENVIRONMENT = false;
    const host = document.body.appendChild(document.createElement('div'));
    const root = createRoot(host);
    try {
      // Clicks from the first microtask after the link is in the document.
      const clicked = new Promise<void>((done) => {
        const seen = new MutationObserver(() => {
          const link = host.querySelector('a[href="/plan/p1"]');
          if (!link) return;
          seen.disconnect();
          link.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
          done();
        });
        seen.observe(host, { childList: true, subtree: true });
      });
      root.render(<Guarded />);
      await clicked;
      expect(reached).not.toHaveBeenCalled();
      expect(decide).toHaveBeenCalledWith('/plan/p1');
    } finally {
      root.unmount();
      host.remove();
      env.IS_REACT_ACT_ENVIRONMENT = actEnv;
    }
  });

  it('should stop listening once unmounted', () => {
    const view = render(<Guarded />);
    const link = screen.getByText('Back to plan');
    view.unmount();
    document.body.appendChild(link);
    // React's handler went with the unmount; keep jsdom from following it.
    link.addEventListener('click', (e) => e.preventDefault());
    fireEvent.click(link);
    expect(decide).not.toHaveBeenCalled();
    link.remove();
  });

  describe('what it leaves to the browser (R32)', () => {
    const add = vi.spyOn(window, 'addEventListener');
    afterEach(() => add.mockClear());

    it('should add no beforeunload, popstate or other unload listener', () => {
      render(<Guarded />);
      const kinds = add.mock.calls.map((c) => c[0]);
      expect(kinds).not.toContain('beforeunload');
      expect(kinds).not.toContain('unload');
      expect(kinds).not.toContain('popstate');
      expect(kinds).not.toContain('pagehide');
    });
  });
});
