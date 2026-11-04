/**
 * The tour's invented content, in one place.
 *
 * Shared so every tile tells **one** story rather than five unrelated
 * screenshots: the step the PLAN tile ticks is the same step the other tiles
 * count, so start-state counts agree wherever they appear (FS-XCTMG R14).
 *
 * Concrete enough to be recognisable, vague enough to age well. No personal
 * data beyond the persona's name, no long notes.
 */

/** The visitor's stand-in. A name only. */
export const PERSONA_NAME = 'Chris Anderson';

export const PLAN_TITLE = 'Learn microservices';

export interface MockStep {
  label: string;
  done: boolean;
}

export interface MockMilestone {
  title: string;
  steps: readonly MockStep[];
}

export const DATABASE_MILESTONE: MockMilestone = {
  title: 'Database',
  steps: [
    { label: 'SQL Tools Drill Basics', done: true },
    { label: 'SQL Tools Drill Practice', done: false },
    { label: 'Advanced SQL Queries Practice', done: false },
    { label: 'Migration strategies', done: false },
    { label: 'Indexes and query plans', done: false },
  ],
};

export const MICROSERVICES_MILESTONE: MockMilestone = {
  title: 'Microservices',
  steps: [
    { label: 'Saga in microservices', done: true },
    { label: 'Message Queue benefits', done: true },
    { label: 'Ordering problem review', done: true },
    { label: 'gRPC connection load balancing', done: true },
    { label: 'RabbitMQ prefetch', done: true },
    { label: 'DLQ setup', done: false },
    { label: 'gRPC deadline propagation', done: true },
    { label: 'Kafka review', done: false },
  ],
};

/**
 * Done and total, derived from the flags rather than typed beside them, so a
 * count shown on a tile can never disagree with the steps it summarises.
 */
export function progressOf(milestone: MockMilestone): { done: number; total: number } {
  return {
    done: milestone.steps.filter((step) => step.done).length,
    total: milestone.steps.length,
  };
}

/** One Day plan entry: an open step to work on today (FS-XCTMG R15, TODAY). */
export interface MockDayEntry {
  /** The step's label, as its milestone lists it. */
  step: string;
  milestone: string;
  /** The AI's one-line reason, kept short. */
  reason: string;
  /** Rough estimate, as the Day plan shows it ("~30 min"). */
  minutes: number;
  /**
   * The day tick, "done for today", already on at start. Not the step's
   * `done`: the step stays open, so PLAN's counts are untouched.
   */
  ticked: boolean;
}

export interface MockDayGroup {
  label: string;
  entries: readonly MockDayEntry[];
}

/**
 * An entry for one of `milestone`'s open steps. Throws at import if the step
 * is missing or already done, so the Day plan can only ever suggest work the
 * PLAN tile still shows as open.
 */
function entryFor(
  milestone: MockMilestone,
  step: string,
  rest: Omit<MockDayEntry, 'step' | 'milestone'>
): MockDayEntry {
  if (!milestone.steps.some((s) => s.label === step && !s.done)) {
    throw new Error(`"${step}" is not an open step of ${milestone.title}`);
  }
  return { step, milestone: milestone.title, ...rest };
}

/** The day the Day plan is for. A weekday only, so it never dates the page. */
export const DAY_LABEL = 'Tuesday';

/** Today's Day plan: a few open steps, grouped, one already ticked. */
export const DAY_PLAN: readonly MockDayGroup[] = [
  {
    label: 'Deep focus',
    entries: [
      entryFor(DATABASE_MILESTONE, 'SQL Tools Drill Practice', {
        reason: 'Builds on yesterday’s basics',
        minutes: 45,
        ticked: true,
      }),
      entryFor(DATABASE_MILESTONE, 'Advanced SQL Queries Practice', {
        reason: 'Next in Database',
        minutes: 30,
        ticked: false,
      }),
    ],
  },
  {
    label: 'Quick wins',
    entries: [
      entryFor(MICROSERVICES_MILESTONE, 'DLQ setup', {
        reason: 'One of two left in Microservices',
        minutes: 20,
        ticked: false,
      }),
      entryFor(MICROSERVICES_MILESTONE, 'Kafka review', {
        reason: 'Pairs with RabbitMQ prefetch',
        minutes: 15,
        ticked: false,
      }),
    ],
  },
];

/** Who wrote a review message: the persona, or Fireplace replying. */
export type MockReviewSpeaker = 'persona' | 'fireplace';

export interface MockReviewMessage {
  from: MockReviewSpeaker;
  text: string;
}

/**
 * The longest a review bubble may run: two lines in the REVIEW tile's narrowest
 * bubble. That is the single-column tile at a 320px phone (narrower than a third
 * of the lg grid at 1024px), where the bubble takes the tile's full 222px with
 * 14px text and 12px side padding (ReviewTileMock, below `sm`): about 34 Lora
 * characters a line, so two lines less one long word of wrap slack. Measured.
 * Longer would be a note, and the tour carries no long notes.
 */
