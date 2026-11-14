'use client';

import Mock from './Mock';
import { PLAN_MILESTONES, PLAN_TITLE, type MockStripMilestone } from './content';
import { RevealFill, RevealSwap } from '../bento/reveal';

/**
 * Gap between one segment starting to fill and the next, in ms. Only open
 * segments fill, so the last one's delay is (open − 1) × STAGGER_MS, and that
 * plus the module's --reveal-ms must stay inside BentoTile's REVEAL_MS. Ten
 * open segments at 50ms end at 450ms + 700ms, inside 1200ms.
 */
const STAGGER_MS = 50;

const TOTAL = PLAN_MILESTONES.length;

/** Closed: every step done. The strip's headline counts these (CONTEXT.md). */
function isClosed(milestone: MockStripMilestone): boolean {
  return milestone.total > 0 && milestone.done === milestone.total;
}

const CLOSED = PLAN_MILESTONES.filter(isClosed).length;

/**
 * When each segment starts filling on reveal: the open ones, left to right,
 * one STAGGER_MS apart. Closed segments are full already and have none.
 */
const FILL_DELAYS = new Map(
  PLAN_MILESTONES.filter((m) => !isClosed(m)).map((m, k) => [m, k * STAGGER_MS])
);
const LAST_FILL = Math.max(0, ...FILL_DELAYS.values());

/**
 * The MILESTONES tile (FS-XCTMG R15): the plan header's overview, styled like
 * production's progress strip. The headline "N of 14 milestones" sits over one
 * segment per milestone, each filled to that milestone's own progress. On
 * reveal the open segments fill one by one and the headline cross-fades to
 * match as the last one lands.
 */
export default function MilestonesTileMock() {
  return (
    // h-full + justify-between: beside CALENDAR at lg the row is taller than
    // this mock, so the strip sits on the cell's floor instead of floating.
    <Mock className="flex h-full flex-col justify-between gap-4 rounded-none border-0 border-transparent bg-transparent p-0 shadow-none">
      <div className="space-y-1.5">
        <p className="text-lg font-medium leading-7 text-foreground">{PLAN_TITLE}</p>
        <p className="text-sm tabular-nums text-foreground/60">
          <RevealSwap
            delay={LAST_FILL}
            before={`${CLOSED} of ${TOTAL} milestones`}
            after={`${TOTAL} of ${TOTAL} milestones`}
          />
        </p>
      </div>

      {/* One equal column per milestone, as production's track. Count-dependent, hence inline. */}
      <div
        data-milestone-track
        className="grid gap-0.5"
        style={{ gridTemplateColumns: `repeat(${TOTAL}, minmax(0, 1fr))` }}
      >
        {PLAN_MILESTONES.map((milestone) => (
          <span
            key={milestone.title}
            data-milestone-segment={milestone.title}
            className="h-2 overflow-hidden rounded-full bg-foreground/[0.08]"
          >
            <RevealFill
              from={milestone.done / milestone.total}
              to={1}
              delay={FILL_DELAYS.get(milestone)}
              className="h-full rounded-full bg-primary"
            />
          </span>
        ))}
      </div>
    </Mock>
  );
}
