/**
 * A day plan, resolved against the plan as it stands now (FS-02JBZ R66, R67;
 * DF4ZB R59 parity). Grown from Today's moves' resolver, which it replaces (A5).
 *
 * A day plan is drawn when the user presses Plan my day and kept for the day,
 * so by the time it is shown the plan may have moved on: an item archived or
 * deleted, a parent closed, an item finished somewhere else. Entries are
 * therefore never shown as stored; each one is read against the plan's live
 * long-term items on every render, and only what still makes sense is kept.
 *
 * The milestone an entry belongs to is its top-level ancestor (R66). The
 * server calls the anchor a *parent item*; this is where the client
 * translates.
 *
 * Kept free of React and of the API module: the section and its tests share
 * one set of rules.
 */
import type { MilestoneSourceItem } from './milestones';

/** The fields an entry needs. Structural, so the generated API type fits. */
export interface DayPlanEntrySource {
  id: string;
  kind: 'existing_item' | 'new_step';
  itemId?: string;
  parentItemId?: string;
  text: string;
  reason: string;
  minutes?: number;
  /** The day tick for the day plan's date. Absent reads as not ticked. */
  ticked?: boolean;
  /** The revision N the server stamped when the plan was drawn (R10, R45). */
  revision?: number;
  /** The item's "revise" switch: only on learning plans' existing items (R49). */
  revisionsEnabled?: boolean;
}

export interface DayPlanGroupSource {
  label: string;
  entries: readonly DayPlanEntrySource[];
}

export interface ResolvedEntry {
  id: string;
  kind: DayPlanEntrySource['kind'];
  /**
   * The top-level ancestor's title, or null when the entry's item is itself
   * top level: its text already names it.
   */
  milestoneTitle: string | null;
  /** The live item's description for an existing item; the proposal otherwise. */
  text: string;
  reason: string;
  minutes?: number;
  /** Present iff kind is `existing_item`. */
  itemId?: string;
  /** Present iff kind is `new_step`: where Add puts the new child. */
  parentItemId?: string;
  /** The item's own `done`, read live. Always false for a new step. */
  done: boolean;
  /** The day tick ("done for today"). Never the item's `done` (R28). */
  ticked: boolean;
  /**
   * Only on a `new_step` entry: the child already added from it, found by
   * `findAddedStep`. Present means the entry is tickable with this id (R76).
   */
  addedItemId?: string;
  /**
   * The revision this entry is, shown as "Revision N" (R81). Absent when not
   * stamped, and when revisions were switched off after the draw: that
   * cancelled the open revision (R47), so the stamp no longer names one.
   */
  revision?: number;
  /**
   * The item's revisions switch, present only where the server sends it:
   * learning plans' existing items (R49, R80). Absent means no switch.
   */
  revisionsEnabled?: boolean;
}

export interface ResolvedGroup {
  label: string;
  entries: ResolvedEntry[];
}

/**
 * The groups and entries still worth showing, in the order drawn. A group
 * whose entries are all hidden is dropped. Pass the plan's whole long-term
 * list, not a page or a filtered set.
 */
export function resolveDayPlan(
  groups: readonly DayPlanGroupSource[],
  items: readonly MilestoneSourceItem[]
): ResolvedGroup[] {
  const byId = new Map(items.map((i) => [i.id, i]));

  // Walks up parentId to the top. A missing link stops the walk where it
  // breaks; the seen set guards against a cycle in bad data.
  const topOf = (start: MilestoneSourceItem): MilestoneSourceItem => {
    let node = start;
    const seen = new Set([node.id]);
    while (node.parentId) {
      const parent = byId.get(node.parentId);
      if (!parent || seen.has(parent.id)) break;
      seen.add(parent.id);
      node = parent;
    }
    return node;
  };

  const resolveEntry = (entry: DayPlanEntrySource): ResolvedEntry | null => {
    const base = {
      id: entry.id,
      kind: entry.kind,
      reason: entry.reason,
      ...(entry.minutes !== undefined ? { minutes: entry.minutes } : {}),
      ticked: entry.ticked ?? false,
      ...(entry.revision !== undefined && entry.revisionsEnabled !== false
        ? { revision: entry.revision }
        : {}),
      ...(entry.revisionsEnabled !== undefined
        ? { revisionsEnabled: entry.revisionsEnabled }
        : {}),
    };

    if (entry.kind === 'existing_item') {
      const live = entry.itemId ? byId.get(entry.itemId) : undefined;
      if (!live || live.archived) return null;
      const top = topOf(live);
      return {
        ...base,
        milestoneTitle: top.id === live.id ? null : top.description,
        text: live.description,
        itemId: live.id,
        done: live.done,
      };
    }

    // A new step needs a parent that can still take a child.
    const parent = entry.parentItemId ? byId.get(entry.parentItemId) : undefined;
    if (!parent || parent.archived || parent.done) return null;
    const addedItemId = findAddedStep(parent.id, entry.text, items);
    return {
      ...base,
      milestoneTitle: topOf(parent).description,
      text: entry.text,
      parentItemId: parent.id,
      done: false,
      ...(addedItemId ? { addedItemId } : {}),
    };
  };

  const resolved: ResolvedGroup[] = [];
  for (const group of groups) {
    const entries = group.entries
      .map(resolveEntry)
      .filter((e): e is ResolvedEntry => e !== null);
    if (entries.length > 0) resolved.push({ label: group.label, entries });
  }
  return resolved;
}

/** Case-folded with runs of whitespace collapsed: "  Book a DEMO " = "book a demo". */
const normalize = (text: string): string =>
  text.trim().replace(/\s+/g, ' ').toLowerCase();

/**
 * The step a `new_step` entry was already added as, if any (R76): a live task
 * child of `parentId` whose description matches `text`, ignoring case and
 * whitespace. This is how an added entry is recognised after a reload, when
 * the id Add returned is gone. Only a live task qualifies, since that is what
 * the server accepts on a tick (R25). Edited text no longer matches, so the
 * entry offers Add again (accepted, rare).
 */
export function findAddedStep(
  parentId: string | undefined,
  text: string,
  items: readonly MilestoneSourceItem[]
): string | undefined {
  if (!parentId) return undefined;
  const wanted = normalize(text);
  return items.find(
    (i) =>
      i.parentId === parentId &&
      !i.archived &&
      (i.type ?? 'task') === 'task' &&
      normalize(i.description) === wanted
  )?.id;
}
