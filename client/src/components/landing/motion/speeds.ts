/**
 * Depth layers for the tour, as apparent scroll speed relative to the page.
 * `1` is natural document flow. Named here so the three layers stay a
 * deliberate scheme rather than a magic number repeated per section.
 */

/** Ambient hearth light — the slowest thing on the page. */
export const GLOW_SPEED = 0.2;

/** Mocked art: drifts just enough to sit behind the words. */
export const ART_SPEED = 0.6;

/* Copy is never wrapped in a layer at all — it rides at 1.0, so text never
   moves relative to the reader. That is the rule the whole motion budget is
   built to protect. */
