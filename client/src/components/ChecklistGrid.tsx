'use client';

import { useEffect, useRef, useState } from 'react';
import { ChevronRight, FileText, Plus, Trash2 } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import type { ChecklistGroup } from '@/lib/nesting';
import type { ChecklistItem } from '@/services/api';

/**
 * The same groups the list renders, drawn as blocks: one card per top-level
 * item, its children inside it, and a progress meter over the task children.
 *
 * A prototype of the "blocks of progress" reading of a plan, sitting behind
 * the header's view toggle. It shares the list's collapse state, its type
 * filter and its paging — only the drawing differs — and deliberately carries
 * a slimmer row than the list: check, rename, remove, add. Scheduling, moving
 * scope, archiving and the note/task flip stay in list view for now.
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
    <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {groups.map(({ item, children }) => {
        const collapsed = collapsedIds.has(item.id);
        const { done, total } = taskProgress(children);
        const noteCount = children.length - total;
        const composing = addingIn === item.id;

        return (
          <div
            key={item.id}
            data-checklist-card={item.id}
            className={cn(
              'group/card rounded-xl border border-foreground/10 bg-card px-4 py-4 transition-[border-color,box-shadow] duration-300',
              'hover:border-foreground/20 hover:shadow-[0_6px_24px_-18px_rgba(0,0,0,0.65)]',
              item.done && 'opacity-60'
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
                  <>
                    <span className="h-1 flex-1 overflow-hidden rounded-full bg-foreground/10">
                      <span
                        data-progress-fill
                        style={{ width: `${(done / total) * 100}%` }}
                        className="block h-full rounded-full bg-primary transition-[width] duration-500 [transition-timing-function:cubic-bezier(0.22,1,0.36,1)]"
                      />
                    </span>
                    <span className="shrink-0 text-sm tabular-nums text-foreground/45">
                      {done}/{total}
                    </span>
                  </>
                ) : (
                  <span className="text-sm text-foreground/45">
                    {noteCount === 1 ? '1 note' : `${noteCount} notes`}
                  </span>
                )}
              </div>
            )}

            {/* Steps. Folded away with the same state the list folds with, so
                a block collapsed here is collapsed there. */}
            {children.length > 0 && !collapsed && (
              <ul className="mt-3 space-y-0.5 border-t border-foreground/10 pt-2.5">
                {children.map((child) => (
                  <li
                    key={child.id}
                    className="group/step flex items-start gap-2.5 rounded-md py-1"
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
                    <button
                      type="button"
                      onClick={() => onDelete(child.id)}
                      aria-label={`Delete ${child.description}`}
                      className="shrink-0 text-foreground/0 outline-none transition-colors group-hover/step:text-foreground/30 hover:!text-primary focus-visible:text-primary"
                    >
                      <Trash2 strokeWidth={1.75} className="h-3.5 w-3.5" />
                    </button>
                  </li>
                ))}
              </ul>
            )}

            {/* Nesting by hand: in a block, a step goes in the block. No Tab
                gesture to know about. Quiet until the card is hovered, so a
                page of blocks isn't a page of buttons. */}
            {!isNote(item) && !collapsed && (
              <div className={cn('mt-1', children.length === 0 && 'mt-2')}>
                {composing ? (
                  <div className="ml-[3px] flex items-center gap-[13px] border-b border-foreground/15 py-1 pl-0 focus-within:border-primary/50">
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
        );
      })}
    </div>
  );
}
