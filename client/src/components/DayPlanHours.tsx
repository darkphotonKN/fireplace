'use client';

import { useRef, useState } from 'react';
import { Minus, Plus } from 'lucide-react';
import { ApiError } from '@/api/client';
import { setDayPlanSettings, type DayPlanSettings } from '@/api/insights';
import { hoursInWords } from '@/lib/dayPlan';
import { cn } from '@/lib/utils';

/**
 * Hours a day on the Day plan (FS-02JBZ §Change: Hours a day): the one save
 * both the first ask (R109–R111) and the cog (R116–R118) go through, the
 * cog's queue of saves, and the cog's button and settings view. Saving never
 * draws (R103, ADR-0019): drawing is the section's job.
 */

/**
 * The user's daily time on a plan, in minutes. `null` is no budget: "Skip,
 * plan anyway" on the first ask, "Use the default" on the cog (R94, R109).
 */
export type HoursAnswer = number | null;

/** How one save of the hours went. */
export type HoursSave =
  | { outcome: 'saved'; settings: DayPlanSettings }
  /** A 404: the plan is not reachable, and the section follows R70. */
  | { outcome: 'gone' }
  /** Any other error, already logged. */
  | { outcome: 'failed' };

const NOT_FOUND = 'NOT_FOUND';

/** Saves the hours once and says how it went. Never throws. */
export async function saveHours(
  planId: string,
  minutes: HoursAnswer
): Promise<HoursSave> {
  try {
    return {
      outcome: 'saved',
      settings: await setDayPlanSettings(planId, minutes),
    };
  } catch (err) {
    // Switch on the domain code, never on `detail` (ADR-0004).
    if (err instanceof ApiError && err.code === NOT_FOUND) {
      return { outcome: 'gone' };
    }
    console.error('Failed to save the hours a day:', err);
    return { outcome: 'failed' };
  }
}

/**
 * The cog's saves (R117): each change is shown at once and saved behind any
 * save still in flight, so requests go one after another and the stored
 * value is the last one pressed. Only the last press decides how the readout
 * settles: on its value, or, if its save fails, back on the last saved one
 * with `failed` set. A 404 calls `onGone` and drops what is still queued.
 */
export function useHoursSaver({
  planId,
  onSaved,
  onGone,
}: {
  planId: string;
  onSaved: (settings: DayPlanSettings) => void;
  onGone: () => void;
}) {
  // The value being saved; `null` while nothing is in flight.
  const [draft, setDraft] = useState<{ minutes: HoursAnswer } | null>(null);
  const [failed, setFailed] = useState(false);
  const chainRef = useRef<Promise<void>>(Promise.resolve());
  const pressRef = useRef(0);
  // Bumped by a 404: a press queued before it belongs to a plan that's gone.
  const epochRef = useRef(0);

  const save = (minutes: HoursAnswer) => {
    const press = ++pressRef.current;
    const epoch = epochRef.current;
    const latest = () => press === pressRef.current;
    setDraft({ minutes });
    setFailed(false);
    chainRef.current = chainRef.current.then(async () => {
      if (epoch !== epochRef.current) return;
      const result = await saveHours(planId, minutes);
      if (result.outcome === 'gone') {
        epochRef.current++;
        setDraft(null);
        onGone();
        return;
      }
      if (result.outcome === 'saved') onSaved(result.settings);
      // A later press is queued behind this one and will decide.
      if (!latest()) return;
      setDraft(null);
      setFailed(result.outcome === 'failed');
    });
  };

  return {
    /** What the stepper shows: the value being saved, else the saved one. */
    readout: (saved: DayPlanSettings): HoursAnswer =>
      draft ? draft.minutes : saved.dailyMinutes ?? null,
    failed,
    save,
    clearFailed: () => setFailed(false),
  };
}

/** The card surface the not-planned states and the settings row share. */
export const QUIET_CARD =
  'rounded-xl bg-foreground/[0.04] px-4 py-4 backdrop-blur-sm';

