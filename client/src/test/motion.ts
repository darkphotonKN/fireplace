import { vi } from 'vitest';

/**
 * Controllable `matchMedia` stub. Only the queries the app actually asks for
 * are answered (reduced motion, `min-width`, and `(hover: hover)` for the
 * sidebar peek); anything else reports `false`.
 */
export function mockMatchMedia(opts: { reducedMotion?: boolean; wide?: boolean; hover?: boolean }) {
  const { reducedMotion = false, wide = true, hover = false } = opts;
  const listeners = new Set<() => void>();

  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: query.includes('prefers-reduced-motion')
      ? reducedMotion
      : query.includes('hover: hover')
        ? hover
        : query.includes('min-width')
          ? wide
          : false,
    media: query,
    onchange: null,
    addEventListener: (_: string, cb: () => void) => listeners.add(cb),
    removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
    addListener: (cb: () => void) => listeners.add(cb),
    removeListener: (cb: () => void) => listeners.delete(cb),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia;

  return listeners;
}

type ObserverEntry = {
  isIntersecting: boolean;
  boundingClientRect: { top: number };
};

/**
 * Controllable `IntersectionObserver` stub. Returns a handle that lets a test
 * drive the callback for whatever element the component observed — which is how
 * the fling case (never intersecting, ends up above the viewport) gets tested.
 */
export function mockIntersectionObserver() {
  const observed: Element[] = [];
  let trigger: (entries: ObserverEntry[]) => void = () => {};
  const unobserve = vi.fn();

  class MockIntersectionObserver {
    constructor(cb: (entries: ObserverEntry[]) => void) {
      trigger = cb;
    }
    observe(el: Element) {
      observed.push(el);
    }
    unobserve = unobserve;
    disconnect = vi.fn();
    takeRecords = () => [];
    root = null;
    rootMargin = '';
    thresholds = [];
  }

  vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);

  return {
    observed,
    unobserve,
    /** Element scrolled into view. */
    enter: () => trigger([{ isIntersecting: true, boundingClientRect: { top: 100 } }]),
    /** Element left the viewport downward (scrolled back up past it). */
    exitBelow: () => trigger([{ isIntersecting: false, boundingClientRect: { top: 900 } }]),
    /**
     * Element ended up ABOVE the viewport without ever being reported as
     * intersecting — the fling-to-bottom case.
     */
    flungPast: () => trigger([{ isIntersecting: false, boundingClientRect: { top: -500 } }]),
  };
}

/**
 * Places every subsequently-rendered element at a given viewport-relative top.
 *
 * jsdom reports all-zero rects, which reads as "already on screen" — so without
 * this a test can never exercise the below-the-fold path at all.
 */
export function stubElementTop(top: number) {
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({
    top,
    bottom: top + 200,
    left: 0,
    right: 0,
    width: 0,
    height: 200,
    x: 0,
    y: top,
    toJSON: () => ({}),
  } as DOMRect);
}

/** A top far enough down that nothing considers it visible. */
export const BELOW_FOLD = 2000;

/** Remove `IntersectionObserver` entirely, as on an old browser. */
export function removeIntersectionObserver() {
  vi.stubGlobal('IntersectionObserver', undefined);
}
