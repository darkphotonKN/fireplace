'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { X } from 'lucide-react';
import {
  createChecklistItem,
  fetchChecklist,
  updateChecklistItem,
} from '@/services/api';
import type { ChecklistItem } from '@/api/checklists';
import { ApiError } from '@/api/client';
import {
  getDayPlan,
  planMyDay,
  setItemRevisions,
  tickDayPlanEntry,
  untickDayPlanEntry,
  type DayPlan,
  type DayPlanEntry,
} from '@/api/insights';
import { readMilestones } from '@/lib/milestones';
import { resolveDayPlan, type ResolvedEntry } from '@/lib/dayPlan';
import { announcePlanItemsChanged } from '@/lib/planItemsChanged';
import { toDateOnly } from '@/lib/itemDates';
import { cn } from '@/lib/utils';

/**
 * The Day plan section (FS-02JBZ R62–R74): where the plan page's Daily card
 * was, the day's entries in up to three labelled groups. Grown from Today's
 * moves (A5), whose show/hide toggle, Add, dismiss and quiet retry carry over.
 *
 * Opening the page only READS the day plan (ADR-0019, D7). An unplanned day
 * shows Plan my day, and only that button (or Plan again) ever asks the
 * server to draw one.
 *
 * Self-contained: every entry points at the plan's long-term items, so the
 * section reads that list itself and acts on it through the existing item
 * endpoints. The rules for which entries still make sense live in
 * `lib/dayPlan`.
 *
 * The entry checkbox is the day tick, "done for today" (D4, R75). It never
 * sets the item's `done`; after a tick, an inline "Also mark '<item>' done?"
 * offers that separately (R77) and goes away on its own.
 *
 * On learning plans an entry may be a revision, labelled "Revision N" (R81),
 * and an existing item's entry carries a small "Don't revise" switch for that
 * item (R80). Both come from the server; the client never decides timing.
 *
 * The middle card's `<Todo>` keeps its own copy of the items, so every item
 * write here (Add, Also mark done) is announced through `planItemsChanged`
 * and the card re-reads.
 */

/** The key Today's moves used, so a hidden section stays hidden (R72). */
const SHOW_KEY = 'showDailyInsights';

const localDay = (): string => toDateOnly(new Date())!;

/** `gone` is a 404 (R70): the page's own "not here" state applies. */
type Status = 'loading' | 'ready' | 'failed' | 'gone';

/** The redraw cap is spent (R15). The client's `ErrorCode` list predates it. */
const REFRESH_LIMIT_REACHED = 'BRIEF_REFRESH_LIMIT_REACHED';

/** What the last Plan my day / Plan again said, until the next one. */
type PlanNotice = 'unchanged' | 'failed' | null;

/** The checklist's copy for a failed create, reused as is (R73). */
const ADD_FAILED = 'Failed to add suggested task. Please try again.';

/** A tick on an entry that can't be ticked as it stands (R26, R79). */
const NOT_TICKABLE = 'DAY_PLAN_ENTRY_NOT_TICKABLE';

/** A tick on an entry from a draw another tab replaced: the entry is gone. */
const NOT_FOUND = 'NOT_FOUND';

const MARK_DONE_FAILED = "Couldn't mark it done. Please try again.";

/** How long "Also mark '<item>' done?" stays (S12). */
const ASK_MS = 8_000;

/** The open "Also mark '<item>' done?": at most one, on the entry just ticked. */
type DoneAsk = { entryId: string; itemId: string; text: string };

