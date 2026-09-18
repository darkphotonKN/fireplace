// Which Parent Items are collapsed, remembered on this device only (FS-0007
// R6). Daily and long-term lists of a plan keep separate state.
//
// Every storage touch is guarded: private mode, blocked site data or a full
// quota must degrade to "everything expanded, nothing remembered", never an
// error. Some browsers throw on the window.localStorage getter itself.

type Scope = 'daily' | 'longterm' | 'archived';

const keyFor = (planId: string, scope: Scope) =>
  `collapsedGroups:${planId}:${scope}`;

export function loadCollapsedIds(planId: string, scope: Scope): Set<string> {
  try {
    const raw = window.localStorage.getItem(keyFor(planId, scope));
    if (!raw) return new Set();
    const parsed: unknown = JSON.parse(raw);
    // Anything but a list of ids is treated as nothing collapsed; the next
    // save overwrites it.
    return Array.isArray(parsed) && parsed.every((id) => typeof id === 'string')
      ? new Set(parsed)
      : new Set();
  } catch {
    return new Set();
  }
}

export function saveCollapsedIds(
  planId: string,
  scope: Scope,
  ids: ReadonlySet<string>,
  liveParentIds: ReadonlySet<string>
): void {
  // A parent that lost its last child has nothing to fold; drop it here.
  const kept = [...ids].filter((id) => liveParentIds.has(id));
  try {
    window.localStorage.setItem(keyFor(planId, scope), JSON.stringify(kept));
  } catch {
    // Not remembered this time; the in-memory state still works.
  }
}
