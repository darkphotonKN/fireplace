'use client';

import {
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
  type Ref,
} from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { CalendarDays, Plus } from 'lucide-react';
import {
  archiveChecklistItem,
  createChecklistItem,
  deleteChecklistItem,
  fetchChecklist,
  updateChecklistDates,
  updateChecklistItem,
} from '@/services/api';
import { ApiError } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';
import { Checkbox } from '@/components/ui/checkbox';
import ItemDateChip from '@/components/ItemDateChip';
import ItemActions, { type ItemActionsHandle } from '@/components/ItemActions';
import DeleteConfirm from '@/components/DeleteConfirm';
import CompletionNudge, { useCompletionNudges } from '@/components/CompletionNudge';
import ProgressMeter from '@/components/ProgressMeter';
import { cn } from '@/lib/utils';
import {
  findMilestone,
  nextStep,
  type Milestone,
  type MilestoneDetail,
} from '@/lib/milestones';

/**
 * A milestone's own page (FS-043WP): the plan's middle card, given over to one
 * milestone. Read out of the plan's long-term list, the list the grid reads,
 * through `findMilestone`, so the page, the card and the strip can never
 * disagree about it (R10). It reads and writes only through the plan's
 * checklist routes and so serves whoever those routes serve (R36).
 *
 * Laid out in the order R11 gives, each block its own piece so the editing
 * slices drop in beside them.
 */

export interface MilestonePageProps {
  planId: string;
  milestoneId: string;
}

/**
 * The list the page was read from, not just the milestone: steps are items of
 * the same list, so edits land in one place and the reading follows.
 */
type Load =
  | { state: 'loading' }
  | { state: 'ready'; items: ChecklistItem[] }
  | { state: 'missing' }
  | { state: 'failed' };

/**
 * The plan can't be reached (no access, deleted) or its id is malformed: to the
 * person, the milestone simply isn't here (R33). Anything else may pass, so it
 * is offered again rather than called gone (R34).
 */
const NOT_HERE_STATUSES = new Set([403, 404, 422]);
const isNotHere = (err: unknown) =>
  err instanceof ApiError && NOT_HERE_STATUSES.has(err.status);

const isNote = (item: ChecklistItem) => (item.type ?? 'task') === 'note';

/** What the page can do to its milestone and steps; each one optimistic (R32). */
interface MilestoneActions {
  toggleDone: (id: string) => void;
  rename: (id: string, text: string) => void;
  /** Why it matters: empty clears it, unchanged sends nothing (R23). */
  setIntention: (id: string, text: string) => void;
  /** Resolves true once the step is in, false once the failure is shown. */
  addStep: (text: string) => Promise<boolean>;
  /** Both ends together: null clears that end. */
  setDates: (id: string, startDate: string | null, dueDate: string | null) => void;
  /**
   * Not optimistic: the page leaves once these land, and a milestone taken
   * out from under it first would flash "not here" on the way (R18). Each
   * resolves true once done, false once the failure is shown.
   */
  archive: (id: string) => Promise<boolean>;
  remove: (id: string) => Promise<boolean>;
}

/**
 * The page's milestone: loaded from the plan's long-term list and kept there,
 * so every edit is a change to that one list and the milestone reading, the
 * next step and the nudge all follow from it. Mutations go through the same
 * checklist calls the plan's lists use, optimistic where those are, and
 * revert with the same error text (R32).
 */
