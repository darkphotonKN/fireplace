import type { ChecklistItem } from '@/services/api';

/** A top-level row and the Child Items rendered beneath it (possibly none). */
export interface ChecklistGroup {
  item: ChecklistItem;
  children: ChecklistItem[];
}

/**
 * Group the rendered rows into top-level items, each with its children, both
 * in their existing order. Pass the already-filtered set: a child whose parent
 * isn't in it falls through as top-level with no children of its own, and a
 * parent whose children were all filtered out gets an empty `children`.
 */
export function groupChecklistItems(items: ChecklistItem[]): ChecklistGroup[] {
  const visibleIds = new Set(items.map((t) => t.id));
  return items
    .filter((t) => !t.parentId || !visibleIds.has(t.parentId))
    .map((item) => ({
      item,
      children: items.filter((t) => t.parentId === item.id),
    }));
}

/**
 * The parent the add bar nests under on Tab: walking up from the bottom of
 * the rendered groups, the nearest top-level task. Child rows resolve to
 * their group's parent, so only group heads are candidates.
 */
export function findAddBarTarget(groups: ChecklistGroup[]): ChecklistItem | null {
  for (let i = groups.length - 1; i >= 0; i--) {
    if (canParentAddBar(groups[i].item)) return groups[i].item;
  }
  return null;
}

/**
 * The add bar's current target by id, or null once it can no longer take
 * children: archived, deleted, converted to a note, outdented under another
 * row, or filtered out (R8.11). It need not be the last group.
 */
export function resolveAddBarTarget(
  groups: ChecklistGroup[],
  id: string | null
): ChecklistItem | null {
  if (!id) return null;
  const item = groups.find((g) => g.item.id === id)?.item;
  return item && canParentAddBar(item) ? item : null;
}

// Only a real top-level task can take children (R0.1–R0.3). A group head with
// a parentId is an orphan whose parent is filtered out, and optimistic rows
// from addInsightAsTodo carry temporary ids the server doesn't know yet.
function canParentAddBar(item: ChecklistItem): boolean {
  return (
    (item.type ?? 'task') === 'task' &&
    !item.parentId &&
    !item.id.startsWith('insight_') &&
    !item.id.startsWith('fallback_')
  );
}

/**
 * Rows in render order, each top-level row followed by its children, leaving
 * out the children of collapsed parents. Ids that aren't a rendered parent
 * (stale, or a parent with no visible children) change nothing.
 */
export function visibleRows(
  groups: ChecklistGroup[],
  collapsedIds: ReadonlySet<string>
): ChecklistItem[] {
  return groups.flatMap(({ item, children }) =>
    collapsedIds.has(item.id) ? [item] : [item, ...children]
  );
}

/**
 * The quiet count a collapsed parent shows after its text: done/total over its
 * task children only, or how many notes it holds when every child is a note.
 * Only meaningful for a parent with at least one child.
 */
export function collapsedCount(children: ChecklistItem[]): string {
  const tasks = children.filter((t) => (t.type ?? 'task') === 'task');
  if (tasks.length > 0) {
    return `${tasks.filter((t) => t.done).length}/${tasks.length}`;
  }
  return children.length === 1 ? '1 note' : `${children.length} notes`;
}
