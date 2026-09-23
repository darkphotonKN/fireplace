'use client';

import type { CSSProperties, ReactNode } from 'react';
import { useSortable } from '@dnd-kit/sortable';
import { GripVertical } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * One draggable thing — a list row, a block, a step — wherever it is drawn.
 *
 * A render prop rather than a wrapper element: the two views draw an item
 * very differently, and neither can afford an extra div in the middle of its
 * layout. The caller spreads `bind` onto the element that should move and
 * puts `handle` wherever that item's other controls live.
 *
 * The grip is built here instead of by each caller so both views offer the
 * same affordance under the same name, and so the keyboard path dnd-kit
 * attaches to it (FS-0009 R3.1, and the touch press-and-hold of §Edge States)
 * cannot be wired up in one view and forgotten in the other.
 */

export interface DragBinding {
  /** Spread onto the element that moves. */
  bind: {
    ref: (node: HTMLElement | null) => void;
    style: CSSProperties;
    'data-sortable-id': string;
  };
  /** True for the item under the pointer, so a caller can dim what it lifts. */
  isDragging: boolean;
  /** The grip. Null when this item has nothing to be dragged past. */
  handle: ReactNode;
}

export interface SortableItemProps {
  id: string;
  /** What the grip is called: "Reorder buy milk". */
  label: string;
  /** No handle, and no drop target: an archived row, or an item alone in its set. */
  disabled?: boolean;
  /** Extra classes for the grip, for the two views' different neighbours. */
  handleClassName?: string;
  children: (drag: DragBinding) => ReactNode;
}

export default function SortableItem({
  id,
  label,
  disabled = false,
  handleClassName,
  children,
}: SortableItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id, disabled });

  const style: CSSProperties = {
    // Written out rather than taken from @dnd-kit/utilities: that package is
    // a transitive dependency here, not a declared one. Scale is deliberately
    // dropped — a row that grows as it is dragged reads as a zoom, not a lift.
    transform: transform
      ? `translate3d(${transform.x}px, ${transform.y}px, 0)`
      : undefined,
    transition,
    // The item being dragged is lifted out of the flow (R3.3): it rides over
    // its neighbours, and the faded original marks the gap it would drop into
    // as the others slide clear of it (R4.3).
    zIndex: isDragging ? 30 : undefined,
    opacity: isDragging ? 0.4 : undefined,
    position: isDragging ? 'relative' : undefined,
  };

  const handle = disabled ? null : (
    <button
      type="button"
      ref={setActivatorNodeRef}
      aria-label={`Reorder ${label}`}
      title="Drag to reorder"
      className={cn(
        'grid h-6 w-6 shrink-0 cursor-grab touch-none place-items-center rounded-md',
        'text-foreground/35 outline-none transition-[color,background-color,opacity] duration-200',
        'hover:bg-foreground/[0.06] hover:text-primary focus-visible:text-primary',
        'active:cursor-grabbing',
        // Quiet until the item is hovered or focused, like its neighbours;
        // always faintly there on touch, where there is no hover to wait for.
        'opacity-0 group-hover/row:opacity-100 group-focus-visible/row:opacity-100 focus-visible:opacity-100',
        '[@media(hover:none)]:opacity-60',
        handleClassName
      )}
      {...attributes}
      {...listeners}
    >
      <GripVertical aria-hidden strokeWidth={1.75} className="h-4 w-4" />
    </button>
  );

  return (
    <>
      {children({
        bind: { ref: setNodeRef, style, 'data-sortable-id': id },
        isDragging,
        handle,
      })}
    </>
  );
}