function useMilestone(planId: string, milestoneId: string) {
  const [load, setLoad] = useState<Load>({ state: 'loading' });
  const [error, setError] = useState<string | null>(null);
  // Bumped by Try again; the load below re-runs on it.
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoad({ state: 'loading' });
    fetchChecklist(planId, 'longterm').then(
      (items) => !cancelled && setLoad({ state: 'ready', items }),
      (err) => {
        if (cancelled) return;
        const missing = isNotHere(err);
        if (!missing) console.error('Failed to load milestone:', err);
        setLoad({ state: missing ? 'missing' : 'failed' });
      }
    );
    return () => {
      cancelled = true;
    };
  }, [planId, milestoneId, attempt]);

  const detail = useMemo(
    () =>
      load.state === 'ready'
        ? findMilestone(load.items, milestoneId, planId)
        : null,
    [load, milestoneId, planId]
  );

  const setItems = (change: (items: ChecklistItem[]) => ChecklistItem[]) =>
    setLoad((prev) =>
      prev.state === 'ready' ? { state: 'ready', items: change(prev.items) } : prev
    );
  const patch = (id: string, fields: Partial<ChecklistItem>) =>
    setItems((items) =>
      items.map((i) => (i.id === id ? { ...i, ...fields } : i))
    );
  const itemOf = (id: string) =>
    load.state === 'ready' ? load.items.find((i) => i.id === id) : undefined;

  const actions: MilestoneActions = {
    toggleDone: async (id) => {
      const item = itemOf(id);
      if (!item) return;
      const was = item.done;
      setError(null);
      patch(id, { done: !was });
      try {
        await updateChecklistItem(id, { done: !was }, planId, 'longterm');
      } catch (err) {
        console.error('Error toggling item:', err);
        patch(id, { done: was });
        setError('Failed to update task status. Please try again.');
      }
    },
    rename: async (id, text) => {
      const item = itemOf(id);
      const next = text.trim();
      // Empty keeps the old text; unchanged has nothing to send (R30).
      if (!item || !next || next === item.description) return;
      const was = item.description;
      setError(null);
      patch(id, { description: next });
      try {
        await updateChecklistItem(id, { description: next }, planId, 'longterm');
      } catch (err) {
        console.error('Error renaming item:', err);
        patch(id, { description: was });
        setError('Failed to update task. Please try again.');
      }
    },
    setIntention: async (id, text) => {
      const item = itemOf(id);
      const next = text.trim();
      if (!item || next === (item.intention ?? '')) return;
      const was = item.intention;
      setError(null);
      patch(id, { intention: next || undefined });
      try {
        // Sent as typed-and-trimmed: the server clears it on empty (R38).
        await updateChecklistItem(id, { intention: next }, planId, 'longterm');
      } catch (err) {
        console.error('Error saving why it matters:', err);
        patch(id, { intention: was });
        setError('Failed to update task. Please try again.');
      }
    },
    // Not optimistic, as the lists' step add isn't: the step needs its id.
    addStep: async (text) => {
      setError(null);
      try {
        const created = await createChecklistItem(text, planId, 'longterm', {
          type: 'task',
          parentId: milestoneId,
        });
        // The end of the list is the end of this milestone's steps (G9).
        setItems((items) => [...items, created]);
        return true;
      } catch (err) {
        console.error('Failed to add step:', err);
        setError('Failed to add step. Please try again.');
        return false;
      }
    },
    setDates: async (id, startDate, dueDate) => {
      const before = itemOf(id);
      if (!before) return;
      setError(null);
      patch(id, {
        startDate: startDate ?? undefined,
        dueDate: dueDate ?? undefined,
      });
      try {
        await updateChecklistDates(planId, id, { startDate, dueDate });
      } catch (err) {
        console.error('Error setting item dates:', err);
        // Only the dates go back; a rename or tick made meanwhile stays.
        patch(id, { startDate: before.startDate, dueDate: before.dueDate });
        setError('Failed to save the dates. Please try again.');
      }
    },
    archive: async (id) => {
      setError(null);
      try {
        await archiveChecklistItem(id, planId, 'longterm');
        return true;
      } catch (err) {
        console.error('Error archiving todo:', err);
        setError('Failed to archive task. Please try again.');
        return false;
      }
    },
    remove: async (id) => {
      setError(null);
      try {
        await deleteChecklistItem(id, planId);
        return true;
      } catch (err) {
        console.error('Error deleting task:', err);
        setError('Failed to delete task');
        return false;
      }
    },
  };

  return {
    load,
    detail,
    error,
    actions,
    retry: () => setAttempt((n) => n + 1),
  };
}

