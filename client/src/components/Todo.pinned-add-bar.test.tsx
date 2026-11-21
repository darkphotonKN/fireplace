import { describe, it, expect } from 'vitest';
import { cleanup, screen, fireEvent, waitFor } from '@testing-library/react';
import { addInput, api, item, renderTodo } from '@/test/todoHarness';

/**
 * FS-0008 R13–R14: the add bar is pinned to the bottom of the list card and
 * stays in view while the list area scrolls, without moving off the end of
 * the list. jsdom computes no geometry, so these assert the structure and the
 * classes that produce the pinning — the visual result is HITL.
 */

const { createChecklistItem } = api;

// Long enough that the card would run off the screen without pinning.
const LONG = Array.from({ length: 24 }, (_, i) => item(`L${i}`));

// The add bar's wrapper is what pins; the form inside keeps its own look.
const pinned = () => addInput().closest('form')!.parentElement!;
// Sticky is inert without a scrolling ancestor: this is the bounded list area
// the rows scroll in and the add bar pins to, in list and grid alike.
const listArea = () => document.querySelector<HTMLElement>('[data-list-area]')!;

// A single page is capped and scrolls inside; a paged list is held at a fixed
// height so paging never moves the controls or the bar under the cursor (R29).
const BOUNDED = /(^|\s)max-h-\[/;
const FIXED = /(^|\s)h-\[/;

const renderLong = (items: unknown[] = LONG, view: 'list' | 'grid' = 'list') =>
  renderTodo(items, { view });

describe('Todo pinned add bar (FS-0008 R13–R14)', () => {
  it('should pin the add bar to the bottom of the list area', async () => {
    await renderLong();
    expect(pinned().className).toMatch(/\bsticky\b/);
    expect(pinned().className).toMatch(/\bbottom-0\b/);
  });

  it('should scroll the list under the bar in a bounded area holding both', async () => {
    await renderLong();
    const area = listArea();
    expect(area.className).toMatch(/\boverflow-y-auto\b/);
    // Bounded either way: a single page caps with max-h, a paged list gets the
    // fixed window so every page is the same height (FS-0008 R5).
    expect(area.className).toMatch(/\b(max-)?h-\[/);
    // The rows must scroll inside the same box the bar sticks to, and the bar
    // must still come last in the DOM so Tab-to-nest keeps its meaning (R13).
    const rows = area.querySelector('ul')!;
    expect(rows).toBeTruthy();
    expect(rows.compareDocumentPosition(pinned()) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('should still add through the pinned bar and hand focus back', async () => {
    createChecklistItem.mockImplementation(async () => item('NEW'));
    await renderLong();
    const input = addInput();
    input.focus();
    fireEvent.change(input, { target: { value: 'item NEW' } });
    fireEvent.submit(input.closest('form')!);
    // 24 items is two pages, so the appended row lands on the last one and
    // the view goes with it (R17) — the arrival is seen, not left off-page.
    expect(await screen.findByText('item NEW')).toBeTruthy();
    await waitFor(() => expect(addInput().readOnly).toBe(false));
    expect(document.activeElement).toBe(addInput());
  });

  it('should leave no pinned strip when the archived view is open', async () => {
    await renderLong();
    fireEvent.click(screen.getByTitle('Settings'));
    fireEvent.click(screen.getByText('View archived tasks'));
    // The back link naming settings is the archived view's own marker.
    await screen.findByText('Back to settings');
    expect(screen.queryByRole('textbox', { name: /^New (task|note)/ })).toBeNull();
    expect(document.querySelector('[data-add-bar-fade]')).toBeNull();
  });

  it('should hold the list area at a fixed height on every page once the list pages', async () => {
    // 24 items is two pages. Without a fixed height the section would
    // collapse from twenty rows to four when page 2 holds the rest, moving
    // the controls and the bar under the cursor (R29).
    await renderLong();
    const area = listArea();
    expect(area.className).toMatch(FIXED);
    expect(area.className).not.toMatch(/\bmax-h-\[/);
    // The rows region grows, so a short page still leaves the bar at the
    // window's bottom rather than floating under the last row.
    expect(area.firstElementChild!.className).toMatch(/\bflex-1\b/);

    // Page 2 is the short last page: four rows, same height.
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(listArea().className).toMatch(FIXED);
  });

  it('should keep a single page bounded and scrolling inside, so one huge family still reaches the bar', async () => {
    await renderLong(LONG.slice(0, 4));
    // Only the cap applies, for the one-huge-family case, and it scrolls
    // inside (R28).
    const area = listArea();
    expect(area.className).toMatch(BOUNDED);
    expect(area.className).toMatch(/\boverflow-y-auto\b/);
  });

  it('should bound one page and fix many pages in the grid too', async () => {
    // Twelve cards is one page, thirteen is two (R26).
    await renderLong(LONG.slice(0, 12), 'grid');
    expect(listArea().className).toMatch(BOUNDED);
    cleanup();

    await renderLong(LONG.slice(0, 13), 'grid');
    expect(listArea().className).toMatch(FIXED);
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(listArea().className).toMatch(FIXED);
  });
});
