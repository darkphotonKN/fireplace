import { differenceInCalendarDays, format } from 'date-fns';

/**
 * When a comment was written, as a person says it (FS-65B3N R26): "just now"
 * under a minute, "N min ago" under an hour, "N h ago" under a day, then
 * "yesterday" and "N days ago" by calendar day for the rest of the week, and
 * after that the date ("Nov 3"; with the year when it is not this year).
 * Days are the browser's local days.
 */
export function relativeTime(at: Date | string, now: Date = new Date()): string {
  const when = typeof at === 'string' ? new Date(at) : at;
  const elapsed = now.getTime() - when.getTime();
  const minutes = Math.floor(elapsed / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} h ago`;
  const days = differenceInCalendarDays(now, when);
  if (days <= 1) return 'yesterday';
  if (days < 7) return `${days} days ago`;
  return format(when, when.getFullYear() === now.getFullYear() ? 'MMM d' : 'MMM d, yyyy');
}

/** The full date and time, for the tooltip and accessible text (R26). */
export function fullDateTime(at: Date | string): string {
  return format(typeof at === 'string' ? new Date(at) : at, 'MMM d, yyyy, h:mm a');
}