export default function MilestonePage({ planId, milestoneId }: MilestonePageProps) {
  const { load, detail, error, actions, retry } = useMilestone(planId, milestoneId);
  const router = useRouter();

  const planHref = `/plan/${planId}`;
  return (
    <div className="space-y-4">
      <BackToPlan href={planHref} />
      {error && (
        <div className="rounded-md bg-red-50 p-2 text-base text-red-600">
          {error}
        </div>
      )}
      {load.state === 'loading' ? (
        <p className="py-4 text-sm text-muted-foreground">Loading milestone…</p>
      ) : load.state === 'failed' ? (
        <LoadFailed onRetry={retry} />
      ) : detail ? (
        <MilestoneView
          detail={detail}
          actions={actions}
          onLeave={() => router.push(planHref)}
        />
      ) : (
        <NotHere planHref={planHref} />
      )}
    </div>
  );
}

/**
 * The milestone itself, in the order R11 gives: title, dates, progress, why it
 * matters, then the next-step slot (the nudge stands in it once every step is
 * done, R28), then every step and the step composer.
 */
function MilestoneView({
  detail,
  actions,
  onLeave,
}: {
  detail: MilestoneDetail<ChecklistItem>;
  actions: MilestoneActions;
  /** Archived or deleted: nothing is left here to show (R18, C11, G7). */
  onLeave: () => void;
}) {
  const { milestone, item, steps } = detail;
  useDocumentTitle(milestone.title);
  const nudges = useCompletionNudges([milestone]);
  const [composing, setComposing] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const titleRef = useRef<InlineTextHandle>(null);
  const menuRef = useRef<ItemActionsHandle>(null);
  // A closed milestone has no next step; its closed state speaks (G11).
  const next = milestone.closed ? null : nextStep(steps);

  return (
    <div className="space-y-6">
      <header className="space-y-2">
        <div className="flex items-start gap-3">
          {/* Hand-settable at any progress; closing stays here, shown closed
              and reopenable (R16, C10). */}
          <span className="mt-[5px] shrink-0">
            <Checkbox
              checked={milestone.closed}
              onCheckedChange={() => actions.toggleDone(milestone.id)}
              aria-label={`Mark ${milestone.title} done`}
            />
          </span>
          <h2
            className={cn(
              'flex min-w-0 flex-1 text-xl font-semibold leading-8',
              milestone.closed && 'line-through text-foreground/50'
            )}
          >
            <InlineText
              ref={titleRef}
              text={milestone.title}
              onSave={(text) => actions.rename(milestone.id, text)}
            />
          </h2>
          {/* The card's menu less what would make the page's subject
              something else: no flip, no move, no indent (R17, G6). */}
          <span className="mt-1 shrink-0">
            <ItemActions
              ref={menuRef}
              item={item}
              onEdit={() => titleRef.current?.edit()}
              onArchive={async () => {
                if (await actions.archive(milestone.id)) onLeave();
              }}
              onDelete={() => setConfirmingDelete(true)}
              onSetDates={(start, due) => actions.setDates(milestone.id, start, due)}
              triggerClassName="opacity-100"
            />
          </span>
        </div>
        <MilestoneDates item={item} onPick={() => menuRef.current?.openDates()} />
      </header>

      {confirmingDelete && (
        <DeleteConfirm
          description={milestone.title}
          onCancel={() => setConfirmingDelete(false)}
          onConfirm={async () => {
            setConfirmingDelete(false);
            if (await actions.remove(milestone.id)) onLeave();
          }}
        />
      )}

      <Progress done={milestone.steps.done} total={milestone.steps.total} />

      <WhyItMatters
        text={item.intention ?? ''}
        onSave={(text) => actions.setIntention(milestone.id, text)}
      />

      <NextStepSlot
        milestone={milestone}
        next={next}
        nudging={nudges.shows(milestone)}
        onToggle={actions.toggleDone}
        onClose={() => actions.toggleDone(milestone.id)}
        onDismiss={() => nudges.dismiss(milestone)}
        onInvite={() => setComposing(true)}
      />

      <div className="space-y-1">
        <StepList
          steps={steps}
          nextId={next?.id ?? null}
          onToggle={actions.toggleDone}
          onRename={actions.rename}
        />
        <AddStep
          title={milestone.title}
          open={composing}
          onOpenChange={setComposing}
          onAdd={actions.addStep}
        />
      </div>
    </div>
  );
}

