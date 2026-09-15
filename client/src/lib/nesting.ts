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
