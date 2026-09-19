'use client';

import { CalendarDays } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatDateRange, isRangePast } from '@/lib/itemDates';
import type { ChecklistItem } from '@/services/api';

/**
 * An item's dates, shown only once it has some. `scheduledTime` is the
 * deprecated mirror of start_date, so it stands in for a start the item has
 * not been re-saved with yet; the range always wins where both exist.
 */
export default function ItemDateChip({
  item,
  className,
}: {
  item: ChecklistItem;
  className?: string;
}) {
  const start = item.startDate ?? item.scheduledTime?.slice(0, 10);
  const label = formatDateRange(start, item.dueDate);
  if (!label) return null;

  const past = isRangePast(start, item.dueDate);
  return (
    <span
      data-item-dates
      className={cn(
        'inline-flex shrink-0 items-center gap-1 text-sm tabular-nums transition-colors',
        past ? 'text-primary/70' : 'text-foreground/40',
        className
      )}
    >
      <CalendarDays aria-hidden strokeWidth={1.75} className="h-3.5 w-3.5" />
      {label}
    </span>
  );
}
