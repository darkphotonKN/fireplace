import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { mockIntersectionObserver } from '@/test/motion';
import BentoTile, { REVEAL_MS } from './BentoTile';

/**
 * `@/test/motion`'s stub answers only width and reduced motion; the tile also
 * asks `(hover: none)`. A local stub, with a switch for the mid-visit toggle.
 */
function mockQueries(initial: { noHover?: boolean; reducedMotion?: boolean }) {
  const state = { noHover: false, reducedMotion: false, ...initial };
  const listeners = new Set<() => void>();
  const answer = (query: string) =>
    query.includes('hover: none')
      ? state.noHover
      : query.includes('prefers-reduced-motion')
        ? state.reducedMotion
        : false;

  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    get matches() {
      return answer(query);
    },
    media: query,
    onchange: null,
    addEventListener: (_: string, cb: () => void) => listeners.add(cb),
    removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
    addListener: (cb: () => void) => listeners.add(cb),
    removeListener: (cb: () => void) => listeners.delete(cb),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia;

  return {
    set(next: Partial<typeof state>) {
      Object.assign(state, next);
      act(() => listeners.forEach((cb) => cb()));
    },
  };
}

const VIEWPORT_HEIGHT = 800;
const TILE_HEIGHT = 200;

/** The top and bottom of a px `rootMargin`, read as CSS shorthand. */
function parseRootMargin(rootMargin = '0px') {
  const px = rootMargin.trim().split(/\s+/).map((part) => {
    if (!/^-?\d+(\.\d+)?px$|^0$/.test(part)) throw new Error(`model only speaks px: ${part}`);
    return parseFloat(part);
  });
  const [top, , bottom = top] = px;
  return { top, bottom };
}

/**
 * An `IntersectionObserver` that sends only what a browser sends, for the
 * default threshold of 0: one entry when observing starts, then one each time
 * `isIntersecting` flips. Nothing in between, so a jump from below the fold to
 * above the viewport is silent unless the observer's own margins make it a flip.
 * The tile sits at `top` in viewport coordinates; `place` moves it there in one
 * frame, the way a fling or a jump-scroll does.
 */
function mockViewport({ top: initialTop }: { top: number }) {
  let top = initialTop;
  const observed: Element[] = [];
  const live = new Set<{ deliver: (force: boolean) => void }>();

  class BrowserIntersectionObserver {
    private last: boolean | undefined;
    private readonly margin: { top: number; bottom: number };
    private target: Element | undefined;
    constructor(
      private readonly cb: IntersectionObserverCallback,
      opts: IntersectionObserverInit = {}
    ) {
      this.margin = parseRootMargin(opts.rootMargin);
    }
    deliver(force: boolean) {
      if (!this.target) return;
      const bottom = top + TILE_HEIGHT;
      const isIntersecting = bottom >= -this.margin.top && top <= VIEWPORT_HEIGHT + this.margin.bottom;
      if (!force && isIntersecting === this.last) return;
      this.last = isIntersecting;
      const entry = { isIntersecting, target: this.target, boundingClientRect: { top, bottom } };
      this.cb([entry as unknown as IntersectionObserverEntry], this as unknown as IntersectionObserver);
    }
    observe(el: Element) {
      observed.push(el);
      this.target = el;
      live.add(this);
      this.deliver(true);
    }
    unobserve() {
      live.delete(this);
    }
    disconnect() {
      live.delete(this);
    }
    takeRecords = () => [];
  }

  vi.stubGlobal('IntersectionObserver', BrowserIntersectionObserver);

  return {
    observed,
    place(next: number) {
      top = next;
      act(() => live.forEach((o) => o.deliver(false)));
    },
  };
}

/** In view, comfortably. */
const IN_VIEW = 300;
/** Below the fold: nothing of it is on screen. */
const BELOW = 2000;
/** Wholly above the viewport, where a fling to the bottom leaves it. */
const ABOVE = -1500;

function renderTile() {
  render(
    <BentoTile label="Plan" mock={<div data-landing-mock aria-hidden="true">mock</div>} />
  );
  return screen.getByRole('link');
}

/** Every state the tile passed through, in order, without repeats. */
function recordStates(el: HTMLElement) {
  const seen: string[] = [el.dataset.reveal ?? ''];
  const observer = new MutationObserver(() => {
    const now = el.dataset.reveal ?? '';
    if (seen[seen.length - 1] !== now) seen.push(now);
  });
  observer.observe(el, { attributes: true, attributeFilter: ['data-reveal'] });
  return seen;
}