export const REVIEW_BUBBLE_MAX_CHARS = 56;

/**
 * A Day plan entry's step, by name. Throws at import if `step` is not in
 * DAY_PLAN, so the review can only talk about the day the TODAY tile plans.
 */
function dayStep(step: string): string {
  if (!DAY_PLAN.some((group) => group.entries.some((entry) => entry.step === step))) {
    throw new Error(`"${step}" is not in the ${DAY_LABEL} Day plan`);
  }
  return step;
}

/**
 * The end-of-day review of that same Tuesday (FS-XCTMG R15, REVIEW): Chris's
 * note and one reply at start; `next` is the reply that slides in on reveal.
 */
export const REVIEW_THREAD: {
  start: readonly MockReviewMessage[];
  next: MockReviewMessage;
} = {
  start: [
    {
      from: 'persona',
      text: `Finished ${dayStep('SQL Tools Drill Practice')}. ${dayStep('DLQ setup')} slipped.`,
    },
    { from: 'fireplace', text: 'Good deep-focus block on the SQL drills.' },
  ],
  next: { from: 'fireplace', text: `I’ve put ${dayStep('DLQ setup')} first in tomorrow’s plan.` },
};

/** One segment of the plan's progress strip: a milestone, by its counts only. */
export interface MockStripMilestone {
  title: string;
  done: number;
  total: number;
}

/** A milestone the PLAN tile shows in full, read through its own steps. */
function stripOf(milestone: MockMilestone): MockStripMilestone {
  return { title: milestone.title, ...progressOf(milestone) };
}

/**
 * Every milestone of Learn microservices, in plan order (FS-XCTMG R15,
 * MILESTONES). The early ones are closed, Database and Microservices are the
 * two the PLAN tile has open, and the rest are still ahead.
 */
export const PLAN_MILESTONES: readonly MockStripMilestone[] = [
  { title: 'HTTP and REST', done: 4, total: 4 },
  { title: 'Go fundamentals', done: 6, total: 6 },
  { title: 'Docker basics', done: 3, total: 3 },
  { title: 'Service boundaries', done: 4, total: 4 },
  stripOf(DATABASE_MILESTONE),
  stripOf(MICROSERVICES_MILESTONE),
  { title: 'API gateway', done: 0, total: 5 },
  { title: 'Auth and tokens', done: 0, total: 4 },
  { title: 'Caching', done: 0, total: 3 },
  { title: 'Observability', done: 0, total: 4 },
  { title: 'Resilience patterns', done: 0, total: 3 },
  { title: 'CI/CD', done: 0, total: 4 },
  { title: 'Kubernetes', done: 0, total: 6 },
  { title: 'Capstone project', done: 0, total: 5 },
];

/**
 * The month the CALENDAR tile shows (FS-XCTMG R15). Fixed and invented, never
 * read from the clock, so server and client render the same grid and the page
 * never dates itself. A name with no year; `firstWeekday` is 0 for Sunday.
 * A 28-day month starting on a Sunday fills exactly four week rows.
 */
export const CALENDAR_MONTH = { name: 'February', firstWeekday: 0, days: 28 } as const;

/** One dated item on the month: a step, over a run of days in one week row. */
export interface MockCalendarItem {
  step: string;
  milestone: string;
  /** First and last day of the month the item covers, inclusive. */
  start: number;
  end: number;
  /** The step's own `done`, so the calendar agrees with the PLAN tile. */
  done: boolean;
}

/**
 * A dated item for one of `milestone`'s steps, done or open as the step is.
 * Throws at import if the step is missing, so the calendar can only show work
 * the PLAN tile lists.
 */
function datedStep(milestone: MockMilestone, step: string, start: number, end: number): MockCalendarItem {
  const found = milestone.steps.find((s) => s.label === step);
  if (!found) throw new Error(`"${step}" is not a step of ${milestone.title}`);
  return { step, milestone: milestone.title, start, end, done: found.done };
}

/** The day that lights up on reveal: the Tuesday of the Day plan (DAY_LABEL). */
export const CALENDAR_HIGHLIGHT = 17;

/**
 * The month's dated work, one item per week row, in reading order: two
 * Microservices steps already done, the Day plan's Tuesday, and what is next.
 */
export const CALENDAR_ITEMS: readonly MockCalendarItem[] = [
  datedStep(MICROSERVICES_MILESTONE, 'Ordering problem review', 3, 5),
  datedStep(MICROSERVICES_MILESTONE, 'gRPC deadline propagation', 10, 13),
  datedStep(DATABASE_MILESTONE, 'SQL Tools Drill Practice', CALENDAR_HIGHLIGHT, 20),
  datedStep(DATABASE_MILESTONE, 'Migration strategies', 24, 27),
];
