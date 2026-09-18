// Page maths for the checklist (FS-0008 R2–R4, R10). The page unit is the
// top-level group, never the row, so a parent and its children always land on
// the same page whatever their number — and a collapsed parent still counts as
// one. Pure: pass the already-filtered groups, get a slice back.

import type { ChecklistGroup } from './nesting';

/** Top-level items per page (R3). */
export const PAGE_SIZE = 10;

/** How many pages the set needs. Always at least one, so an empty list is
 *  page 1 of 1 rather than page 1 of 0. */
export function pageCount(total: number, size: number = PAGE_SIZE): number {
  return Math.max(1, Math.ceil(total / size));
}

/** A page number brought inside the set's range: below one floors to the
 *  first page, past the end clamps to the last (R10). */
export function clampPage(
  page: number,
  total: number,
  size: number = PAGE_SIZE
): number {
  return Math.min(Math.max(page, 1), pageCount(total, size));
}

/** The groups on a 1-based page, whole. A page past the end gives nothing;
 *  callers clamp first rather than relying on that. */
export function pageSlice<T>(
  groups: readonly T[],
  page: number,
  size: number = PAGE_SIZE
): T[] {
  const start = (page - 1) * size;
  return groups.slice(start, start + size);
}

/** The 1-based page holding an item, whether it heads a group or sits inside
 *  one, or null when the id isn't in the set at all. */
export function pageOfItem(
  groups: readonly ChecklistGroup[],
  id: string,
  size: number = PAGE_SIZE
): number | null {
  const index = groups.findIndex(
    ({ item, children }) =>
      item.id === id || children.some((child) => child.id === id)
  );
  return index === -1 ? null : Math.floor(index / size) + 1;
}
