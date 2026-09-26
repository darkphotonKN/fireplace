import { toDateOnly } from './itemDates';

/**
 * The two day-stamps the day gate reads (FS-KSJFR R11–R15, D3).
 *
 * `touched today` is whether the user has MUTATED a plan or a checklist item
 * during the browser-local day; the gate opens `/` on the focus prompt only
 * when nothing has been. It is written from the API layer's mutating functions
 * (R12) — the one place every mutation already passes through — and never from
 * a component, so a mutation added later cannot quietly stop counting.
 *
 * The gate's own "already fired today" stamp lives here too. Nothing reads it
 * until the gate itself ships, but one module owning both keys beats two
 * modules each owning half of the same storage convention.
 *
 * DAYS ARE LOCAL. The stamp is `toDateOnly` of now, the same "YYYY-MM-DD"
 * produced from local parts that checklist dates use — see the header of
 * ./itemDates for why a date-only value is never built through
 * `new Date(string)` or `toISOString()`. A UTC stamp would turn the day over in
 * the middle of the user's evening (or their morning, east of Greenwich).
 *
 * STORAGE IS BEST-EFFORT (R15). Private mode, blocked site data and a full
 * quota all reach us as a throw — from the accessor as often as from the method
 * — and this module is read synchronously before paint. So every touch is
 * guarded, and absent, malformed and unreadable all read the same: `false`.
 * The gate then errs toward showing the prompt, which costs one skip click,
 * rather than toward an error in a render path.
 */

const TOUCHED_KEY = 'touchedToday';
const GATE_KEY = 'gateFiredToday';

const localDay = (): string => toDateOnly(new Date())!;

function stampedToday(key: string): boolean {
  try {
    // Anything that isn't today's local day — absent, stale, or garbage a
    // different version of this code left behind — reads as "no".
    return window.localStorage.getItem(key) === localDay();
  } catch {
    return false;
  }
}

function stampToday(key: string): void {
  try {
    window.localStorage.setItem(key, localDay());
  } catch {
    // Not remembered on this device; the gate shows the prompt one extra time.
  }
}

/** Records that the user mutated something. Call only after a mutation succeeds. */
export function markTouchedToday(): void {
  stampToday(TOUCHED_KEY);
}

/** True only while the stamp names the current browser-local day. */
export function wasTouchedToday(): boolean {
  return stampedToday(TOUCHED_KEY);
}

/** Records that the gate has had its turn today — fired counts, acted-on doesn't. */
export function markGateFiredToday(): void {
  stampToday(GATE_KEY);
}

export function hasGateFiredToday(): boolean {
  return stampedToday(GATE_KEY);
}
