"use client";

import { useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
  format,
  isSameDay,
  isSameMonth,
  differenceInCalendarDays,
} from "date-fns";

import {
  CalendarItem,
  CalendarView,
  getPlanCalendar,
} from "@/services/api";
import {
  formatViewAnchor,
  getDaysInWindow,
  layoutItem,
  resolveWindow,
  stepAnchor,
} from "@/lib/calendar";

interface CalendarProps {
  planId: string;
  className?: string;
  /** Compact mode renders a denser grid (used inside CalendarCard on plan page). */
  compact?: boolean;
}

const VIEW_KEY = "calendarView";

const weekDays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function readStoredView(): CalendarView {
  if (typeof window === "undefined") return "month";
  const v = window.localStorage.getItem(VIEW_KEY);
  return v === "week" ? "week" : "month";
}

export function Calendar({ planId, className = "", compact = false }: CalendarProps) {
  const [view, setView] = useState<CalendarView>("month");
  const [anchor, setAnchor] = useState<Date>(() => new Date());
  const [items, setItems] = useState<CalendarItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Hydrate persisted view after mount (avoids SSR / hydration mismatch).
  useEffect(() => {
    setView(readStoredView());
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(VIEW_KEY, view);
    }
  }, [view]);

  const window_ = useMemo(() => resolveWindow(view, formatViewAnchor(anchor, view)), [view, anchor]);
  const days = useMemo(() => getDaysInWindow(window_), [window_]);

  useEffect(() => {
    let cancelled = false;
    const date = formatViewAnchor(anchor, view);
    setLoading(true);
    setError(null);
    getPlanCalendar(planId, view, date)
      .then((res) => {
        if (cancelled) return;
        setItems(res.result?.items ?? []);
      })
      .catch((err) => {
        if (cancelled) return;
        console.error("Failed to fetch calendar:", err);
        setError("Failed to load calendar");
        setItems([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [planId, view, anchor]);

  const goPrev = () => setAnchor((a) => stepAnchor(a, view, -1));
  const goNext = () => setAnchor((a) => stepAnchor(a, view, 1));
  const goToday = () => setAnchor(new Date());

  const heading = view === "month" ? format(anchor, "MMMM yyyy") : `Week of ${format(window_.start, "MMM d, yyyy")}`;

  return (
    <div className={`bg-white/5 backdrop-blur-sm rounded-xl ${compact ? "p-4" : "p-6"} ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <h2 className={compact ? "text-lg font-semibold" : "text-2xl font-bold"}>{heading}</h2>
        <div className="flex items-center gap-2">
          <ViewToggle view={view} onChange={setView} />
          <button
            onClick={goToday}
            className="px-2 py-1 text-xs rounded hover:bg-white/10 transition-colors"
            aria-label="Today"
          >
            Today
          </button>
          <button onClick={goPrev} className="p-1 hover:bg-white/10 rounded transition-colors" aria-label="Previous">
            <ChevronLeft className="w-4 h-4" />
          </button>
          <button onClick={goNext} className="p-1 hover:bg-white/10 rounded transition-colors" aria-label="Next">
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {error && <div className="text-sm text-red-400 mb-2">{error}</div>}
      {loading && <div className="text-sm text-gray-500 mb-2">Loading…</div>}

      {view === "month" ? (
        <MonthGrid days={days} anchor={anchor} items={items} window={window_} compact={compact} />
      ) : (
        <WeekGrid days={days} items={items} window={window_} compact={compact} />
      )}
    </div>
  );
}

function ViewToggle({ view, onChange }: { view: CalendarView; onChange: (v: CalendarView) => void }) {
  const base = "px-2 py-1 text-xs rounded transition-colors";
  return (
    <div className="inline-flex bg-white/5 rounded p-0.5">
      <button
        onClick={() => onChange("week")}
        className={`${base} ${view === "week" ? "bg-amber-500/20 text-amber-400" : "hover:bg-white/10"}`}
        aria-pressed={view === "week"}
      >
        Week
      </button>
      <button
        onClick={() => onChange("month")}
        className={`${base} ${view === "month" ? "bg-amber-500/20 text-amber-400" : "hover:bg-white/10"}`}
        aria-pressed={view === "month"}
      >
        Month
      </button>
    </div>
  );
}

interface GridSectionProps {
  days: Date[];
  items: CalendarItem[];
  window: { start: Date; end: Date };
  compact: boolean;
}

function MonthGrid({ days, anchor, items, window, compact }: GridSectionProps & { anchor: Date }) {
  // Monthly view: chunk into weeks of 7. resolveWindow gives first..last of month;
  // pad with leading/trailing days so the grid is full weeks.
  const padded = padToFullWeeks(days);
  const weeks: Date[][] = [];
  for (let i = 0; i < padded.length; i += 7) weeks.push(padded.slice(i, i + 7));

  return (
    <div className="space-y-1">
      <div className="grid grid-cols-7 gap-1 mb-1">
        {weekDays.map((d) => (
          <div key={d} className="text-center text-xs font-semibold text-gray-500 py-1">
            {compact ? d[0] : d}
          </div>
        ))}
      </div>
      {weeks.map((week, wIdx) => (
        <WeekRow
          key={wIdx}
          days={week}
          items={items}
          window={window}
          rowCellMinHeight={compact ? 56 : 96}
          dimOutOfMonth={anchor}
        />
      ))}
    </div>
  );
}

function WeekGrid({ days, items, window, compact }: GridSectionProps) {
  return (
    <div className="space-y-1">
      <div className="grid grid-cols-7 gap-1 mb-1">
        {weekDays.map((d) => (
          <div key={d} className="text-center text-xs font-semibold text-gray-500 py-1">
            {compact ? d[0] : d}
          </div>
        ))}
      </div>
      <WeekRow days={days} items={items} window={window} rowCellMinHeight={compact ? 96 : 160} />
    </div>
  );
}

interface WeekRowProps {
  days: Date[];
  items: CalendarItem[];
  window: { start: Date; end: Date };
  rowCellMinHeight: number;
  dimOutOfMonth?: Date; // when set, days not in this month render dimmed
}

/**
 * Renders one Sun..Sat row plus the bars/chips that intersect the row.
 * Bars span across cells via a CSS grid overlay above the day cells.
 */
function WeekRow({ days, items, window, rowCellMinHeight, dimOutOfMonth }: WeekRowProps) {
  const today = new Date();
  const rowStart = days[0];
  const rowEnd = days[days.length - 1];

  // Items whose range intersects this row.
  const placed = items
    .map((item) => layoutItem(item, { start: rowStart, end: rowEnd }))
    .filter((p) => p.shape !== "none")
    .filter((p) => p.visibleEnd >= rowStart && p.visibleStart <= rowEnd);

  return (
    <div className="relative">
      <div className="grid grid-cols-7 gap-1">
        {days.map((d, i) => {
          const isToday = isSameDay(d, today);
          const dim = dimOutOfMonth ? !isSameMonth(d, dimOutOfMonth) : false;
          return (
            <div
              key={i}
              className={`rounded-lg border p-1 text-xs transition-colors ${
                dim ? "border-transparent opacity-40" : "border-white/10 bg-white/5"
              } ${isToday ? "ring-2 ring-amber-500/50 bg-amber-500/10" : ""}`}
              style={{ minHeight: rowCellMinHeight }}
            >
              <div className={`font-medium ${isToday ? "text-amber-400" : ""}`}>{format(d, "d")}</div>
            </div>
          );
        })}
      </div>

      {/* Bars / chips overlay */}
      <div
        className="absolute left-0 right-0 grid grid-cols-7 gap-1 pointer-events-none"
        style={{ top: 22 }}
      >
        {placed.map((p, i) => {
          const startCol = differenceInCalendarDays(p.visibleStart, rowStart);
          const endCol = differenceInCalendarDays(p.visibleEnd, rowStart);
          const span = endCol - startCol + 1;
          const stackOffset = i * 22;

          if (p.shape === "chip") {
            return (
              <div
                key={p.item.id}
                className="pointer-events-auto"
                style={{ gridColumn: `${startCol + 1} / span 1`, transform: `translateY(${stackOffset}px)` }}
                title={p.item.description}
              >
                <div
                  className={`mx-1 px-1.5 py-0.5 text-[11px] rounded truncate ${
                    p.item.done
                      ? "bg-white/10 text-gray-400 line-through"
                      : "bg-amber-500/20 text-amber-200"
                  }`}
                >
                  {p.item.description}
                </div>
              </div>
            );
          }

          return (
            <div
              key={p.item.id}
              className="pointer-events-auto"
              style={{ gridColumn: `${startCol + 1} / span ${span}`, transform: `translateY(${stackOffset}px)` }}
              title={p.item.description}
            >
              <div
                className={`mx-1 px-1.5 py-0.5 text-[11px] rounded truncate flex items-center gap-1 ${
                  p.item.done
                    ? "bg-white/10 text-gray-400 line-through"
                    : "bg-amber-500/30 text-amber-100"
                } ${p.clipsLeft ? "rounded-l-none border-l-2 border-amber-400" : ""} ${
                  p.clipsRight ? "rounded-r-none border-r-2 border-amber-400" : ""
                }`}
              >
                {p.clipsLeft && <span aria-hidden>‹</span>}
                <span className="truncate">{p.item.description}</span>
                {p.clipsRight && <span aria-hidden>›</span>}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/**
 * Pad a sequence of days to start on Sunday and end on Saturday so the grid is
 * always full weeks.
 */
function padToFullWeeks(days: Date[]): Date[] {
  if (days.length === 0) return days;
  const first = days[0];
  const last = days[days.length - 1];
  const lead = first.getDay(); // 0 Sun .. 6 Sat
  const trail = 6 - last.getDay();
  const padded: Date[] = [];
  for (let i = lead; i > 0; i--) {
    const d = new Date(first);
    d.setDate(first.getDate() - i);
    padded.push(d);
  }
  padded.push(...days);
  for (let i = 1; i <= trail; i++) {
    const d = new Date(last);
    d.setDate(last.getDate() + i);
    padded.push(d);
  }
  return padded;
}
