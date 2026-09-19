/**
 * Where a floating panel goes next to the thing it belongs to.
 *
 * Pure, so the rules can be checked without a browser: the maths is the part
 * that has rules, and the part a test can hold on to.
 */

export interface PanelRect {
  top: number;
  left: number;
  right: number;
}

export interface PanelSize {
  width: number;
  height: number;
}

export interface PanelViewport {
  width: number;
  height: number;
}

export interface PanelPlacement {
  top: number;
  left: number;
}

/** Air between the panel and both the anchor and the window's edges. */
export const PANEL_GAP = 8;

export function placePanel(
  anchor: PanelRect,
  size: PanelSize,
  viewport: PanelViewport
): PanelPlacement {
  // Beside it on the right by default; flipped to the left when the right
  // would run off; pinned inside the window when neither side fits, which is
  // what a narrow window comes to.
  let left = anchor.right + PANEL_GAP;
  if (left + size.width > viewport.width - PANEL_GAP) {
    left = anchor.left - size.width - PANEL_GAP;
  }
  left = Math.max(
    PANEL_GAP,
    Math.min(left, viewport.width - size.width - PANEL_GAP)
  );

  // Top-aligned with the anchor, lifted just enough to keep the whole panel
  // on screen. A panel taller than the window sits at the top rather than
  // being pushed off it.
  const top = Math.max(
    PANEL_GAP,
    Math.min(anchor.top, viewport.height - size.height - PANEL_GAP)
  );

  return { top, left };
}
