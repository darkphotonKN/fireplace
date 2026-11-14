/**
 * Shared constants for the app shell's chrome: the top bar, the docked plans
 * panel and the narrow-width drawer.
 */

/** The coral focus ring every shell control shares (FS-S6WVY R30). */
export const FOCUS_RING = 'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring';

/**
 * Tailwind's `md`: from here up the plans panel docks (FS-S6WVY); below it the
 * drawer (FS-49CQ2). The CSS side is `not all and (min-width: 768px)`.
 */
export const DOCKED_QUERY = '(min-width: 768px)';
