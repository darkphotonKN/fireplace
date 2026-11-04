/**
 * Depth layers for the tour, as apparent scroll speed relative to the page.
 * `1` is natural document flow. Two layers survive FS-XCTMG: the glow behind
 * the hero and the close, and the copy. Tiles are hover targets and never
 * drift (R21), so the old 0.6x art layer is gone.
 */

/** Ambient hearth light — the slowest thing on the page. */
export const GLOW_SPEED = 0.2;

/* Copy is never wrapped in a layer at all — it rides at 1.0, so text never
   moves relative to the reader. That is the rule the whole motion budget is
   built to protect. */
