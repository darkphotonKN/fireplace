/**
 * What a note shows outside its own page (FS-PGG2N R4, R10, R11).
 *
 * A note's **card text** is its summary when it has one, else its body (the
 * item's `description`). Every place that draws a note away from its page
 * reads it from here, and none of it is stored: the clamp is a render, not a
 * shortened copy.
 *
 * Not `notes.ts`: `@/api/notes` is the gateway's hidden notes system, which
 * this has nothing to do with.
 */

/** Lines of card text a note card shows before it clamps (R10). */
export const NOTE_CARD_LINES = 4;
/** Lines a child note row, a milestone page note step or a list note row shows (R11). */
export const NOTE_ROW_LINES = 2;
/**
 * Body length, in characters, past which a note with no summary is "long":
 * the leave prompt asks for a summary only past it (R32, C4).
 */
export const NOTE_SUMMARY_THRESHOLD = 280;
/**
 * The longest summary the server keeps, in characters (R2, R29): code points,
 * as the server counts runes, so an emoji is one, not two.
 */
export const NOTE_SUMMARY_MAX = 280;

/** How many characters (code points) a text has, counted as the server does. */
export function charCount(text: string): number {
  return Array.from(text).length;
}

/** The text cut to `max` characters (code points), never splitting one. */
export function clipChars(text: string, max: number): string {
  const chars = Array.from(text);
  return chars.length <= max ? text : chars.slice(0, max).join('');
}

/**
 * The search param that opens a note's page in writing mode: the menu's
 * Rename on a note lands there with it (R14), and the page strips it once read.
 */
export const NOTE_EDIT_PARAM = 'edit';

/** A note's page, in the plan's middle card (R16); `edit` opens it writing. */
export function notePageHref(planId: string, noteId: string, edit = false): string {
  return `/plan/${planId}/n/${noteId}${edit ? `?${NOTE_EDIT_PARAM}=1` : ''}`;
}

/** The fields card text is read from; any item shape that has them will do. */
export interface NoteTextSource {
  description: string;
  summary?: string | null;
}

/** The summary if there is one, else the body (R4). */
export function cardText(item: NoteTextSource): string {
  return item.summary || item.description;
}

/**
 * The first non-blank line, trimmed, for one-line places: the calendar, menu
 * names, the delete confirmation, the tab title (R11, R21). Past `max`
 * characters it is cut, the last one given to an ellipsis.
 */
export function firstLine(text: string, max?: number): string {
  const line = text.split('\n').find((l) => l.trim() !== '')?.trim() ?? '';
  if (max === undefined || line.length <= max) return line;
  return `${line.slice(0, max - 1).trimEnd()}…`;
}

/**
 * How a one-line place names an item: a note by its card text's first line, so
 * a menu or a confirmation never reads out a whole body; anything else by its
 * description, as before (R11).
 */
export function itemName(item: NoteTextSource & { type?: string }): string {
  return item.type === 'note' ? firstLine(cardText(item)) : item.description;
}

/**
 * The Tailwind clamp for a line count. Spelled out so the JIT sees every class
 * it may be asked for; a tunable set outside them clamps at the nearest end.
 */
const CLAMPS = [
  'line-clamp-1',
  'line-clamp-2',
  'line-clamp-3',
  'line-clamp-4',
  'line-clamp-5',
  'line-clamp-6',
] as const;
export function clampClass(lines: number): string {
  return CLAMPS[Math.min(Math.max(lines, 1), CLAMPS.length) - 1];
}

/**
 * A note row's text (R11, R48): upright in a small chip of paper, line breaks
 * kept, clamped to the row's lines, a long URL wrapping rather than widening
 * the row. Size and colour are the row's own to add.
 */
export const NOTE_ROW_CHIP = [
  'w-fit max-w-full rounded-md bg-note-paper px-2 py-0.5 text-foreground/88',
  'whitespace-pre-line break-words',
  clampClass(NOTE_ROW_LINES),
].join(' ');

/** The same chip as the way to the note's page (R13): a quiet ember edge on hover. */
export const NOTE_ROW_LINK = [
  NOTE_ROW_CHIP,
  'outline-none transition-shadow hover:shadow-[0_0_0_1px_hsl(var(--primary)/0.35)] focus-visible:ring-2 focus-visible:ring-ring/60',
].join(' ');
