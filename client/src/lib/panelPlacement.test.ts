import { describe, it, expect } from 'vitest';
import { placePanel } from './panelPlacement';

const anchor = (left: number, right: number, top = 100) => ({ left, right, top });
const size = { width: 200, height: 300 };
const viewport = { width: 1000, height: 800 };

describe('placePanel', () => {
  it('should sit beside the anchor on the right, with a gap', () => {
    expect(placePanel(anchor(100, 400), size, viewport)).toEqual({
      top: 100,
      left: 408,
    });
  });

  it('should flip to the left when the right would run off the window', () => {
    // 900 + 8 + 200 is past the edge, so it goes to the anchor's other side.
    expect(placePanel(anchor(600, 900), size, viewport).left).toBe(392);
  });

  it('should stay inside the window when neither side has room', () => {
    // A wide anchor in a narrow window: pinned rather than half off-screen.
    const narrow = { width: 360, height: 800 };
    const place = placePanel(anchor(20, 340), size, narrow);
    expect(place.left).toBeGreaterThanOrEqual(8);
    expect(place.left + size.width).toBeLessThanOrEqual(narrow.width - 8);
  });

  it('should lift a panel that would hang off the bottom', () => {
    // Anchored at 700 with 300 of panel: 800 - 300 - 8 is as low as it goes.
    expect(placePanel(anchor(100, 400, 700), size, viewport).top).toBe(492);
  });

  it('should not push a panel taller than the window off the top', () => {
    const tall = { width: 200, height: 900 };
    expect(placePanel(anchor(100, 400, 700), tall, viewport).top).toBe(8);
  });

  it('should leave a panel alone when it already fits where it is', () => {
    expect(placePanel(anchor(100, 400, 200), size, viewport)).toEqual({
      top: 200,
      left: 408,
    });
  });
});
