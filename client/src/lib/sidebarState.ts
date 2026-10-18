/**
 * The sidebar's remembered open/hidden choice (FS-S6WVY §Remembered state).
 *
 * One key, `sidebar`, holding `open` or `hidden`. A `<head>` script reads it
 * before first paint and mirrors it onto `<html data-sidebar>`, and CSS takes
 * the sidebar geometry from that attribute, so a reload paints the right state
 * on the first frame (R5). The attribute is then the page's live copy of the
 * state: React reads it, the toggle and other tabs update it.
 */

export type SidebarState = 'open' | 'hidden';

const SIDEBAR_KEY = 'sidebar';
/** Written for nearly everyone by the old home effect, so never read (R4). */
const LEGACY_SIDEBAR_KEY = 'sidebarCollapsed';
const SIDEBAR_ATTRIBUTE = 'data-sidebar';

/** Only an explicit `hidden` hides; anything else is "no preference" (R1, R2). */
export function parseSidebarState(raw: string | null): SidebarState {
  return raw === 'hidden' ? 'hidden' : 'open';
}

/**
 * Pre-paint apply, inlined into `<head>` beside the theme script. It runs
 * before any bundle, so it cannot import parseSidebarState; it restates the
 * same rule, and sidebarState.test.ts holds the two to the same cases.
 */
export const SIDEBAR_HEAD_SCRIPT = `(function () {
  var state = 'open';
  try {
    if (localStorage.getItem('${SIDEBAR_KEY}') === 'hidden') state = 'hidden';
  } catch (e) {}
  document.documentElement.setAttribute('${SIDEBAR_ATTRIBUTE}', state);
})();`;

/** Set once the first frame has painted; CSS animates the sidebar only after it (R6). */
const SIDEBAR_READY_ATTRIBUTE = 'data-sidebar-ready';

const listeners = new Set<() => void>();

/** The stored choice, read fresh; blocked storage reads as no preference (R8). */
function readStoredSidebarState(): SidebarState {
  try {
    return parseSidebarState(window.localStorage.getItem(SIDEBAR_KEY));
  } catch {
    return 'open';
  }
}

/**
 * The page's live state: what the head script painted and the toggle or
 * another tab has since updated. Falls back to storage only if nothing has
 * painted yet. Holding the state on the attribute, not in storage, is what
 * keeps the toggle working when storage is blocked.
 */
export function getSidebarState(): SidebarState {
  const painted = document.documentElement.getAttribute(SIDEBAR_ATTRIBUTE);
  return painted === null ? readStoredSidebarState() : parseSidebarState(painted);
}

/** The server has no storage; it renders open and the attribute corrects the paint. */
export function getServerSidebarState(): SidebarState {
  return 'open';
}

/** Show a state in this page. Never writes storage. */
function applySidebarState(state: SidebarState) {
  document.documentElement.setAttribute(SIDEBAR_ATTRIBUTE, state);
  listeners.forEach((notify) => notify());
}

/**
 * The pin/hide toggle's write, and the only write of the choice anywhere
 * (R3). Drops the legacy key on the way (R4).
 */
export function chooseSidebarState(state: SidebarState) {
  applySidebarState(state);
  try {
    window.localStorage.setItem(SIDEBAR_KEY, state);
    window.localStorage.removeItem(LEGACY_SIDEBAR_KEY);
  } catch {
    // Blocked: the choice holds for this page, it just isn't remembered (R8).
  }
}

/**
 * Another tab's write arrives here as a `storage` event and is applied, never
 * re-written (R7). One listener for the page, however many subscribers.
 */
function followOtherTabs(event: StorageEvent) {
  // A null key means another tab cleared storage: no preference, so open.
  if (event.key !== SIDEBAR_KEY && event.key !== null) return;
  applySidebarState(parseSidebarState(event.key === null ? null : event.newValue));
}

/**
 * useSyncExternalStore's subscribe. The first subscriber starts following
 * other tabs and the last one out stops.
 */
export function subscribeSidebarState(onChange: () => void) {
  listeners.add(onChange);
  if (listeners.size === 1) window.addEventListener('storage', followOtherTabs);
  return () => {
    listeners.delete(onChange);
    if (listeners.size === 0) window.removeEventListener('storage', followOtherTabs);
  };
}

export function markSidebarReady() {
  document.documentElement.setAttribute(SIDEBAR_READY_ATTRIBUTE, '');
}