export default function DayPlanSection({ planId }: { planId: string }) {
  const [shown, setShown] = useState(true);
  const [status, setStatus] = useState<Status>('loading');
  const [items, setItems] = useState<ChecklistItem[]>([]);
  const [dayPlan, setDayPlan] = useState<DayPlan | null>(null);
  // Per day and per page view: a reload brings dismissed entries back (S18).
  const [dismissed, setDismissed] = useState<ReadonlySet<string>>(new Set());
  // Entry id → the item Add created from it: the id its first tick binds (R76).
  const [added, setAdded] = useState<ReadonlyMap<string, string>>(new Map());
  const [adding, setAdding] = useState<ReadonlySet<string>>(new Set());
  // Entries with a tick or untick in flight: one request per entry at a time.
  const [ticking, setTicking] = useState<ReadonlySet<string>>(new Set());
  // Entries with a revisions switch in flight, likewise.
  const [switching, setSwitching] = useState<ReadonlySet<string>>(new Set());
  const [ask, setAsk] = useState<DoneAsk | null>(null);
  const [planning, setPlanning] = useState(false);
  const [notice, setNotice] = useState<PlanNotice>(null);
  const [error, setError] = useState<string | null>(null);

  // The date the day plan on screen belongs to, and a counter so a read that
  // lost a race never writes over a newer one.
  const dayRef = useRef<string>(localDay());
  const readRef = useRef(0);

  // Loaded in an effect, not the initializer: there is no storage during the
  // server render.
  useEffect(() => {
    try {
      const saved = window.localStorage.getItem(SHOW_KEY);
      if (saved !== null) setShown(saved === 'true');
    } catch {
      // Storage blocked: shown, and not remembered.
    }
  }, []);

  const toggleShown = () => {
    setAsk(null);
    const next = !shown;
    setShown(next);
    try {
      window.localStorage.setItem(SHOW_KEY, String(next));
    } catch {
      // As above.
    }
  };

  // Day plan and items together: an entry can't be judged without both.
  const load = useCallback(async () => {
    const read = ++readRef.current;
    const day = localDay();
    dayRef.current = day;
    setStatus('loading');
    try {
      const [nextPlan, nextItems] = await Promise.all([
        getDayPlan(planId, day),
        fetchChecklist(planId, 'longterm'),
      ]);
      if (read !== readRef.current) return;
      setDayPlan(nextPlan);
      setItems(nextItems);
      setStatus('ready');
    } catch (err) {
      if (read !== readRef.current) return;
      if (err instanceof ApiError && err.status === 404) {
        setStatus('gone');
        return;
      }
      // Quiet by design (R71): out-of-range dates and 503s land here too.
      console.error('Failed to load the day plan:', err);
      setStatus('failed');
    }
  }, [planId]);

  // A fresh read without the skeleton, for when the day plan on screen turned
  // out stale (R79). It yields to any read or draw started meanwhile.
  const refresh = useCallback(async () => {
    const read = readRef.current;
    try {
      const [nextPlan, nextItems] = await Promise.all([
        getDayPlan(planId, dayRef.current),
        fetchChecklist(planId, 'longterm'),
      ]);
      if (read !== readRef.current) return;
      setDayPlan(nextPlan);
      setItems(nextItems);
    } catch (err) {
      // Keep what is on screen; the next return to the page reads again.
      console.error('Failed to re-read the day plan:', err);
    }
  }, [planId]);

  // "Also mark done?" goes away on its own (S12).
  useEffect(() => {
    if (!ask) return;
    const timer = setTimeout(() => setAsk(null), ASK_MS);
    return () => clearTimeout(timer);
  }, [ask]);

  // A new day (or plan) starts clean: what was dismissed or added belonged to
  // the day plan it was done to.
  const startDay = useCallback(() => {
    setDismissed(new Set());
    setAdded(new Map());
    setAsk(null);
    setNotice(null);
    setError(null);
    load();
  }, [load]);

  useEffect(() => {
    startDay();
  }, [startDay]);

  // Coming back to the page: a new local day reads that day's plan (R64); the
  // same day re-reads the items only, so an item finished elsewhere renders
  // done. Focus and visibilitychange both fire on a tab switch; one read
  // covers both.
  useEffect(() => {
    let inFlight = false;
    const refresh = async () => {
      if (document.visibilityState === 'hidden' || inFlight) return;
      if (localDay() !== dayRef.current) {
        startDay();
        return;
      }
      const read = readRef.current;
      inFlight = true;
      try {
        const nextItems = await fetchChecklist(planId, 'longterm');
        if (read === readRef.current) setItems(nextItems);
      } catch {
        // Keep what is on screen; the next return tries again.
      } finally {
        inFlight = false;
      }
    };
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', refresh);
    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', refresh);
    };
  }, [planId, startDay]);

  const groups = useMemo(
    () =>
      resolveDayPlan(dayPlan?.groups ?? [], items)
        .map((g) => ({
          ...g,
          entries: g.entries.filter((e) => !dismissed.has(e.id)),
        }))
        .filter((g) => g.entries.length > 0),
    [dayPlan, items, dismissed]
  );
  const hasMilestones = useMemo(
    () => readMilestones(items).total > 0,
    [items]
  );

  const add = async (entry: ResolvedEntry) => {
    if (!entry.parentItemId || adding.has(entry.id)) return;
    setAsk(null);
    setAdding((prev) => new Set(prev).add(entry.id));
    setError(null);
    try {
      // A long-term child of the entry's parent, never a daily item (R73, R84).
      const created = await createChecklistItem(entry.text, planId, 'longterm', {
        type: 'task',
        parentId: entry.parentItemId,
      });
      setItems((prev) => [...prev, created]);
      setAdded((prev) => new Map(prev).set(entry.id, created.id));
      announcePlanItemsChanged(planId);
    } catch (err) {
      console.error('Failed to add day plan step:', err);
      setError(ADD_FAILED);
    } finally {
      setAdding((prev) => {
        const next = new Set(prev);
        next.delete(entry.id);
        return next;
      });
    }
  };

  const dismiss = (entry: ResolvedEntry) => {
    setAsk(null);
    setDismissed((prev) => new Set(prev).add(entry.id));
  };

  const setTickingOf = (entryId: string, on: boolean) =>
    setTicking((prev) => toggled(prev, entryId, on));

  const patchEntry = (entryId: string, patch: Partial<DayPlanEntry>) =>
    setDayPlan((prev) => prev && withEntry(prev, entryId, patch));

  // The day tick (R75), optimistic. A new step ticks with the item it was
  // added as (R76); the server binds the entry to it and returns it as an
  // existing item. `done` is left alone (R28).
  const tick = async (entry: ResolvedEntry) => {
    const bindTo =
      entry.kind === 'new_step'
        ? added.get(entry.id) ?? entry.addedItemId
        : undefined;
    const itemId = entry.itemId ?? bindTo;
    // No tick while a redraw is in flight: the redraw could drop it.
    if (!itemId || planning || ticking.has(entry.id)) return;

    setTickingOf(entry.id, true);
    patchEntry(entry.id, { ticked: true });
    const item = items.find((i) => i.id === itemId);
    // Replaces any open ask: ticking is the "other action" that drops it.
    setAsk(
      item && !item.done ? { entryId: entry.id, itemId, text: entry.text } : null
    );
    try {
      const next = await tickDayPlanEntry(entry.id, bindTo);
      patchEntry(entry.id, next);
    } catch (err) {
      patchEntry(entry.id, { ticked: false });
      setAsk((prev) => (prev?.entryId === entry.id ? null : prev));
      // Switch on the domain code, never on `detail` (ADR-0004).
      if (
        err instanceof ApiError &&
        (err.code === NOT_TICKABLE || err.code === NOT_FOUND)
      ) {
        // The plan moved on under the day plan (an item archived, deleted),
        // or another tab drew a new plan and this entry went with the old
        // one: quietly read what is true now (R79).
        refresh();
      } else {
        console.error('Failed to tick the day plan entry:', err);
      }
    } finally {
      setTickingOf(entry.id, false);
    }
  };

  const untick = async (entry: ResolvedEntry) => {
    if (planning || ticking.has(entry.id)) return;
    setAsk(null);
    setTickingOf(entry.id, true);
    patchEntry(entry.id, { ticked: false });
    try {
      await untickDayPlanEntry(entry.id);
    } catch (err) {
      patchEntry(entry.id, { ticked: true });
      console.error('Failed to untick the day plan entry:', err);
    } finally {
      setTickingOf(entry.id, false);
    }
  };

  // The "Don't revise" switch (R80), optimistic: it is the item's switch, so
  // the label follows it at once (an off switch cancels the open revision,
  // R47). Sent by item, but tracked per entry like a tick.
  const setRevisions = async (entry: ResolvedEntry, enabled: boolean) => {
    if (!entry.itemId || switching.has(entry.id)) return;
    setAsk(null);
    setSwitching((prev) => toggled(prev, entry.id, true));
    patchEntry(entry.id, { revisionsEnabled: enabled });
    try {
      await setItemRevisions(entry.itemId, planId, enabled);
    } catch (err) {
      patchEntry(entry.id, { revisionsEnabled: !enabled });
      console.error('Failed to switch revisions for the item:', err);
    } finally {
      setSwitching((prev) => toggled(prev, entry.id, false));
    }
  };

  // "Also mark done?" → Yes: the item's own `done`, through the existing item
  // update (R77), optimistic. The day tick stays as it is (R28).
  const markDone = async ({ itemId }: DoneAsk) => {
    setAsk(null);
    setError(null);
    const setDone = (done: boolean) =>
      setItems((prev) =>
        prev.map((i) => (i.id === itemId ? { ...i, done } : i))
      );
    setDone(true);
    try {
      const updated = await updateChecklistItem(itemId, { done: true }, planId);
      setItems((prev) => prev.map((i) => (i.id === itemId ? updated : i)));
      announcePlanItemsChanged(planId);
    } catch (err) {
      setDone(false);
      console.error('Failed to mark the item done:', err);
      setError(MARK_DONE_FAILED);
    }
  };

  // Plan my day and Plan again are the same call (R65, R68). The first draw of
  // the day is free; the server says what is left in `refreshesRemaining`.
  const plan = async () => {
    if (!dayPlan || planning || dayPlan.refreshesRemaining <= 0) return;
    // Past local midnight the plan on screen is yesterday's; today's is a
    // read, not a redraw of the old one.
    if (localDay() !== dayRef.current) {
      startDay();
      return;
    }
    const read = ++readRef.current;
    setPlanning(true);
    setNotice(null);
    setAsk(null);
    try {
      // The items are read again alongside, so a step created since the last
      // read resolves. Best effort: a counted draw is already spent, so a
      // failed item read must not throw its entries away.
      const [next, nextItems] = await Promise.all([
        planMyDay(planId, dayRef.current),
        fetchChecklist(planId, 'longterm').catch(() => null),
      ]);
      if (read !== readRef.current) return;
      if (nextItems) setItems(nextItems);
      if (isUnchangedRedraw(dayPlan, next)) {
        setNotice('unchanged');
      } else {
        // New entries replace the old; what was added or dismissed belonged
        // to entries that are gone.
        const kept = new Set(
          next.groups.flatMap((g) => g.entries.map((e) => e.id))
        );
        setAdded((prev) => keepOnlyKeys(prev, kept));
        setDismissed((prev) => keepOnly(prev, kept));
      }
      setDayPlan(next);
    } catch (err) {
      if (read !== readRef.current) return;
      // Switch on the domain code, never on `detail` (ADR-0004).
      if (err instanceof ApiError && err.code === REFRESH_LIMIT_REACHED) {
        // Another tab spent the last one; the stored plan is unchanged, so
        // the entries on screen stay and only the action retires (R68).
        setDayPlan((prev) => prev && { ...prev, refreshesRemaining: 0 });
        return;
      }
      // Quiet, like a failed read (R71): the entries stay and the stored plan
      // is untouched, so retrying is correct (a 503 spends nothing).
      console.error('Failed to plan the day:', err);
      setNotice('failed');
    } finally {
      // Unconditional: a read that superseded this draw must not leave the
      // action stuck. Only one draw is ever in flight.
      setPlanning(false);
    }
  };

  if (status === 'gone') return null;

  const isPlanned = dayPlan?.status === 'planned';
  const isBasic = isPlanned && dayPlan?.source === 'rules';

  return (
    <section aria-labelledby="day-plan-heading">
      <div className="flex justify-between items-center mb-4">
        <div className="flex items-center gap-2">
          <h2 id="day-plan-heading" className="text-lg font-semibold">
            Day plan
          </h2>
          <DayPlanInfo />
        </div>
        <button
          type="button"
          onClick={toggleShown}
          className="text-sm text-muted-foreground hover:text-gray-700 dark:hover:text-gray-300"
        >
          {shown ? 'Hide' : 'Show'}
        </button>
      </div>

      {shown && (
        <div className="space-y-3">
          {error && (
            <div className="p-2 text-base text-red-600 bg-red-50 rounded-md">
              {error}
            </div>
          )}

          {status === 'loading' ? (
            <DayPlanSkeleton label="Loading the day plan" />
          ) : status === 'failed' || !dayPlan ? (
            <p className="flex items-center gap-2 text-sm text-muted-foreground">
              Couldn&apos;t load the day plan.
              <button
                type="button"
                onClick={load}
                className="underline-offset-4 hover:underline hover:text-foreground"
              >
                Try again
              </button>
            </p>
          ) : !isPlanned ? (
            planning ? (
              <DayPlanSkeleton label="Planning your day" />
            ) : (
              <NotPlanned
                failed={notice === 'failed'}
                retired={dayPlan.refreshesRemaining <= 0}
                onPlan={plan}
              />
            )
          ) : (
            <>
              {isBasic && (
                <p className="text-sm text-muted-foreground">
                  <span className="mr-2 rounded bg-foreground/10 px-1.5 py-0.5 text-xs font-medium text-foreground">
                    Basic plan
                  </span>
                  The AI wasn&apos;t available, so this plan was made by
                  simple rules.
                </p>
              )}
              {groups.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {hasMilestones
                    ? 'Nothing to plan for today.'
                    : 'Start a milestone in your plan and the day plan will help you push it forward.'}
                </p>
              ) : (
                <div className="space-y-4">
                  {groups.map((group, i) => (
                    <div key={`${i}-${group.label}`}>
                      <h3 className="mb-2 text-sm font-medium text-muted-foreground">
                        {group.label}
                      </h3>
                      <ul className="space-y-3">
                        {group.entries.map((entry) => (
                          <EntryRow
                            key={entry.id}
                            entry={entry}
                            added={
                              added.has(entry.id) || !!entry.addedItemId
                            }
                            adding={adding.has(entry.id)}
                            ticking={ticking.has(entry.id)}
                            planning={planning}
                            switching={switching.has(entry.id)}
                            ask={ask?.entryId === entry.id ? ask : null}
                            onAdd={() => add(entry)}
                            onTick={() =>
                              entry.ticked ? untick(entry) : tick(entry)
                            }
                            onMarkDone={markDone}
                            onRevisions={(enabled) =>
                              setRevisions(entry, enabled)
                            }
                            onDismiss={() => dismiss(entry)}
                          />
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              )}
              <PlanBar
                label={isBasic ? 'Plan my day' : 'Plan again'}
                remaining={dayPlan.refreshesRemaining}
                planning={planning}
                notice={notice}
                onPlan={plan}
              />
            </>
          )}
        </div>
      )}
    </section>
  );
}

/**
 * The "i" tooltip (R63, D11): "your daily progression". Same markup as the
 * plan-focus tooltip in the plan layout. It opens downward over the section,
 * so the layout lifts this card above the middle card (see its comment).
 */
function DayPlanInfo() {
  return (
    <div className="relative group">
      <button
        type="button"
        className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
        aria-label="Information about the day plan"
        aria-describedby="day-plan-info"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="w-4 h-4"
        >
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z"
            clipRule="evenodd"
          />
        </svg>
      </button>
      <div
        id="day-plan-info"
        role="tooltip"
        className="absolute left-0 top-full mt-2 w-64 opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none z-50"
      >
        <div className="bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl">
          <p className="text-xs text-white mb-1">Your daily progression</p>
          <p className="text-xs text-gray-400">
            Plan your day, then tick what you did today. What&apos;s left, and
            anything due for revision, comes back on its own.
          </p>
        </div>
        <div className="absolute left-1 -top-1 w-2 h-2 bg-gray-900 border-l border-t border-gray-700 transform rotate-45"></div>
      </div>
    </div>
  );
}

function NotPlanned({
  failed,
  retired,
  onPlan,
}: {
  failed: boolean;
  retired: boolean;
  onPlan: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p
        aria-live="polite"
        className="flex items-center gap-2 text-sm text-muted-foreground"
      >
        {failed ? (
          <PlanFailed onRetry={onPlan} />
        ) : (
          "You haven't planned today yet."
        )}
      </p>
      <button
        type="button"
        onClick={onPlan}
        disabled={retired}
        className="shrink-0 text-sm px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 dark:hover:bg-gray-700/50 disabled:opacity-50"
        style={{ color: 'rgb(247, 111, 83)' }}
      >
        Plan my day
      </button>
    </div>
  );
}

/** A failed draw, said quietly (R71): the stored plan is untouched. */
function PlanFailed({ onRetry }: { onRetry: () => void }) {
  return (
    <>
      Couldn&apos;t plan your day.
      <button
        type="button"
        onClick={onRetry}
        className="underline-offset-4 hover:underline hover:text-foreground"
      >
        Try again
      </button>
    </>
  );
}

function EntryRow({
  entry,
  added,
  adding,
  ticking,
  planning,
  switching,
  ask,
  onAdd,
  onTick,
  onMarkDone,
  onRevisions,
  onDismiss,
}: {
  entry: ResolvedEntry;
  added: boolean;
  adding: boolean;
  ticking: boolean;
  /** A redraw is in flight: day ticks wait for it (it could drop them). */
  planning: boolean;
  switching: boolean;
  ask: DoneAsk | null;
  onAdd: () => void;
  onTick: () => void;
  onMarkDone: (ask: DoneAsk) => void;
  onRevisions: (enabled: boolean) => void;
  onDismiss: () => void;
}) {
  // Only an entry about a real item can be ticked (D3): an existing item, or
  // a new step once it has been added.
  const tickable = entry.kind === 'existing_item' || added;
  return (
    <li className="flex items-start gap-3">
      {tickable ? (
        <input
          type="checkbox"
          checked={entry.ticked}
          onChange={onTick}
          disabled={ticking || planning}
          aria-label={`Done for today: ${entry.text}`}
          title="Done for today"
          className="mt-1 h-4 w-4 shrink-0 rounded border-gray-300 dark:border-gray-600"
          style={{ accentColor: 'rgb(247, 111, 83)' }}
        />
      ) : (
        // Keeps the text column aligned with tickable rows.
        <span aria-hidden className="h-4 w-4 shrink-0" />
      )}
      <div className="min-w-0 flex-1">
        {entry.milestoneTitle && (
          <p className="text-xs uppercase tracking-wide text-muted-foreground">
            {entry.milestoneTitle}
          </p>
        )}
        <p className="flex flex-wrap items-baseline gap-x-2">
          <span
            className={cn(
              'text-base text-foreground',
              entry.done && 'line-through text-foreground/50'
            )}
          >
            {entry.text}
          </span>
          {entry.revision !== undefined && (
            <span className="whitespace-nowrap rounded bg-foreground/10 px-1.5 py-0.5 text-xs font-medium text-foreground">
              {`Revision ${entry.revision}`}
            </span>
          )}
        </p>
        <p className="text-sm text-muted-foreground">
          {entry.reason}
          {entry.minutes !== undefined && (
            <span className="ml-2 whitespace-nowrap text-xs">
              ~{entry.minutes} min
            </span>
          )}
        </p>
        {entry.kind === 'existing_item' &&
          entry.revisionsEnabled !== undefined && (
            <DontRevise
              text={entry.text}
              off={!entry.revisionsEnabled}
              disabled={switching}
              onChange={(off) => onRevisions(!off)}
            />
          )}
        {ask && (
          // Inline and quiet, never a modal (D4): it names the item, whatever
          // level it sits at, and goes away on its own.
          <p
            aria-live="polite"
            className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground"
          >
            <span>{`Also mark '${ask.text}' done?`}</span>
            <button
              type="button"
              onClick={() => onMarkDone(ask)}
              disabled={planning}
              className="underline underline-offset-4 hover:text-foreground"
            >
              Yes
            </button>
          </p>
        )}
      </div>
      {entry.kind === 'new_step' &&
        (added ? (
          <span className="shrink-0 text-sm text-muted-foreground">Added</span>
        ) : (
          <button
            type="button"
            onClick={onAdd}
            disabled={adding}
            aria-label={`Add ${entry.text}`}
            className="shrink-0 text-sm px-2 py-1 rounded bg-white/5 hover:bg-white/10 dark:hover:bg-gray-700/50 disabled:opacity-50"
            style={{ color: 'rgb(247, 111, 83)' }}
          >
            Add
          </button>
        ))}
      <button
        type="button"
        onClick={onDismiss}
        aria-label={`Dismiss ${entry.text}`}
        className="shrink-0 mt-[2px] text-muted-foreground hover:text-foreground transition-colors"
      >
        <X strokeWidth={1.75} className="h-4 w-4" />
      </button>
    </li>
  );
}

/**
 * The "Don't revise" switch (R80): a small toggle, on when the item's
 * revisions are switched OFF, so the wording reads as what it does.
 */
function DontRevise({
  text,
  off,
  disabled,
  onChange,
}: {
  text: string;
  off: boolean;
  disabled: boolean;
  onChange: (off: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={off}
      aria-label={`Don't revise: ${text}`}
      onClick={() => onChange(!off)}
      disabled={disabled}
      className="mt-1 inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground disabled:opacity-50"
    >
      <span
        aria-hidden
        className={cn(
          'relative inline-block h-3 w-5 rounded-full transition-colors',
          off ? 'bg-[rgb(247,111,83)]' : 'bg-foreground/20'
        )}
      >
        <span
          className={cn(
            'absolute top-0.5 h-2 w-2 rounded-full bg-white transition-all',
            off ? 'left-2.5' : 'left-0.5'
          )}
        />
      </span>
      Don&apos;t revise
    </button>
  );
}

/**
 * Whether a draw came back as the plan already on screen, i.e. the rule gate
 * served the stored plan and spent nothing (R17, R68).
 *
 * The response carries no "gate served" flag, so this compares `generatedAt`:
 * a draw that reached the model was made at a new instant. Chosen over
 * "`refreshesRemaining` did not drop" because that count is shared by every
 * tab, so a count read earlier can be stale, while `generatedAt` names the
 * exact draw this tab is showing.
 */
function isUnchangedRedraw(onScreen: DayPlan, next: DayPlan): boolean {
  return (
    onScreen.status === 'planned' && next.generatedAt === onScreen.generatedAt
  );
}

/** `ids` with `id` added (`on`) or removed. */
const toggled = (
  ids: ReadonlySet<string>,
  id: string,
  on: boolean
): ReadonlySet<string> => {
  const next = new Set(ids);
  if (on) next.add(id);
  else next.delete(id);
  return next;
};

const keepOnly = (
  ids: ReadonlySet<string>,
  kept: ReadonlySet<string>
): ReadonlySet<string> => new Set([...ids].filter((id) => kept.has(id)));

const keepOnlyKeys = <V,>(
  map: ReadonlyMap<string, V>,
  kept: ReadonlySet<string>
): ReadonlyMap<string, V> =>
  new Map([...map].filter(([id]) => kept.has(id)));

/** The day plan with one entry patched, wherever it sits. */
function withEntry(
  plan: DayPlan,
  entryId: string,
  patch: Partial<DayPlanEntry>
): DayPlan {
  return {
    ...plan,
    groups: plan.groups.map((g) => ({
      ...g,
      entries: g.entries.map((e) => (e.id === entryId ? { ...e, ...patch } : e)),
    })),
  };
}

function PlanBar({
  label,
  remaining,
  planning,
  notice,
  onPlan,
}: {
  label: string;
  remaining: number;
  planning: boolean;
  notice: PlanNotice;
  onPlan: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
      <p aria-live="polite" className="flex min-w-0 items-center gap-2">
        {notice === 'unchanged' &&
          'Nothing has changed since this plan was made.'}
        {notice === 'failed' && <PlanFailed onRetry={onPlan} />}
      </p>
      <button
        type="button"
        onClick={onPlan}
        // Retired at zero (R68): the count is on the day plan, so no request
        // is needed to find out.
        disabled={planning || remaining <= 0}
        className="shrink-0 underline-offset-4 hover:underline hover:text-foreground disabled:opacity-50 disabled:no-underline disabled:hover:text-muted-foreground"
      >
        {planning ? 'Planning…' : label} · {remaining} left
      </button>
    </div>
  );
}

function DayPlanSkeleton({ label }: { label: string }) {
  return (
    <ul
      data-testid="day-plan-skeleton"
      aria-busy="true"
      aria-label={label}
      className="space-y-3"
    >
      {[0, 1, 2].map((i) => (
        <li key={i} className="space-y-1.5 animate-pulse">
          <div className="h-3 w-24 rounded bg-foreground/10" />
          <div className="h-4 w-3/4 rounded bg-foreground/10" />
          <div className="h-3 w-1/2 rounded bg-foreground/5" />
        </li>
      ))}
    </ul>
  );
}
