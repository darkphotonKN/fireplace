'use client';

import Mock from './Mock';
import {
  DATABASE_MILESTONE,
  MICROSERVICES_MILESTONE,
  PLAN_TITLE,
  progressOf,
  type MockMilestone,
  type MockStep,
} from './content';
import { RevealFill, RevealIn, RevealSwap } from '../bento/reveal';

/** Steps a card shows at start, and how many more arrive on reveal. */
const SHOWN = 3;
const ARRIVING = 1;
/** Stagger between the tick and each arriving row, in ms. */
const STAGGER_MS = 120;

/**
 * The PLAN tile (FS-XCTMG R15): the Database and Microservices milestone cards,
 * styled like the production milestone cards. On reveal the first open
 * Database step ticks, its count cross-fades 1/5 → 2/5 and its bar fills to
 * match, and one more step arrives in each card.
 */
export default function PlanTileMock() {
  return (
    <Mock className="space-y-3 rounded-none border-0 bg-transparent p-0 shadow-none">
      <p className="text-sm font-medium text-foreground/60">{PLAN_TITLE}</p>
      <MilestoneCard milestone={DATABASE_MILESTONE} ticks />
      <MilestoneCard milestone={MICROSERVICES_MILESTONE} />
    </Mock>
  );
}

function MilestoneCard({ milestone, ticks = false }: { milestone: MockMilestone; ticks?: boolean }) {
  const { done, total } = progressOf(milestone);
  // The step that ticks on reveal: the first one still open.
  const tickAt = ticks ? milestone.steps.findIndex((step) => !step.done) : -1;
  const after = tickAt >= 0 ? done + 1 : done;
  const visible = milestone.steps.slice(0, SHOWN + ARRIVING);

  return (
    <div data-milestone={milestone.title} className="rounded-xl bg-foreground/[0.04] px-4 py-4">
      <div className="flex items-start gap-2.5">
        <span className="mt-[3px] shrink-0">
          <Box done={false} />
        </span>
        <span className="text-base font-medium leading-6 text-foreground">{milestone.title}</span>
      </div>

      <div className="mt-2.5 flex items-center gap-2.5">
        <span className="h-1 flex-1 overflow-hidden rounded-full bg-foreground/[0.08]">
          <RevealFill from={done / total} to={after / total} className="h-full rounded-full bg-primary" />
        </span>
        <span className="shrink-0 text-sm tabular-nums text-foreground/45">
          {after === done ? (
            `${done}/${total}`
          ) : (
            <RevealSwap before={`${done}/${total}`} after={`${after}/${total}`} />
          )}
        </span>
      </div>

      <ul className="mt-3 space-y-0.5 border-t border-foreground/[0.07] pt-2.5">
        {visible.map((step, i) => {
          const row = <StepRow step={step} ticks={i === tickAt} />;
          return i < SHOWN ? (
            <li key={step.label}>{row}</li>
          ) : (
            <RevealIn key={step.label} as="li" delay={STAGGER_MS * (i - SHOWN + 1)}>
              {row}
            </RevealIn>
          );
        })}
      </ul>
    </div>
  );
}

function StepRow({ step, ticks }: { step: MockStep; ticks: boolean }) {
  return (
    <div data-step data-step-done={String(step.done)} className="flex items-start gap-2.5 py-1">
      <span className="mt-[2px] shrink-0">
        {ticks ? <RevealSwap before={<Box done={false} />} after={<Box done />} /> : <Box done={step.done} />}
      </span>
      {ticks ? (
        <RevealSwap before={<StepLabel step={step} />} after={<StepLabel step={{ ...step, done: true }} />} />
      ) : (
        <StepLabel step={step} />
      )}
    </div>
  );
}

function StepLabel({ step }: { step: MockStep }) {
  return (
    <span
      className={
        step.done
          ? 'text-sm leading-5 text-foreground/40 line-through'
          : 'text-sm leading-5 text-foreground/80'
      }
    >
      {step.label}
    </span>
  );
}

/** A fake checkbox: a `<span>`, never an `<input>` (mocks are decorative). */
function Box({ done }: { done: boolean }) {
  if (!done) return <span className="block h-4 w-4 rounded-sm border border-foreground/25" />;
  return (
    <span data-tick className="flex h-4 w-4 items-center justify-center rounded-sm bg-primary">
      <svg viewBox="0 0 12 12" className="h-3 w-3 text-primary-foreground">
        <path
          d="M2.5 6.2 4.8 8.5 9.5 3.8"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </span>
  );
}