describe('BentoTile', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should play once on first intersection on a no-hover device, then rest at end', () => {
    mockQueries({ noHover: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    expect(tile).toHaveAttribute('data-reveal', 'start');

    viewport.place(IN_VIEW);
    expect(tile).toHaveAttribute('data-reveal', 'playing');

    act(() => vi.advanceTimersByTime(REVEAL_MS));
    expect(tile).toHaveAttribute('data-reveal', 'end');

    // Scrolling away and back never replays it.
    viewport.place(BELOW);
    viewport.place(IN_VIEW);
    expect(tile).toHaveAttribute('data-reveal', 'end');
  });

  it('should rest at end, never playing, when it mounts already above the viewport', () => {
    mockQueries({ noHover: true });
    mockViewport({ top: ABOVE });
    const tile = renderTile();

    expect(tile).toHaveAttribute('data-reveal', 'end');
  });

  it('should land at end when flung past without an intersecting frame', () => {
    mockQueries({ noHover: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    const seen = recordStates(tile);
    expect(tile).toHaveAttribute('data-reveal', 'start');

    // One frame below the fold, the next wholly above it.
    viewport.place(ABOVE);

    expect(tile).toHaveAttribute('data-reveal', 'end');
    expect(seen).not.toContain('playing');
  });

  it('should not watch the viewport on a hover-capable device', () => {
    mockQueries({ noHover: false });
    const observer = mockIntersectionObserver();
    const tile = renderTile();

    // Hover is CSS-only there; JS never moves the state.
    expect(observer.observed).toHaveLength(0);
    expect(tile).toHaveAttribute('data-reveal', 'start');
  });

  it('should render end, never playing, under reduced motion', () => {
    mockQueries({ noHover: true, reducedMotion: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    const seen = recordStates(tile);

    viewport.place(IN_VIEW);
    act(() => vi.advanceTimersByTime(REVEAL_MS));

    expect(tile).toHaveAttribute('data-reveal', 'end');
    expect(seen).not.toContain('playing');
  });

  it('should snap to end when reduced motion is switched on mid-reveal', () => {
    const queries = mockQueries({ noHover: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    viewport.place(IN_VIEW);
    expect(tile).toHaveAttribute('data-reveal', 'playing');

    queries.set({ reducedMotion: true });

    expect(tile).toHaveAttribute('data-reveal', 'end');
  });

  it('should return a hover-capable tile to start when reduced motion is switched off', () => {
    const queries = mockQueries({ noHover: false, reducedMotion: true });
    mockViewport({ top: IN_VIEW });
    const tile = renderTile();
    expect(tile).toHaveAttribute('data-reveal', 'end');

    queries.set({ reducedMotion: false });

    // Its reveal is CSS on hover again, which a held "end" would pin open.
    expect(tile).toHaveAttribute('data-reveal', 'start');
  });

  it('should settle at end when hover appears mid-reveal', () => {
    const queries = mockQueries({ noHover: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    viewport.place(IN_VIEW);
    expect(tile).toHaveAttribute('data-reveal', 'playing');

    queries.set({ noHover: false });
    act(() => vi.advanceTimersByTime(REVEAL_MS));

    expect(tile).toHaveAttribute('data-reveal', 'end');
  });

  it('should keep a played tile at end after reduced motion goes on and off again', () => {
    const queries = mockQueries({ noHover: true });
    const viewport = mockViewport({ top: BELOW });
    const tile = renderTile();
    viewport.place(IN_VIEW);
    act(() => vi.advanceTimersByTime(REVEAL_MS));

    queries.set({ reducedMotion: true });
    queries.set({ reducedMotion: false });

    expect(tile).toHaveAttribute('data-reveal', 'end');
  });

  it('should start at start when no effects have run', async () => {
    const { renderToString } = await import('react-dom/server');
    const html = renderToString(<BentoTile label="Plan" mock={<p>Database 1/5</p>} />);

    expect(html).toContain('data-reveal="start"');
    expect(html).toContain('Database 1/5');
  });

  it('should be one link to sign-up named by its label and "Try it"', () => {
    mockQueries({});
    mockIntersectionObserver();
    render(<BentoTile label="Plan" mock={<div data-landing-mock aria-hidden="true" />} />);

    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(1);
    expect(links[0]).toHaveAttribute('href', '/auth?tab=signup');
    expect(links[0]).toHaveAccessibleName('Plan Try it');
  });

  it('should put the label bottom-left and "Try it →" bottom-right, with the arrow hidden', () => {
    mockQueries({});
    mockIntersectionObserver();
    renderTile();

    const footer = screen.getByText('Plan').parentElement!;
    expect(footer).toHaveClass('justify-between');
    expect(footer.firstElementChild).toHaveTextContent('Plan');
    expect(footer.lastElementChild).toHaveTextContent('Try it');
    expect(screen.getByText('→')).toHaveAttribute('aria-hidden', 'true');
  });

  it('should wear the tile tokens', () => {
    mockQueries({});
    mockIntersectionObserver();
    const tile = renderTile();

    expect(tile).toHaveClass('rounded-tile', 'bg-card', 'border', 'border-border', 'shadow-tile');
  });

  it('should invert to the secondary pair when asked', () => {
    mockQueries({});
    mockIntersectionObserver();
    render(<BentoTile label="Today" inverted mock={<div />} />);
    const tile = screen.getByRole('link');

    expect(tile).toHaveClass('bg-secondary', 'text-secondary-foreground');
    expect(tile).not.toHaveClass('bg-card');
  });

  it.each([
    ['md', ['md:col-span-2', 'lg:col-span-1']],
    ['lg', ['lg:col-span-2']],
  ] as const)('should span two columns at %s only', (span, classes) => {
    mockQueries({});
    mockIntersectionObserver();
    render(<BentoTile label="Calendar" span={span} mock={<div />} />);

    expect(screen.getByRole('link')).toHaveClass(...classes);
  });
});