/** At most this many characters, as the server holds it (C5, R38). */
const INTENTION_MAX = 280;

/**
 * Why the milestone matters, in its owner's words, or a quiet question when
 * there are none yet (R22). One paragraph: Enter saves rather than breaking a
 * line, and typing stops at the server's limit (R23, G10). Only ever here, on
 * the page; no card, row or composer shows or asks for it (R25).
 */
function WhyItMatters({
  text,
  onSave,
}: {
  text: string;
  onSave: (text: string) => void;
}) {
  return (
    <section aria-label="Why it matters" className="flex">
      <InlineText
        text={text}
        onSave={onSave}
        prompt="Why does this matter?"
        label="Why does this matter?"
        maxLength={INTENTION_MAX}
        className="text-base leading-6 text-foreground/75"
      />
    </section>
  );
}

/**
 * The one thing to do next, on its own (R26–R28). With no open task step it
 * holds the completion nudge when every step is done, or an invitation to add
 * the first step when there are none (notes don't count, G12). A closed
 * milestone, or a nudge waved off, leaves it empty.
 */
function NextStepSlot({
  milestone,
  next,
  nudging,
  onToggle,
  onClose,
  onDismiss,
  onInvite,
}: {
  milestone: Milestone;
  next: ChecklistItem | null;
  nudging: boolean;
  onToggle: (id: string) => void;
  onClose: () => void;
  onDismiss: () => void;
  onInvite: () => void;
}) {
  const invite = !milestone.closed && milestone.steps.total === 0;
  if (!next && !nudging && !invite) return null;

  return (
    <section aria-label="Next step">
      {next ? (
        <div className="rounded-xl bg-primary/[0.05] px-4 py-3">
          <p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-primary/70">
            Next step
          </p>
          <div className="flex items-start gap-3">
            <span className="mt-[3px] shrink-0">
              <Checkbox
                checked={next.done}
                onCheckedChange={() => onToggle(next.id)}
                aria-label={`Mark ${next.description} done`}
              />
            </span>
            <span className="text-base leading-6 text-foreground">
              {next.description}
            </span>
          </div>
        </div>
      ) : nudging ? (
        <CompletionNudge
          title={milestone.title}
          onClose={onClose}
          onDismiss={onDismiss}
        />
      ) : (
        <button
          type="button"
          onClick={onInvite}
          className="flex w-full items-center gap-3 rounded-xl border border-dashed border-foreground/[0.12] px-4 py-3 text-left text-base text-foreground/55 outline-none transition-colors hover:border-primary/40 hover:text-primary focus-visible:border-primary/40 focus-visible:text-primary"
        >
          <Plus aria-hidden strokeWidth={1.75} className="h-4 w-4 shrink-0" />
          Add the first step
        </button>
      )}
    </section>
  );
}

/**
 * The dates, or a quiet "Add dates" where there are none; either opens the
 * menu's own date picker, so there is one picker, not two (R19).
 */
function MilestoneDates({
  item,
  onPick,
}: {
  item: ChecklistItem;
  onPick: () => void;
}) {
  // The list's items carry no scheduledTime (the backend dropped it), so the
  // chip has a date to show exactly when one of these is set.
  const dated = Boolean(item.startDate || item.dueDate);
  return (
    <button
      type="button"
      onClick={onPick}
      title={dated ? 'Change dates' : undefined}
      className="flex w-fit items-center gap-1 text-sm text-foreground/40 outline-none transition-colors hover:text-primary focus-visible:text-primary"
    >
      {dated ? (
        <ItemDateChip item={item} />
      ) : (
        <>
          <CalendarDays aria-hidden strokeWidth={1.75} className="h-3.5 w-3.5" />
          Add dates
        </>
      )}
    </button>
  );
}

/** The tab is named after the milestone while it is open, so a bookmark is too (R12). */
function useDocumentTitle(title: string) {
  useEffect(() => {
    const previous = document.title;
    document.title = title;
    return () => {
      document.title = previous;
    };
  }, [title]);
}