/**
 * The Day plan's settings button (R116): the Items card's cog, glyph and
 * all, on tokens.
 */
export function SettingsCog({ onOpen }: { onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label="Day plan settings"
      title="Settings"
      className="rounded-full p-1.5 text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden
        className="h-4 w-4"
      >
        <path
          fillRule="evenodd"
          d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  );
}

/** The stepper's range and step (R94, R116), and where unset starts (R117). */
const HOURS_MIN = 30;
const HOURS_MAX = 720;
const HOURS_STEP = 30;
const HOURS_START = 120;

/** A −/+ in the stepper's pill: quiet, ember on hover, like the view toggle. */
const STEPPER_BUTTON =
  'grid h-8 w-8 place-items-center rounded-full text-foreground/50 outline-none transition-colors duration-200 hover:bg-primary/10 hover:text-primary focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-foreground/50';

/**
 * The cog's settings view (R116): "Back to day plan" and one row, Hours a
 * day. It saves on change; nothing here draws (R118, ADR-0019).
 */
export function HoursSettings({
  minutes,
  failed,
  onChange,
  onBack,
}: {
  /** The readout: the value being saved, else the saved one. */
  minutes: HoursAnswer;
  failed: boolean;
  onChange: (minutes: HoursAnswer) => void;
  onBack: () => void;
}) {
  // From unset, either button starts at the middle chip (R117).
  const step = (by: number) =>
    onChange(minutes === null ? HOURS_START : minutes + by);
  return (
    <div className="space-y-4 animate-in fade-in-0 slide-in-from-top-1 duration-200 motion-reduce:slide-in-from-top-0">
      <button
        type="button"
        onClick={onBack}
        className="flex items-center gap-1 text-base text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:text-foreground"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden
          className="h-4 w-4"
        >
          <path
            fillRule="evenodd"
            d="M9.707 14.707a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 1.414L7.414 9H15a1 1 0 110 2H7.414l2.293 2.293a1 1 0 010 1.414z"
            clipRule="evenodd"
          />
        </svg>
        Back to day plan
      </button>
      <div className={cn(QUIET_CARD, 'space-y-3')}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p
            id="day-plan-hours-label"
            className="text-base leading-6 text-foreground"
          >
            Hours a day
          </p>
          <div
            role="group"
            aria-labelledby="day-plan-hours-label"
            className="flex items-center gap-1 rounded-full border border-foreground/10 p-0.5"
          >
            <button
              type="button"
              onClick={() => step(-HOURS_STEP)}
              disabled={minutes !== null && minutes <= HOURS_MIN}
              aria-label="Less time"
              className={STEPPER_BUTTON}
            >
              <Minus strokeWidth={1.75} className="h-4 w-4" />
            </button>
            <output
              aria-labelledby="day-plan-hours-label"
              aria-live="polite"
              className="min-w-28 text-center text-sm tabular-nums text-foreground"
            >
              {minutes === null ? 'Typical day' : hoursInWords(minutes)}
            </output>
            <button
              type="button"
              onClick={() => step(HOURS_STEP)}
              disabled={minutes !== null && minutes >= HOURS_MAX}
              aria-label="More time"
              className={STEPPER_BUTTON}
            >
              <Plus strokeWidth={1.75} className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-sm text-foreground/45">
          <p>Applies the next time you plan your day.</p>
          {minutes !== null && (
            <button
              type="button"
              onClick={() => onChange(null)}
              className="underline-offset-4 outline-none transition-colors hover:text-foreground hover:underline focus-visible:text-foreground focus-visible:underline"
            >
              Use the default
            </button>
          )}
        </div>
        {failed && (
          <p aria-live="polite" className="text-sm text-foreground/50">
            Couldn&apos;t save. Try again.
          </p>
        )}
      </div>
    </div>
  );
}
