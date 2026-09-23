import type { ChecklistItem } from '@/services/api';

export type MoveDirection = 'up' | 'down';

/** Top-level items share the one null parent; undefined and null mean the same. */
const parentOf = (item: ChecklistItem): string | null => item.parentId ?? null;

/**
 * `order`'s members re-seated, in the order given, into the slots those
 * members already occupy in `all`. Non-members keep their exact positions.
 *
 * The one ordering primitive here, because both callers need the same thing:
 * the set being moved is scattered through a flat array — children are not
 * adjacent to their parent, and a filter's hidden siblings sit between the
 * visible ones — so a move is a refill of the set's own slots, never a splice
 * of a contiguous run.
 */
function reseat<T>(
  all: readonly T[],
  keyOf: (x: T) => string,
  order: readonly string[]
): T[] {
  const members = new Set(order);
  const byKey = new Map(all.map((x) => [keyOf(x), x]));
  const seated = order
    .map((k) => byKey.get(k))
    .filter((x): x is T => x !== undefined);
  let next = 0;
  return all.map((x) => (members.has(keyOf(x)) ? seated[next++] : x));
}

/**
 * Every item sharing `id`'s parent, in their current order.
 *
 * Read from the WHOLE item set the client holds, never from what a view is
 * showing: a type filter hides siblings without removing them from the set,
 * and an order computed over only the visible ones scrambles the rest (R7.2).
 */
export function siblingsOf(
  items: ChecklistItem[],
  id: string
): ChecklistItem[] {
  const self = items.find((i) => i.id === id);
  if (!self) return [];
  return items.filter((i) => parentOf(i) === parentOf(self));
}

/**
 * The WHOLE sibling set's ids after moving `id` one place in `direction`, or
 * null when the move cannot happen — an end of the set, or a set of one. A
 * null means nothing is sent: there is no order to write but the current one.
 *
 * `visible` is the ids a view is currently showing; omitted, every sibling
 * counts. The step is taken among the visible siblings, because a step taken
 * in the full set can swap the item with a hidden one and change nothing on
 * screen — a control that does nothing when clicked. The ids that come back
 * are still the complete set: hidden siblings keep the slots they held, so
 * their relative order is untouched (R7.2).
 */
export function moveOneStep(
  items: ChecklistItem[],
  id: string,
  direction: MoveDirection,
  visible?: ReadonlySet<string>
): string[] | null {
  const siblingIds = siblingsOf(items, id).map((s) => s.id);
  const onScreen = visible
    ? siblingIds.filter((s) => visible.has(s))
    : siblingIds;

  const from = onScreen.indexOf(id);
  if (from === -1) return null;
  const to = direction === 'down' ? from + 1 : from - 1;
  if (to < 0 || to >= onScreen.length) return null;

  const stepped = [...onScreen];
  [stepped[from], stepped[to]] = [stepped[to], stepped[from]];
  return reseat(siblingIds, (s) => s, stepped);
}

/**
 * The items with `ids`' members re-seated, in the order given, into the slots
 * those members already occupy. Everything else keeps its place.
 *
 * This is the optimistic render (R9.1). The client orders by array position —
 * `sequence` is never read here — and the server returns one flat sequence in
 * which a parent's children need not be adjacent, so a move is expressed by
 * refilling the set's own slots rather than by splicing a contiguous run.
 */
export function applyOrder(
  items: ChecklistItem[],
  ids: readonly string[]
): ChecklistItem[] {
  return reseat(items, (i) => i.id, ids);
}

/** Whether the panel should offer this move at all (R6.2, §Edge States). */
export function canMove(
  items: ChecklistItem[],
  id: string,
  direction: MoveDirection,
  visible?: ReadonlySet<string>
): boolean {
  return moveOneStep(items, id, direction, visible) !== null;
}

/**
 * The WHOLE sibling set's ids after dragging `id` onto `overId`'s place, or
 * null when there is nothing to write.
 *
 * Null covers the two refusals a drag has to make, and they are refusals for
 * the same reason — there is no new order — not errors:
 *   - `overId` is not one of `id`'s siblings: a drag never re-parents, so the
 *     item goes back where it started and nothing is sent (R5.1).
 *   - `overId` is `id`: the drop is where the drag began (R3.4).
 *
 * `visible` is the ids a view is showing. The landing place is read among
 * those, because that is what the gesture aimed at: the item lands where the
 * row it was dropped on was, and the siblings a filter hides keep the slots
 * they already held (R7.3, R7.2). The ids returned are still the complete
 * set, hidden members included.
 */
export function moveOnto(
  items: ChecklistItem[],
  id: string,
  overId: string,
  visible?: ReadonlySet<string>
): string[] | null {
  if (id === overId) return null;

  const siblingIds = siblingsOf(items, id).map((s) => s.id);
  if (!siblingIds.includes(overId)) return null;

  const onScreen = visible
    ? siblingIds.filter((s) => visible.has(s))
    : siblingIds;

  const from = onScreen.indexOf(id);
  const to = onScreen.indexOf(overId);
  if (from === -1 || to === -1) return null;

  const moved = [...onScreen];
  moved.splice(to, 0, ...moved.splice(from, 1));
  return reseat(siblingIds, (s) => s, moved);
}
