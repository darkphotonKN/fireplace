import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { mockMatchMedia } from '@/test/motion';
import ParallaxLayer from './ParallaxLayer';

function scroll(y: number) {
  window.scrollY = y;
  act(() => {
    window.dispatchEvent(new Event('scroll'));
  });
}

/** Scroll and let the queued animation frame actually run. */
async function scrollAndFlush(y: number) {
  window.scrollY = y;
  await act(async () => {
    window.dispatchEvent(new Event('scroll'));
    await new Promise((resolve) => setTimeout(resolve, 32));
  });
}

describe('ParallaxLayer', () => {
  beforeEach(() => {
    window.scrollY = 0;
  });

  it('should apply no transform below 640px', () => {
    mockMatchMedia({ wide: false });

    render(
      <ParallaxLayer speed={0.2} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );
    scroll(500);

    expect(screen.getByTestId('layer').style.transform).toBe('');
  });

  it('should apply a translate transform when wide and scrolled', () => {
    mockMatchMedia({ wide: true });

    render(
      <ParallaxLayer speed={0.2} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );
    scroll(500);

    expect(screen.getByTestId('layer').style.transform).toMatch(/translate3d\(0px, -?[\d.]+px, 0px\)/);
  });

  it('should apply no transform when the user prefers reduced motion', () => {
    mockMatchMedia({ wide: true, reducedMotion: true });

    render(
      <ParallaxLayer speed={0.2} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );
    scroll(500);

    expect(screen.getByTestId('layer').style.transform).toBe('');
  });

  it('should clamp the offset so a distant layer is never dragged into view', async () => {
    mockMatchMedia({ wide: true });

    render(
      <ParallaxLayer speed={0} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );
    await scrollAndFlush(100000);

    const offset = Number(
      /translate3d\(0px, (-?[\d.]+)px, 0px\)/.exec(screen.getByTestId('layer').style.transform)?.[1]
    );
    expect(Math.abs(offset)).toBeLessThanOrEqual(80);
  });

  it('should not read layout during scroll', async () => {
    mockMatchMedia({ wide: true });
    render(
      <ParallaxLayer speed={0.6} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );

    // Geometry is measured at mount; only what happens *after* matters here.
    const readLayout = vi.spyOn(Element.prototype, 'getBoundingClientRect');
    await scrollAndFlush(400);
    await scrollAndFlush(800);

    expect(readLayout).not.toHaveBeenCalled();
  });

  it('should stop writing transforms after unmount', async () => {
    mockMatchMedia({ wide: true });
    const { unmount } = render(
      <ParallaxLayer speed={0.6} data-testid="layer">
        <p>ember</p>
      </ParallaxLayer>
    );
    const el = screen.getByTestId('layer');

    unmount();
    await scrollAndFlush(900);

    expect(el.style.transform).toBe('');
  });
});
