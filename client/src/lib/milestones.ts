/**
 * The milestone reading of a plan (FS-FSFEC, ADR-0014): which of its items are
 * milestones, how far along each one is, and how many are closed.
 *
 * The single derivation the plan header and the dashboard both read, so the
 * surface stays small: items in, milestones and a summary out. Pass the whole
 * list, not a page or a filtered set; what the screen happens to show has no
 * say in what the plan is made of (R16).
 */

/**
 * The fields the reading needs. Structural, so both the page's item type and
 * the generated API type fit without a mapping step.
 */
export interface MilestoneSourceItem {
  id: string;
  description: string;
  done: boolean;
  scope?: string;
  type?: string;
  parentId?: string | null;
  archived?: boolean;
}

export interface Milestone {
  id: string;
  title: string;
  /** The item's `done`: a person closed it. */
  closed: boolean;
  /** Done and total task steps, archived steps left out (R6). */
  steps: { done: number; total: number };
  /** 0..1, what the strip fills its segment to (R6–R8). */
  progress: number;
}

export interface MilestoneReading {
  milestones: Milestone[];
  /** N in "N of M milestones". */
  closed: number;
  /** M in "N of M milestones". */
  total: number;
}

const isTask = (item: MilestoneSourceItem) => (item.type ?? 'task') === 'task';

// Archived work is out of the plan (C9), whatever level it sits at.
const isLive = (item: MilestoneSourceItem) => !item.archived;

export function readMilestones(
  items: readonly MilestoneSourceItem[]
): MilestoneReading {
  const milestones = items
    .filter(
      (item) =>
        isLive(item) &&
        isTask(item) &&
        !item.parentId &&
        item.scope === 'longterm'
    )
    .map((item) => {
      const steps = stepsOf(items, item.id);
      return {
        id: item.id,
        title: item.description,
        closed: item.done,
        steps,
        progress: progressOf(item.done, steps),
      };
    });
  return {
    milestones,
    closed: milestones.filter((m) => m.closed).length,
    total: milestones.length,
  };
}

/**
 * Every task step done and the milestone still open: the state that offers to
 * close it (R10). Only an offer; `done` is set by a person (R13). A milestone
 * with no task steps has nothing to complete and keeps its manual tick (R7).
 */
export function isComplete(milestone: Milestone): boolean {
  const { done, total } = milestone.steps;
  return !milestone.closed && total > 0 && done === total;
}

/** Task steps only: note children are shown but never counted (R1, R6). */
function stepsOf(items: readonly MilestoneSourceItem[], parentId: string) {
  const steps = items.filter(
    (c) => c.parentId === parentId && isLive(c) && isTask(c)
  );
  return { done: steps.filter((c) => c.done).length, total: steps.length };
}

// A closed milestone is full whatever its steps say (R8); one with no task
// steps has only its own tick to go on (R7).
function progressOf(closed: boolean, steps: { done: number; total: number }) {
  if (closed) return 1;
  return steps.total === 0 ? 0 : steps.done / steps.total;
}
