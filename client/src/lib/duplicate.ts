/**
 * Puts what a duplicate created into a view's flat item list (FS-F6AEE R16):
 * the copy, then its copied steps in the order the server returned them,
 * directly after the source.
 *
 * Position, not `sequence`, is what places it. The server re-sequences the
 * source's whole sibling set but only returns the new items, so every other
 * item's `sequence` is stale from here on; the views read order from the
 * array (a family is its head's position plus its children in array order),
 * so splicing right after the source is enough. For a top-level source that
 * puts the copy after the source and before the next top-level item, whatever
 * position the source's own steps hold; for a step it puts the copy right
 * after it among its parent's steps.
 *
 * A source no longer in the list still gets its copy shown, at the end: the
 * server has it either way.
 */
export function insertDuplicate<T extends { id: string }>(
  items: T[],
  sourceId: string,
  created: T[]
): T[] {
  const at = items.findIndex((t) => t.id === sourceId);
  if (at === -1) return [...items, ...created];
  return [...items.slice(0, at + 1), ...created, ...items.slice(at + 1)];
}