/** The Settings back button, pointed at the plan's URL rather than history (R9, G13). */
function BackToPlan({ href }: { href: string }) {
  return (
    <Link
      href={href}
      className="flex w-fit items-center text-base text-muted-foreground hover:text-gray-700 dark:hover:text-gray-300"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden
        className="mr-1 h-4 w-4"
      >
        <path
          fillRule="evenodd"
          d="M9.707 14.707a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 1.414L7.414 9H15a1 1 0 110 2H7.414l2.293 2.293a1 1 0 010 1.414z"
          clipRule="evenodd"
        />
      </svg>
      Back to plan
    </Link>
  );
}

/** Calm, never a crash or a blank card: the id is no milestone of this plan (C8). */
function NotHere({ planHref }: { planHref: string }) {
  return (
    <div className="space-y-2 py-6 text-center">
      <p className="text-base font-medium">This milestone isn&apos;t here</p>
      <p className="text-sm text-muted-foreground">
        It may have been archived, removed, or turned into something else.
      </p>
      <Link
        href={planHref}
        className="inline-block text-sm text-primary transition-colors hover:text-primary/80"
      >
        Return to the plan
      </Link>
    </div>
  );
}

/** A blip, not a verdict: say so and offer the load again (R34, G3). */
function LoadFailed({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="space-y-3 py-6 text-center">
      <p className="text-base font-medium">Couldn&apos;t load this milestone</p>
      <button
        type="button"
        onClick={onRetry}
        className="rounded-full px-3.5 py-1.5 text-sm font-medium text-primary ring-1 ring-primary/30 transition-colors hover:bg-primary/10"
      >
        Try again
      </button>
    </div>
  );
}

/** The card's meter, a touch heavier (R20); nothing to show with no task steps. */
function Progress({ done, total }: { done: number; total: number }) {
  if (total === 0) return null;
  return (
    <div data-milestone-progress className="flex items-center gap-3">
      <ProgressMeter done={done} total={total} className="h-1.5" />
    </div>
  );
}

/**
 * Every live step in stored order: tasks with their tick, notes as notes
 * (R29). Ticking and renaming are all a step can do here (R30, G8); the next
 * step is marked where it sits (R27).
 */
function StepList({
  steps,
  nextId,
  onToggle,
  onRename,
}: {
  steps: ChecklistItem[];
  nextId: string | null;
  onToggle: (id: string) => void;
  onRename: (id: string, text: string) => void;
}) {
  if (steps.length === 0) return null;
  return (
    <ul aria-label="Steps" className="space-y-1 border-t border-foreground/[0.07] pt-4">
      {steps.map((step) => {
        const note = isNote(step);
        const isNext = step.id === nextId;
        return (
          <li
            key={step.id}
            aria-current={isNext ? 'step' : undefined}
            className={cn(
              '-mx-2 flex items-start gap-3 rounded-md px-2 py-1.5',
              isNext && 'bg-primary/[0.05]'
            )}
          >
            {note ? (
              <span
                aria-hidden
                className="mt-[9px] h-1 w-1 shrink-0 rounded-full bg-foreground/30"
              />
            ) : (
              <span className="mt-[3px] shrink-0">
                <Checkbox
                  checked={step.done}
                  onCheckedChange={() => onToggle(step.id)}
                  aria-label={`Mark ${step.description} done`}
                />
              </span>
            )}
            <InlineText
              text={step.description}
              onSave={(text) => onRename(step.id, text)}
              className={cn(
                'text-base leading-6',
                note
                  ? 'italic text-foreground/60'
                  : step.done
                    ? 'text-foreground/40 line-through'
                    : 'text-foreground/85'
              )}
            />
          </li>
        );
      })}
    </ul>
  );
}

/** Starts a rename from outside the text, as the menu's Rename does (R17). */
interface InlineTextHandle {
  edit: () => void;
}

