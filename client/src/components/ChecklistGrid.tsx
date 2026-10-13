'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { ChevronRight, FileText, Maximize2, Plus } from 'lucide-react';
import {
  SortableContext,
  rectSortingStrategy,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { Checkbox } from '@/components/ui/checkbox';
import ItemActions from '@/components/ItemActions';
import SortableItem from '@/components/SortableItem';
import ItemDateChip from '@/components/ItemDateChip';
import CompletionNudge, { useCompletionNudges } from '@/components/CompletionNudge';
import ProgressMeter from '@/components/ProgressMeter';
import { cn } from '@/lib/utils';
import { isComplete, readMilestones, type Milestone } from '@/lib/milestones';
import type { ChecklistGroup } from '@/lib/nesting';
import type { ChecklistItem } from '@/services/api';

/**
 * The same groups the list renders, drawn as cards: one card per top-level
 * item, its children inside it, and a progress meter over the task children.
 *
 * The milestone reading of a plan, behind the header's view toggle. It shares
 * the list's collapse state, its type filter and its paging — only the drawing
 * differs. Cards and steps carry the list row's actions panel — rename,
 * dates, the note/task flip, archive, remove, move — less indent, which has
 * nowhere to go between cards; a step can be outdented out of its card.
 */

export interface ChecklistGridProps {
  /** This page's groups, already filtered — same input the list rows get. */
  groups: ChecklistGroup[];
  collapsedIds: ReadonlySet<string>;
  onToggleCollapsed: (id: string) => void;
  onToggleDone: (id: string) => void;
  onRename: (id: string, text: string) => unknown;
  onDelete: (id: string) => unknown;
  onAddChild: (parentId: string, text: string) => Promise<unknown>;
  onToggleType: (id: string) => unknown;
  onArchive: (id: string) => unknown;
  onSetDates: (id: string, startDate: string | null, dueDate: string | null) => unknown;
  /** A step can be pulled out of its block; a block has nothing to leave. */
  onOutdent: (id: string) => unknown;
  /**
   * The move actions one item's panel should carry — blocks among blocks,
   * steps within their own block. Handed over ready-made rather than worked
   * out here: the ends of a set are decided over every sibling, including the
   * ones this page and this filter are not showing (R6.2, R7.2), and the grid
   * only ever sees its own slice.
   */
  moveProps: (id: string) => {
    onMoveUp?: () => void;
    onMoveDown?: () => void;
  };
  /**
   * Whether this item gets a grip. Decided by the caller for the same reason
   * `moveProps` is: an item alone in its set has nothing to be dragged past,
   * and only the whole set can say so. The drag itself belongs to a
   * `DndContext` above this component — cards and steps are drop targets for
   * each other only within their own sets (R4.1, R4.2, R5.1).
   */
  canDrag: (id: string) => boolean;
  /**
   * The milestone the new-milestone composer is filling, if any. Its card is
   * ringed and starred so the target of the composer's input can be found
   * wherever it sits on the page (FS-FSFEC R33).
   */
  tracedId?: string | null;
  /**
   * Items just created, which fade in where they landed — the same arrival
   * the list gives a new row, so a card made from the composer is found.
   */
  arriving?: Readonly<Record<string, boolean>>;
  /**
   * Where a milestone's own page lives (FS-043WP R5, R6). Given it, every
   * milestone card carries a link there and an Open entry in its menu; note
   * cards and steps never do, being no milestone. The grid knows which cards
   * are milestones; the caller knows the plan and its URLs.
   */
  milestoneHref?: (id: string) => string;
  /**
   * The menu's Open on a milestone card (FS-043WP R6). Navigation stays with
   * the caller, as the URLs do; without it the menu offers no Open.
   */
  onOpenMilestone?: (id: string) => void;
}

const isNote = (item: ChecklistItem) => (item.type ?? 'task') === 'note';

/** done / total over the task children only; notes never count as progress. */
function taskProgress(children: ChecklistItem[]) {
  const tasks = children.filter((c) => !isNote(c));
  return { done: tasks.filter((c) => c.done).length, total: tasks.length };
}

export default function ChecklistGrid({
  groups,
  collapsedIds,
  onToggleCollapsed,
  onToggleDone,
  onRename,
  onDelete,
  onAddChild,
  onToggleType,
  onArchive,
  onSetDates,
  onOutdent,
  moveProps,
  canDrag,
  tracedId = null,
  arriving = {},
  milestoneHref,
  onOpenMilestone,
}: ChecklistGridProps) {
  // Rename and add live here rather than in the list's shared edit state: the
  // two views are never on screen together, and a card's input has nothing to
  // say to the list's.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editText, setEditText] = useState('');
  const [addingIn, setAddingIn] = useState<string | null>(null);
  const [addText, setAddText] = useState('');
  const [isAdding, setIsAdding] = useState(false);
  const addInputRef = useRef<HTMLInputElement>(null);

  // Each card's milestone, read by the same derivation the strip uses so the
  // two can't disagree about a card's progress or whether it is complete.
  const reading = useMemo(
    () => readMilestones(groups.flatMap((g) => [g.item, ...g.children])).milestones,
    [groups]
  );
  const milestones = useMemo(
    () => new Map<string, Milestone>(reading.map((m) => [m.id, m])),
    [reading]
  );
  // The nudge and its memory of being waved off, shared with the milestone
  // page so the offer behaves the same wherever it is made (FS-043WP R21).
  const nudges = useCompletionNudges(reading);

  // Opening the composer, or coming back after a save, puts the cursor in it
  // so steps can be entered back to back the way the add bar allows.
  useEffect(() => {
    if (addingIn && !isAdding) addInputRef.current?.focus();
  }, [addingIn, isAdding]);

  const commitEdit = (id: string) => {
    const text = editText.trim();
    if (text) onRename(id, text);
    setEditingId(null);
    setEditText('');
  };

  const submitChild = async (parentId: string) => {
    const text = addText.trim();
    if (!text || isAdding) return;
    setIsAdding(true);
    try {
      await onAddChild(parentId, text);
      setAddText('');
    } finally {
      setIsAdding(false);
    }
  };

  const renderText = (item: ChecklistItem, className: string) =>
    editingId === item.id ? (
      <input
        autoFocus
        value={editText}
        onChange={(e) => setEditText(e.target.value)}
        onBlur={() => commitEdit(item.id)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commitEdit(item.id);
          if (e.key === 'Escape') setEditingId(null);
        }}
        aria-label={`Rename ${item.description}`}
        className={cn(
          className,
          'min-w-0 flex-1 border-b border-primary/50 bg-transparent caret-primary focus:outline-none'
        )}
      />
    ) : (
      <span
        onClick={() => {
          setEditingId(item.id);
          setEditText(item.description);
        }}
        className={cn(className, 'min-w-0 flex-1 cursor-text')}
      >
        {item.description}
      </span>
    );

  return (
    // items-start so a one-line block stays one line instead of stretching to
    // its neighbour's height — a block with no steps yet should look like it.
    <SortableContext
      items={groups.map((g) => g.item.id)}
      strategy={rectSortingStrategy}
    >
    <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {groups.map(({ item, children }) => {
        const collapsed = collapsedIds.has(item.id);
        const milestone = milestones.get(item.id);
        const { done, total } = milestone?.steps ?? taskProgress(children);
        const noteCount = children.length - total;
        const composing = addingIn === item.id;
        const complete = !!milestone && isComplete(milestone);
        const traced = item.id === tracedId;
        const openHref = milestone && milestoneHref?.(item.id);

        return (
          <SortableItem
            key={item.id}
            id={item.id}
            label={item.description}
            disabled={!canDrag(item.id)}
            // The card is its own hover group; the shared grip waits on a row.
            handleClassName="group-hover/card:opacity-100"
          >
          {({ bind, handle }) => (
          <div
            {...bind}
            data-checklist-card={item.id}
            data-complete={complete || undefined}
            data-traced={traced || undefined}
            data-actions-anchor
            className={cn(
              'group/card rounded-xl bg-foreground/[0.04] px-4 py-4 backdrop-blur-sm',
              'transition-[background-color,box-shadow] duration-300',
              // No border to thicken on hover — the fill lifts instead, and
              // an ember glow sits under it, the same one the Add button
              // wears once there is something to add.
              'hover:bg-foreground/[0.06] hover:shadow-[0_2px_18px_-12px_rgba(247,111,83,0.55)]',
              // Complete: the hover glow, held, with a hairline of ember so
              // it reads as a state rather than a hover.
              complete &&
                'shadow-[0_0_0_1px_rgba(247,111,83,0.28),0_4px_24px_-10px_rgba(247,111,83,0.6)]',
              item.done && 'opacity-60',
              // Traced: a ring of ember, composing with any glow above.
              traced && 'ring-2 ring-primary/60',
              arriving[item.id] && 'animate-fadeIn'
            )}
          >
            {/* Head: what the block is, and how far along it is. */}
            <div className="flex items-start gap-2.5">
              {isNote(item) ? (
                <FileText
                  aria-hidden
                  strokeWidth={1.75}
                  className="mt-[3px] h-[18px] w-[18px] shrink-0 text-foreground/35"
                />
              ) : (
                <span className="mt-[3px] shrink-0">
                  <Checkbox
                    id={`card-${item.id}`}
                    checked={item.done}
                    onCheckedChange={() => onToggleDone(item.id)}
                    aria-label={`Mark ${item.description} done`}
                  />
                </span>
              )}
              {renderText(
                item,
                cn(
                  'text-base leading-6 text-foreground',
                  children.length > 0 && 'font-medium',
                  item.done && 'line-through text-foreground/50',
                  isNote(item) && 'italic font-normal text-foreground/75'
                )
              )}
              {traced && (
                <span
                  aria-hidden
                  data-traced-mark
                  className="shrink-0 text-sm leading-6 text-primary"
                >
                  ✦
                </span>
              )}
              {/* The way into the milestone's page: a real link, so it opens
                  in a new tab and copies. Always there but muted, so a page
                  of cards isn't a page of buttons; the title keeps renaming. */}
              {openHref && (
                <Link
                  href={openHref}
                  aria-label={`Open ${item.description}`}
                  title="Open"
                  className="mt-[1px] grid h-6 w-6 shrink-0 place-items-center rounded-md text-foreground/30 outline-none transition-colors hover:text-primary focus-visible:text-primary"
                >
                  <Maximize2 aria-hidden strokeWidth={1.75} className="h-3.5 w-3.5" />
                </Link>
              )}
              {handle && (
                <span
                  className="mt-[1px] shrink-0"
                  onClick={(e) => e.stopPropagation()}
                >
                  {handle}
                </span>
              )}
              <span
                className="mt-[1px] shrink-0"
                onClick={(e) => e.stopPropagation()}
              >
                <ItemActions
                  item={item}
                  onOpen={
                    openHref && onOpenMilestone
                      ? () => onOpenMilestone(item.id)
                      : undefined
                  }
                  onEdit={() => {
                    setEditingId(item.id);
                    setEditText(item.description);
                  }}
                  onToggleType={() => onToggleType(item.id)}
                  onArchive={() => onArchive(item.id)}
                  onDelete={() => onDelete(item.id)}
                  onSetDates={(start, due) => onSetDates(item.id, start, due)}
                  {...moveProps(item.id)}
                />
              </span>
              {children.length > 0 && (
                <button
                  type="button"
                  aria-expanded={!collapsed}
                  aria-label={`${collapsed ? 'Expand' : 'Collapse'} ${item.description}`}
                  onClick={() => onToggleCollapsed(item.id)}
                  className="-mr-1 mt-[1px] grid h-6 w-6 shrink-0 place-items-center rounded-md text-foreground/35 outline-none transition-colors hover:text-primary focus-visible:text-primary"
                >
                  <ChevronRight
                    aria-hidden
                    className={cn(
                      'h-4 w-4 transition-transform duration-200',
                      !collapsed && 'rotate-90'
                    )}
                  />
                </button>
              )}
            </div>

            {/* The meter is the whole point of the card: progress at a glance
                without opening anything. A block whose children are all notes
                has no progress to show, so it says what it holds instead. */}
            {children.length > 0 && (
              <div className="mt-2.5 flex items-center gap-2.5">
                {total > 0 ? (
                  <ProgressMeter done={done} total={total} />
                ) : (
                  <span className="text-sm text-foreground/45">
                    {noteCount === 1 ? '1 note' : `${noteCount} notes`}
                  </span>
                )}
                <ItemDateChip item={item} />
              </div>
            )}
            {children.length === 0 && (
              <ItemDateChip item={item} className="mt-2" />
            )}

            {/* Every step done: say so and offer to close it. Only an offer,
                through the checkbox's own handler; nothing here sets done on
                its own (R10, R11, R13). */}
            {milestone && nudges.shows(milestone) && (
              <CompletionNudge
                title={item.description}
                onClose={() => onToggleDone(item.id)}
                onDismiss={() => nudges.dismiss(milestone)}
                className="mt-2.5"
              />
            )}

            {/* Steps. Folded away with the same state the list folds with, so
                a block collapsed here is collapsed there. */}
            {children.length > 0 && !collapsed && (
              <SortableContext
                items={children.map((c) => c.id)}
                strategy={verticalListSortingStrategy}
              >
              <ul className="mt-3 space-y-0.5 border-t border-foreground/[0.07] pt-2.5">
                {children.map((child) => (
                  <SortableItem
                    key={child.id}
                    id={child.id}
                    label={child.description}
                    disabled={!canDrag(child.id)}
                  >
                  {({ bind, handle: childHandle }) => (
                  <li
                    {...bind}
                    className="group/row flex items-start gap-2.5 rounded-md py-1"
                  >
                    {isNote(child) ? (
                      <span
                        aria-hidden
                        className="mt-[7px] h-1 w-1 shrink-0 rounded-full bg-foreground/30"
                      />
                    ) : (
                      <span className="mt-[2px] shrink-0">
                        <Checkbox
                          id={`card-step-${child.id}`}
                          checked={child.done}
                          onCheckedChange={() => onToggleDone(child.id)}
                          aria-label={`Mark ${child.description} done`}
                        />
                      </span>
                    )}
                    {renderText(
                      child,
                      cn(
                        'text-sm leading-5',
                        child.done
                          ? 'text-foreground/40 line-through'
                          : 'text-foreground/80',
                        isNote(child) && 'italic text-foreground/60'
                      )
                    )}
                    <ItemDateChip item={child} className="self-center" />
                    {childHandle && (
                      <span
                        className="self-center"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {childHandle}
                      </span>
                    )}
                    <span
                      className="self-center"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <ItemActions
                        item={child}
                        onEdit={() => {
                          setEditingId(child.id);
                          setEditText(child.description);
                        }}
                        onToggleType={() => onToggleType(child.id)}
                        onArchive={() => onArchive(child.id)}
                        onDelete={() => onDelete(child.id)}
                        onSetDates={(start, due) =>
                          onSetDates(child.id, start, due)
                        }
                        onOutdent={() => onOutdent(child.id)}
                        {...moveProps(child.id)}
                      />
                    </span>
                  </li>
                  )}
                  </SortableItem>
                ))}
              </ul>
              </SortableContext>
            )}

            {/* Nesting by hand: in a block, a step goes in the block. No Tab
                gesture to know about. Quiet until the card is hovered, so a
                page of blocks isn't a page of buttons. */}
            {!isNote(item) && !collapsed && (
              <div className={cn('mt-1', children.length === 0 && 'mt-2')}>
                {composing ? (
                  <div className="ml-[3px] flex items-center gap-[13px] border-b border-foreground/[0.12] py-1 pl-0 focus-within:border-primary/50">
                    <Plus
                      aria-hidden
                      strokeWidth={1.75}
                      className="h-3.5 w-3.5 shrink-0 text-primary/70"
                    />
                    <input
                      ref={addInputRef}
                      value={addText}
                      onChange={(e) => setAddText(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') submitChild(item.id);
                        if (e.key === 'Escape') {
                          setAddingIn(null);
                          setAddText('');
                        }
                      }}
                      onBlur={() => {
                        if (!addText.trim()) setAddingIn(null);
                      }}
                      readOnly={isAdding}
                      placeholder="Add a step…"
                      aria-label={`Add a step to ${item.description}`}
                      className="min-w-0 flex-1 bg-transparent text-sm leading-5 text-foreground caret-primary placeholder:italic placeholder:text-foreground/35 focus:outline-none read-only:opacity-60"
                    />
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => setAddingIn(item.id)}
                    className="ml-[3px] flex items-center gap-[13px] py-1 text-sm text-foreground/0 outline-none transition-colors group-hover/card:text-foreground/40 hover:!text-primary focus-visible:text-primary [@media(hover:none)]:text-foreground/40"
                  >
                    <Plus strokeWidth={2} className="h-3.5 w-3.5" />
                    add step
                  </button>
                )}
              </div>
            )}
          </div>
          )}
          </SortableItem>
        );
      })}
    </div>
    </SortableContext>
  );
}
