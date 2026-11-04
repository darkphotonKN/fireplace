'use client';

import { cn } from '@/lib/utils';
import Mock from './Mock';
import { DAY_LABEL, DAY_PLAN, type MockDayEntry } from './content';
import { RevealFill, RevealSwap } from '../bento/reveal';

/**
 * Gap between one tick landing and the next, in ms. The last tick's delay plus
 * the module's --reveal-ms must stay inside BentoTile's REVEAL_MS.
 */
const STAGGER_MS = 160;

const ENTRIES = DAY_PLAN.flatMap((group) => group.entries);
const TOTAL = ENTRIES.length;
const TICKED_AT_START = ENTRIES.filter((entry) => entry.ticked).length;

/**
 * When each entry's tick lands: the entries still open at start, in reading
 * order, one STAGGER_MS apart. Ticked-at-start entries have none.
 */
const TICK_DELAYS = new Map(
  ENTRIES.filter((entry) => !entry.ticked).map((entry, k) => [entry, k * STAGGER_MS])
);

/**
 * The TODAY tile (FS-XCTMG R15): a Day plan, styled like the production Day
 * plan section, on the inverted tile, so every colour is drawn from
 * `secondary-foreground` (R12). On reveal the open entries tick one after
 * another and the day meter steps up with each.
 */
export default function TodayTileMock() {
  return (
    <Mock className="space-y-4 rounded-none border-0 border-transparent bg-transparent p-0 shadow-none">
      <DayMeter />
      {DAY_PLAN.map((group) => (
        <div
          key={group.label}
          data-day-group={group.label}
          className="rounded-xl bg-secondary-foreground/[0.06] px-4 py-4"
        >
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-base font-medium leading-6 text-secondary-foreground">
              {group.label}
            </span>
            <span className="shrink-0 text-sm tabular-nums text-secondary-foreground/45">
              {minutesLabel(group.entries.reduce((sum, entry) => sum + entry.minutes, 0))}
            </span>
          </div>
          <ul className="mt-3 space-y-3 border-t border-secondary-foreground/10 pt-3">
            {group.entries.map((entry) => (
              <EntryRow key={entry.step} entry={entry} delay={TICK_DELAYS.get(entry)} />
            ))}
          </ul>
        </div>
      ))}
    </Mock>
  );
}

/**
 * "N of M done today" over a quiet meter. The meter is one layer per tick, all
 * at the start share, each rising to its tick's share at its tick's moment:
 * stacked, so the bar steps up with the ticks on transform alone.
 */
function DayMeter() {
  const lastTick = Math.max(0, ...TICK_DELAYS.values());
  return (
    <div className="space-y-2">
      <p className="flex gap-1 text-sm tabular-nums text-secondary-foreground/55">
        <span>{DAY_LABEL} ·</span>
        <RevealSwap
          delay={lastTick}
          before={`${TICKED_AT_START} of ${TOTAL} done today`}
          after={`${TOTAL} of ${TOTAL} done today`}
        />
      </p>
      <span className="relative block h-1.5 overflow-hidden rounded-full bg-secondary-foreground/15">
        {Array.from(TICK_DELAYS.values(), (delay, k) => (
          <RevealFill
            key={k}
            from={TICKED_AT_START / TOTAL}
            to={(TICKED_AT_START + k + 1) / TOTAL}
            delay={delay}
            className="absolute inset-0 rounded-full bg-primary"
          />
        ))}
      </span>
    </div>
  );
}

/** `delay` is when this entry's tick lands; none means it is ticked at start. */
function EntryRow({ entry, delay }: { entry: MockDayEntry; delay?: number }) {
  const ticks = delay !== undefined;
  return (
    <li data-entry={entry.step} data-entry-ticked={String(entry.ticked)} className="flex items-start gap-2.5">
      <span className="mt-[2px] shrink-0">
        {ticks ? (
          <RevealSwap delay={delay} before={<Box ticked={false} />} after={<Box ticked />} />
        ) : (
          <Box ticked={entry.ticked} />
        )}
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-secondary-foreground/40">{entry.milestone}</p>
        {ticks ? (
          <RevealSwap
            delay={delay}
            before={<EntryText step={entry.step} ticked={false} />}
            after={<EntryText step={entry.step} ticked />}
          />
        ) : (
          <EntryText step={entry.step} ticked={entry.ticked} />
        )}
        <p className="text-xs leading-5 text-secondary-foreground/45">
          {entry.reason}
          <span className="ml-2 whitespace-nowrap tabular-nums text-secondary-foreground/35">
            ~{entry.minutes} min
          </span>
        </p>
      </div>
    </li>
  );
}

/** A day tick settles the row rather than striking it: done for today isn't done. */
function EntryText({ step, ticked }: { step: string; ticked: boolean }) {
  return (
    <span
      className={cn(
        'block text-sm leading-5',
        ticked ? 'text-secondary-foreground/55' : 'text-secondary-foreground'
      )}
    >
      {step}
    </span>
  );
}

/** A fake checkbox: a `<span>`, never an `<input>` (mocks are decorative). */
function Box({ ticked }: { ticked: boolean }) {
  if (!ticked) return <span className="block h-4 w-4 rounded-sm border border-secondary-foreground/30" />;
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

/** A rough total, as the Day plan's group header puts it: "~35 min", "~1h 15m". */
function minutesLabel(total: number): string {
  if (total < 60) return `~${total} min`;
  const rest = total % 60;
  return `~${Math.floor(total / 60)}h${rest ? ` ${rest}m` : ''}`;
}