/**
 * Text that edits in place: click it, or Enter or Space on it, then Enter or
 * blur saves and Esc cancels (R15, R23, R30, US29). At rest it is a button
 * named by what it shows, so a heading still reads as its title; the input
 * carries `label`. An edit ended from the keyboard hands focus back to the
 * text. Whether an empty or unchanged save goes anywhere is the caller's to
 * decide; this only reports what was typed. With no text, a `prompt` stands
 * in quietly and is used the same way.
 */
function InlineText({
  text,
  onSave,
  className,
  prompt,
  label = `Rename ${text}`,
  maxLength,
  ref,
}: {
  text: string;
  onSave: (text: string) => void;
  className?: string;
  prompt?: string;
  label?: string;
  maxLength?: number;
  ref?: Ref<InlineTextHandle>;
}) {
  const [draft, setDraft] = useState<string | null>(null);
  // Esc unmounts the input; a blur on the way out must not save it.
  const cancelled = useRef(false);
  // Enter or Esc ends the edit from the keyboard: focus returns to the text.
  // A blur means focus already went somewhere else, so it is left there.
  const refocus = useRef(false);
  const restRef = useRef<HTMLButtonElement>(null);
  const edit = () => {
    cancelled.current = false;
    refocus.current = false;
    setDraft(text);
  };
  useImperativeHandle(ref, () => ({ edit }));

  useEffect(() => {
    if (draft === null && refocus.current) {
      refocus.current = false;
      restRef.current?.focus();
    }
  }, [draft]);

  if (draft === null) {
    return (
      <button
        ref={restRef}
        type="button"
        onClick={edit}
        className={cn(
          className,
          'min-w-0 flex-1 cursor-text text-left rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50',
          !text && prompt && 'italic text-foreground/40 hover:text-primary'
        )}
      >
        {text || prompt}
      </button>
    );
  }

  const commit = () => {
    if (cancelled.current) return;
    cancelled.current = true;
    setDraft(null);
    onSave(draft);
  };
  return (
    <input
      autoFocus
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          refocus.current = true;
          commit();
        }
        if (e.key === 'Escape') {
          refocus.current = true;
          cancelled.current = true;
          setDraft(null);
        }
      }}
      aria-label={label}
      maxLength={maxLength}
      className={cn(
        className,
        'min-w-0 flex-1 border-b border-primary/50 bg-transparent caret-primary focus:outline-none'
      )}
    />
  );
}

/**
 * The always-visible "+ add step" and its composer (R31, G9). Each step lands
 * as a task at the end of this milestone; the composer stays open, cleared
 * and focused, for the next one. Enter waits on a create in flight, and a
 * failed one keeps its text to try again.
 */
function AddStep({
  title,
  open,
  onOpenChange,
  onAdd,
}: {
  title: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (text: string) => Promise<boolean>;
}) {
  const [text, setText] = useState('');
  const [adding, setAdding] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Opening, from here or the invitation, puts the cursor in it.
  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  const close = () => {
    onOpenChange(false);
    setText('');
  };

  const submit = async () => {
    const step = text.trim();
    if (!step || adding) return;
    setAdding(true);
    const added = await onAdd(step);
    setAdding(false);
    if (added) setText('');
    inputRef.current?.focus();
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => onOpenChange(true)}
        className="flex items-center gap-3 py-1.5 text-base text-foreground/45 outline-none transition-colors hover:text-primary focus-visible:text-primary"
      >
        <Plus aria-hidden strokeWidth={2} className="h-4 w-4" />
        add step
      </button>
    );
  }

  return (
    <div className="flex items-center gap-3 border-b border-foreground/[0.12] py-1.5 focus-within:border-primary/50">
      <Plus aria-hidden strokeWidth={1.75} className="h-4 w-4 shrink-0 text-primary/70" />
      <input
        ref={inputRef}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') submit();
          if (e.key === 'Escape') close();
        }}
        onBlur={() => {
          if (!text.trim() && !adding) close();
        }}
        readOnly={adding}
        placeholder="Add a step…"
        aria-label={`Add a step to ${title}`}
        className="min-w-0 flex-1 bg-transparent text-base leading-6 text-foreground caret-primary placeholder:italic placeholder:text-foreground/35 focus:outline-none read-only:opacity-60"
      />
    </div>
  );
}
