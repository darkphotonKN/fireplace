import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ItemActions from './ItemActions';
import type { ChecklistItem } from '@/services/api';

/**
 * One panel holding everything you can do to an item, floating beside it.
 * What is pinned here is the behaviour the old icon strip lost: a date RANGE
 * (both ends, inclusive), clearing it, and a menu that closes like a menu.
 */

const item = (extra: Partial<ChecklistItem> = {}) =>
  ({
    id: 'i1',
    description: 'Learn Go',
    done: false,
    scope: 'longterm',
    type: 'task',
    parentId: null,
    ...extra,
  }) as ChecklistItem;

const handlers = () => ({
  onEdit: vi.fn(),
  onToggleType: vi.fn(),
  onArchive: vi.fn(),
  onDelete: vi.fn(),
  onSetDates: vi.fn(),
});

function open(props: Partial<React.ComponentProps<typeof ItemActions>> = {}) {
  const h = handlers();
  render(<ItemActions item={item()} {...h} {...props} />);
  fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
  return h;
}

const day = (label: string) =>
  document.querySelector(`[aria-label^="Choose ${label}"]`) as HTMLElement;

describe('ItemActions', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date(2026, 2, 1));
  });
  afterEach(() => vi.useRealTimers());

  it('should not be left on screen by the focus it is handed back on close', () => {
    // Closing hands focus back to the trigger, which is right for a keyboard
    // and wrong to show for a mouse — so the reveal keys off focus-VISIBLE,
    // never focus-within. jsdom computes no styles, so this pins the rule
    // rather than the pixels; the behaviour itself was checked in a browser.
    const h = handlers();
    render(<ItemActions item={item()} {...h} />);
    const trigger = screen.getByRole('button', { name: 'Actions for Learn Go' });
    expect(trigger.className).toMatch(/focus-visible:opacity-100/);
    expect(trigger.className).not.toMatch(/focus-within/);
  });

  it('should keep the panel shut until it is asked for', () => {
    const h = handlers();
    render(<ItemActions item={item()} {...h} />);
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('should close on Escape and on a click outside', () => {
    open();
    expect(screen.getByRole('menu')).toBeTruthy();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByRole('menu')).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('should save a single day as both ends of the range', () => {
    const { onSetDates } = open();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Set dates' }));
    fireEvent.click(day('Monday, March 9th, 2026'));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    // One day is a range of one, which is what the plan calendar draws as a
    // chip rather than a bar.
    expect(onSetDates).toHaveBeenCalledWith('2026-03-09', '2026-03-09');
  });

  it('should save both ends when a range is picked', () => {
    const { onSetDates } = open();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Set dates' }));
    fireEvent.click(day('Monday, March 9th, 2026'));
    fireEvent.click(day('Monday, March 16th, 2026'));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSetDates).toHaveBeenCalledWith('2026-03-09', '2026-03-16');
  });

  it('should clear both ends, so an item can lose its dates', () => {
    const { onSetDates } = open({
      item: item({ startDate: '2026-03-09', dueDate: '2026-03-16' }),
    });
    // The wording says there is already something to change.
    fireEvent.click(screen.getByRole('menuitem', { name: 'Change dates' }));
    fireEvent.click(screen.getByRole('button', { name: 'Clear' }));

    expect(onSetDates).toHaveBeenCalledWith(null, null);
  });

  it('should open the picker on the dates the item already has', () => {
    open({ item: item({ startDate: '2026-03-09', dueDate: '2026-03-16' }) });
    fireEvent.click(screen.getByRole('menuitem', { name: 'Change dates' }));
    expect(day('Monday, March 9th, 2026').className).toMatch(/range-start/);
    expect(day('Monday, March 16th, 2026').className).toMatch(/range-end/);
  });

  it('should not save a picker that was opened and left empty', () => {
    const { onSetDates } = open();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Set dates' }));
    const save = screen.getByRole('button', { name: 'Save' }) as HTMLButtonElement;
    expect(save.disabled).toBe(true);
    // And it stays a dead end: nothing is written by pressing it anyway.
    fireEvent.click(save);
    expect(onSetDates).not.toHaveBeenCalled();
  });

  it.each([
    ['Rename', 'onEdit'],
    ['Make a note', 'onToggleType'],
    ['Archive', 'onArchive'],
    ['Delete', 'onDelete'],
  ] as const)('should run %s and close', (label, key) => {
    const h = open();
    fireEvent.click(screen.getByRole('menuitem', { name: label }));
    expect(h[key]).toHaveBeenCalled();
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('should offer a note the way back to being a task', () => {
    const h = open({ item: item({ type: 'note' }) });
    fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' }));
    expect(h.onToggleType).toHaveBeenCalled();
  });

  it('should offer nesting only where there is somewhere to nest', () => {
    open();
    expect(screen.queryByRole('menuitem', { name: /Nest|Move out/ })).toBeNull();
  });

  it('should nest a top-level item and free a nested one', () => {
    const onIndent = vi.fn();
    const onOutdent = vi.fn();
    const h = handlers();
    const { rerender } = render(
      <ItemActions item={item()} {...h} onIndent={onIndent} onOutdent={onOutdent} />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Nest under above' }));
    expect(onIndent).toHaveBeenCalled();

    rerender(
      <ItemActions
        item={item({ parentId: 'p1' })}
        {...h}
        onIndent={onIndent}
        onOutdent={onOutdent}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Learn Go' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Move out' }));
    expect(onOutdent).toHaveBeenCalled();
  });
});
