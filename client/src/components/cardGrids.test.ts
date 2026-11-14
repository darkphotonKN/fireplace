import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { createRequire } from 'module';
import path from 'path';

/**
 * FS-49CQ2 §Card grids follow the space they get (R12–R14).
 *
 * jsdom does no layout, so the column count can't be measured here. What can be
 * held still is the contract the layout rests on: one shared intrinsic-column
 * definition (R13), used by the six card grids instead of viewport counts (R12),
 * and the non-card grids left on their own columns (R14). The widths themselves
 * are checked in a browser.
 */

const require = createRequire(import.meta.url);
const clientRoot = path.resolve(__dirname, '../..');
const read = (rel: string) => readFileSync(path.join(clientRoot, rel), 'utf8');

const CARD_GRIDS = [
  'src/components/ChecklistGrid.tsx',
  'src/components/DayPlanSection.tsx',
  'src/app/plan/[planId]/layout.tsx',
  'src/app/myplans/page.tsx',
  'src/components/home/Dashboard.tsx',
  'src/components/home/FocusPrompt.tsx',
];

/** The calendar, tab-list and auth grids keep their own columns (R14). */
const NON_CARD_GRIDS = [
  'src/components/calendar/Calendar.tsx',
  'src/components/notes/NotesContainer.tsx',
  'src/app/auth/page.tsx',
];

const VIEWPORT_COLUMNS = /\b(?:sm|md|lg|xl|2xl):grid-cols-/;

describe('card grids (FS-49CQ2 R12–R14)', () => {
  it('should define one intrinsic-column track list with an 18rem minimum (R13)', () => {
    const config = require(path.join(clientRoot, 'tailwind.config.js'));
    expect(config.theme.extend.gridTemplateColumns.cards).toBe(
      'repeat(auto-fill, minmax(min(100%, 18rem), 1fr))'
    );
  });

  it.each(CARD_GRIDS)('should lay out %s with the shared card columns, not viewport counts (R12)', (file) => {
    const source = read(file);
    expect(source).toContain('grid-cols-cards');
    expect(source).not.toMatch(VIEWPORT_COLUMNS);
  });

  it('should keep the 18rem minimum out of the grids themselves (R13)', () => {
    for (const file of CARD_GRIDS) {
      expect(read(file)).not.toMatch(/18rem|minmax\(/);
    }
  });

  it.each(NON_CARD_GRIDS)('should leave %s off the shared card columns (R14)', (file) => {
    expect(read(file)).not.toContain('grid-cols-cards');
  });
});
