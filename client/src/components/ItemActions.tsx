'use client';

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import {
  Archive,
  CalendarDays,
  CheckSquare,
  FileText,
  IndentDecrease,
  IndentIncrease,
  MoreHorizontal,
  Pencil,
  Trash2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { parseDateOnly, toDateOnly } from '@/lib/itemDates';
import type { ChecklistItem } from '@/services/api';

/**
 * Everything you can do to one item, in one quiet panel that floats beside it.
 *
 * Both views mount this. The list anchors it to the trigger, which sits just
 * after the item's text, so the panel opens where you are looking; a card
 * marks itself `data-actions-anchor` and the panel floats beside the whole
 * block instead of over its steps.
 *
 * It is rendered in a portal because a card's `backdrop-blur` makes its
 * subtree a containing block for fixed positioning — anchored to the
 * viewport, the panel would otherwise be trapped inside the card.
 */

export interface ItemActionsProps {
  item: ChecklistItem;
  onEdit: () => void;
  onToggleType: () => void;
  onArchive: () => void;
  onDelete: () => void;
  /** Both ends together: null clears that end. */
  onSetDates: (startDate: string | null, dueDate: string | null) => void;
  /** List view only — the card has no row above to nest under. */
  onIndent?: () => void;
  onOutdent?: () => void;
  /** Extra classes for the trigger, e.g. when it should stay visible. */
  triggerClassName?: string;
}

const PANEL_GAP = 8;
const PANEL_WIDTH = 208; // w-52

type Placement = { top: number; left: number };

export default function ItemActions({
  item,
  onEdit,
  onToggleType,
  onArchive,
  onDelete,
  onSetDates,
  onIndent,
  onOutdent,
  triggerClassName,
}: ItemActionsProps) {
  const [open, setOpen] = useState(false);
  const [picking, setPicking] = useState(false);
  const [range, setRange] = useState<[Date | null, Date | null]>([null, null]);
  const [place, setPlace] = useState<Placement | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const isNote = (item.type ?? 'task') === 'note';

  const close = useCallback(() => {
    setOpen(false);
    setPicking(false);
    triggerRef.current?.focus();
  }, []);

  // Beside the card where one claims the anchor, otherwise beside the trigger
  // — which in the list is right after the text, so the panel lands where the
  // eye already is rather than out at the card's edge.
  const reposition = useCallback(() => {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const anchor =
      (trigger.closest('[data-actions-anchor]') as HTMLElement | null) ??
      trigger;
    const rect = anchor.getBoundingClientRect();
    // Measured once it is up, so the picker's own width is what gets flipped
    // against the window edge rather than a number kept in step by hand.
    const width = panelRef.current?.offsetWidth || PANEL_WIDTH;

    // Right of the anchor by default; flipped to its left when that would run
    // off the window, and pinned inside the window if neither side fits.
    let left = rect.right + PANEL_GAP;
    if (left + width > window.innerWidth - PANEL_GAP) {
      left = rect.left - width - PANEL_GAP;
    }
    left = Math.max(PANEL_GAP, Math.min(left, window.innerWidth - width - PANEL_GAP));

    const height = panelRef.current?.offsetHeight ?? 0;
    const top = Math.max(
      PANEL_GAP,
      Math.min(rect.top, window.innerHeight - height - PANEL_GAP)
    );
    setPlace({ top, left });
  }, []);

  useLayoutEffect(() => {
    if (open) reposition();
  }, [open, picking, reposition]);

  // The list area scrolls under the panel, so follow it rather than leaving
  // the panel pointing at nothing. Capture, to catch the inner scroller too.
  useEffect(() => {
    if (!open) return;
    const onScroll = () => reposition();
    window.addEventListener('scroll', onScroll, true);
    window.addEventListener('resize', onScroll);
    return () => {
      window.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('resize', onScroll);
    };
  }, [open, reposition]);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      const target = e.target as Node;
      if (panelRef.current?.contains(target)) return;
      if (triggerRef.current?.contains(target)) return;
      setOpen(false);
      setPicking(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open, close]);

  const run = (action: () => void) => () => {
    action();
    close();
  };

  const openPicker = () => {
    setRange([parseDateOnly(item.startDate), parseDateOnly(item.dueDate)]);
    setPicking(true);
  };

  const saveDates = () => {
    const [start, end] = range;
    // A range picked but not finished is a single day, which is what the
    // calendar already makes of one date on its own.
    onSetDates(toDateOnly(start), toDateOnly(end ?? start));
    close();
  };

  const rowClass =
    'flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-sm text-foreground/75 outline-none transition-colors duration-150 hover:bg-foreground/[0.06] hover:text-primary focus-visible:bg-foreground/[0.06] focus-visible:text-primary';

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={`Actions for ${item.description}`}
        onClick={(e) => {
          e.stopPropagation();
          setOpen((o) => !o);
          setPicking(false);
        }}
        onKeyDown={(e) => e.stopPropagation()}
        className={cn(
          'grid h-6 w-6 shrink-0 place-items-center rounded-full text-foreground/35 outline-none transition-[color,background-color,opacity] duration-200',
          'hover:bg-foreground/[0.06] hover:text-primary focus-visible:text-primary',
          // Quiet until wanted: shown on hover or focus within the row or
          // card, always there once open, and always there on touch.
          open
            ? 'opacity-100 text-primary'
            : 'opacity-0 group-hover/row:opacity-100 group-hover/card:opacity-100 group-focus-within/row:opacity-100 group-focus-within/card:opacity-100 [@media(hover:none)]:opacity-60',
          triggerClassName
        )}
      >
        <MoreHorizontal aria-hidden className="h-4 w-4" />
      </button>

      {open &&
        typeof document !== 'undefined' &&
        createPortal(
          <div
            ref={panelRef}
            role="menu"
            data-item-actions
            style={{ top: place?.top ?? 0, left: place?.left ?? 0 }}
            className={cn(
              'fixed z-50 max-h-[calc(100vh-1rem)] overflow-y-auto rounded-xl bg-card/95 p-1.5 backdrop-blur-md',
              'ring-1 ring-foreground/10 shadow-[0_18px_40px_-24px_rgba(0,0,0,0.85)]',
              'animate-in fade-in zoom-in-95 duration-150',
              picking ? 'w-fit' : 'w-52',
              // Invisible until measured, so it never flashes at 0,0.
              place ? 'opacity-100' : 'opacity-0'
            )}
          >
            {picking ? (
              <div className="p-1">
                <p className="px-1 pb-2 text-sm text-foreground/50">
                  Pick a day, or a start and an end.
                </p>
                <DatePicker
                  inline
                  selectsRange
                  startDate={range[0] ?? undefined}
                  endDate={range[1] ?? undefined}
                  onChange={(dates) =>
                    setRange(dates as [Date | null, Date | null])
                  }
                  calendarClassName="fp-datepicker"
                />
                <div className="mt-2 flex items-center justify-between gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      onSetDates(null, null);
                      close();
                    }}
                    className="rounded-lg px-2.5 py-1.5 text-sm text-foreground/50 transition-colors hover:text-primary"
                  >
                    Clear
                  </button>
                  <button
                    type="button"
                    onClick={saveDates}
                    disabled={!range[0]}
                    className="rounded-full bg-primary px-3.5 py-1.5 text-sm font-medium text-primary-foreground shadow-[0_2px_14px_-3px_rgba(247,111,83,0.6)] transition-all duration-300 hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-40"
                  >
                    Save
                  </button>
                </div>
              </div>
            ) : (
              <>
                <button type="button" role="menuitem" className={rowClass} onClick={openPicker}>
                  <CalendarDays aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  {item.startDate || item.dueDate ? 'Change dates' : 'Set dates'}
                </button>
                <button type="button" role="menuitem" className={rowClass} onClick={run(onEdit)}>
                  <Pencil aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  Rename
                </button>
                <button
                  type="button"
                  role="menuitem"
                  className={rowClass}
                  onClick={run(onToggleType)}
                >
                  {isNote ? (
                    <CheckSquare aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  ) : (
                    <FileText aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  )}
                  {isNote ? 'Make a task' : 'Make a note'}
                </button>
                {(onIndent || onOutdent) && (
                  <button
                    type="button"
                    role="menuitem"
                    className={rowClass}
                    onClick={run(() =>
                      item.parentId ? onOutdent?.() : onIndent?.()
                    )}
                  >
                    {item.parentId ? (
                      <IndentDecrease aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                    ) : (
                      <IndentIncrease aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                    )}
                    {item.parentId ? 'Move out' : 'Nest under above'}
                  </button>
                )}
                <div className="my-1 h-px bg-foreground/[0.07]" />
                <button
                  type="button"
                  role="menuitem"
                  className={rowClass}
                  onClick={run(onArchive)}
                >
                  <Archive aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  Archive
                </button>
                <button
                  type="button"
                  role="menuitem"
                  className={rowClass}
                  onClick={run(onDelete)}
                >
                  <Trash2 aria-hidden strokeWidth={1.75} className="h-4 w-4" />
                  Delete
                </button>
              </>
            )}
          </div>,
          document.body
        )}
    </>
  );
}
