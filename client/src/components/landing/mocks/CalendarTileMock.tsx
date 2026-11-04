'use client';

import { cn } from '@/lib/utils';
import Mock from './Mock';
import { CALENDAR_HIGHLIGHT, CALENDAR_ITEMS, CALENDAR_MONTH, type MockCalendarItem } from './content';
import { RevealGlow, RevealSwap } from '../bento/reveal';

const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/** A day's place in the grid: its row, and its column (0 = Sunday). */
function cellOf(day: number) {
  const index = CALENDAR_MONTH.firstWeekday + day - 1;
  return { week: Math.floor(index / 7), column: index % 7 };
}

/**
 * The month as full week rows: `null` pads the days before the 1st and after
 * the last, as production's grid does with the neighbouring months' days.
 */
const WEEKS: (number | null)[][] = (() => {
  const lead = CALENDAR_MONTH.firstWeekday;
  const cells: (number | null)[] = [
    ...Array<null>(lead).fill(null),
    ...Array.from({ length: CALENDAR_MONTH.days }, (_, i) => i + 1),
  ];
  while (cells.length % 7) cells.push(null);
  return Array.from({ length: cells.length / 7 }, (_, w) => cells.slice(w * 7, w * 7 + 7));
})();

/**
 * The CALENDAR tile (FS-XCTMG R15): a month card styled like production's
 * compact calendar, with the plan's steps dated across it as bars. On reveal
 * one day lights up: the Tuesday the TODAY tile plans.
 */
export default function CalendarTileMock() {
  return (
    <Mock className="space-y-3 rounded-none border-0 border-transparent bg-transparent p-0 shadow-none">
      <div className="flex items-baseline justify-between gap-3">
        <p className="text-lg font-medium leading-7 text-foreground">{CALENDAR_MONTH.name}</p>
        <p className="text-sm text-foreground/60">Schedule</p>
      </div>

      <div className="space-y-1">
        <div className="grid grid-cols-7 gap-1">
          {WEEKDAYS.map((name) => (
            <span key={name} className="py-0.5 text-center text-xs font-semibold text-foreground/45">
              {name[0]}
            </span>
          ))}
        </div>
        {WEEKS.map((week, w) => (
          <WeekRow
            key={w}
            days={week}
            items={CALENDAR_ITEMS.filter((item) => cellOf(item.start).week === w)}
          />
        ))}
      </div>
    </Mock>
  );
}

/** One Sun..Sat row: the day cells, and the bars of the items dated in it. */
function WeekRow({ days, items }: { days: (number | null)[]; items: readonly MockCalendarItem[] }) {
  return (
    <div className="relative">
      <div className="grid grid-cols-7 gap-1">
        {days.map((day, i) =>
          day === null ? (
            <span key={`pad-${i}`} className="h-11 rounded-md" />
          ) : (
            <DayCell key={day} day={day} />
          )
        )}
      </div>

      {/* Bars sit under the day numbers and span their days, as production's overlay. */}
      <div className="absolute inset-x-0 top-[22px] grid grid-cols-7 gap-1">
        {items.map((item) => {
          const { column } = cellOf(item.start);
          return (
            <span
              key={item.step}
              data-calendar-item={item.step}
              // Count-dependent placement, hence inline.
              style={{ gridColumn: `${column + 1} / span ${item.end - item.start + 1}` }}
              className={cn(
                'mx-1 truncate rounded px-1.5 py-0.5 text-[11px] leading-4',
                item.done ? 'bg-foreground/[0.06] text-foreground/45 line-through' : 'bg-primary/15 text-foreground/80'
              )}
            >
              {item.step}
            </span>
          );
        })}
      </div>
    </div>
  );
}

/** A day of the month. The highlighted one carries a layer that lights up on reveal. */
function DayCell({ day }: { day: number }) {
  const lit = day === CALENDAR_HIGHLIGHT;
  return (
    <span
      data-calendar-day={day}
      data-weekday={WEEKDAYS[cellOf(day).column]}
      className="relative block h-11 rounded-md border border-foreground/[0.07] bg-foreground/[0.03] p-1 text-xs"
    >
      {lit && <RevealGlow className="absolute inset-0 rounded-md bg-primary/10 ring-2 ring-primary/50" />}
      <span className="relative block font-medium tabular-nums leading-4">
        {lit ? (
          <RevealSwap
            before={<span className="text-foreground/70">{day}</span>}
            after={<span className="text-primary">{day}</span>}
          />
        ) : (
          <span className="text-foreground/70">{day}</span>
        )}
      </span>
    </span>
  );
}
