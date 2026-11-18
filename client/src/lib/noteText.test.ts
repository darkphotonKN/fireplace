import { describe, it, expect } from 'vitest';
import {
  cardText,
  charCount,
  clipChars,
  clampClass,
  firstLine,
  itemName,
  NOTE_CARD_LINES,
  NOTE_ROW_LINES,
  NOTE_SUMMARY_THRESHOLD,
} from './noteText';

describe('cardText', () => {
  it('should be the summary when the note has one (FS-PGG2N R4)', () => {
    expect(cardText({ description: 'the long body', summary: 'the gist' })).toBe(
      'the gist'
    );
  });

  it.each([
    ['no summary field', undefined],
    ['a null summary', null],
    ['an empty summary', ''],
  ])('should fall back to the body with %s', (_, summary) => {
    expect(cardText({ description: 'the long body', summary })).toBe('the long body');
  });
});

describe('firstLine', () => {
  it('should take the text up to the first line break', () => {
    expect(firstLine('Groceries\nmilk\neggs')).toBe('Groceries');
  });

  it('should skip leading blank lines and trim the line', () => {
    expect(firstLine('\n\n  Groceries  \nmilk')).toBe('Groceries');
  });

  it('should leave a short single line as it is', () => {
    expect(firstLine('Book flights')).toBe('Book flights');
  });

  it('should cut a line longer than the maximum, marking the cut', () => {
    const cut = firstLine('a'.repeat(100), 60);
    expect(cut).toHaveLength(60);
    expect(cut.endsWith('…')).toBe(true);
  });

  it('should not cut a line exactly at the maximum', () => {
    expect(firstLine('a'.repeat(60), 60)).toBe('a'.repeat(60));
  });

  it('should be empty for empty text', () => {
    expect(firstLine('')).toBe('');
  });
});

describe('tunables', () => {
  it('should name the clamps and the summary threshold (R10, R11, R32)', () => {
    expect(NOTE_CARD_LINES).toBe(4);
    expect(NOTE_ROW_LINES).toBe(2);
    expect(NOTE_SUMMARY_THRESHOLD).toBe(280);
  });
});

describe('clampClass', () => {
  it('should name the Tailwind clamp for a line count', () => {
    expect(clampClass(NOTE_CARD_LINES)).toBe('line-clamp-4');
    expect(clampClass(NOTE_ROW_LINES)).toBe('line-clamp-2');
  });

  it('should hold a count outside the spelled-out range at its nearest end', () => {
    expect(clampClass(0)).toBe('line-clamp-1');
    expect(clampClass(9)).toBe('line-clamp-6');
  });
});

describe('itemName', () => {
  it("should name a note by its card text's first line", () => {
    expect(itemName({ type: 'note', description: 'Body\nmore', summary: null })).toBe('Body');
    expect(itemName({ type: 'note', description: 'Body\nmore', summary: 'Gist' })).toBe('Gist');
  });

  it('should name a task by its description, as before', () => {
    expect(itemName({ type: 'task', description: 'Book flights' })).toBe('Book flights');
    expect(itemName({ description: 'Untyped' })).toBe('Untyped');
  });
});

describe('charCount and clipChars (R2, R29)', () => {
  it('should count characters as the server does, an emoji as one', () => {
    expect(charCount('gist')).toBe(4);
    expect(charCount('🔥🔥')).toBe(2);
    expect('🔥🔥'.length).toBe(4);
  });

  it('should cut to a number of characters without splitting one', () => {
    expect(clipChars('🔥🔥🔥', 2)).toBe('🔥🔥');
    expect(clipChars('abc', 5)).toBe('abc');
    expect(clipChars('a🔥b', 2)).toBe('a🔥');
  });
});
