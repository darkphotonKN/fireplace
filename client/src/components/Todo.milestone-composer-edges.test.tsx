import { describe, it, expect, beforeEach } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  addInput as listInput,
  api,
  card,
  composer,
  composerState,
  counter,
  echoCreate,
  fillingInput,
  item,
  namingInput,
  renderTodo as renderHarness,
  restingButton,
  startFilling,
  topLevel,
  traced,
  typeFilter as filterTab,
  viewButton,
} from '@/test/todoHarness';

/**
 * The grid composer when the plan moves under it (FS-FSFEC R37–R39, R41):
 * paging away while filling, the traced milestone going away, notes, and a
 * switch to the list view and back.
 */

const { createChecklistItem } = api;
const typeIcon = () =>
  within(composer()).queryByRole('button', { name: /^Switch to adding a/ });
// Through userEvent, so a delete's async round trip settles before the asserts.
const cardAction = async (
  user: ReturnType<typeof userEvent.setup>,
  title: string,
  action: string
) => {
  await user.click(screen.getByRole('button', { name: `Actions for ${title}` }));
  await user.click(screen.getByRole('menuitem', { name: action }));
};
const renderTodo = (items: unknown[] = [item('P0')]) => renderHarness(items);

describe('Todo new-milestone composer, when the plan moves under it', () => {
  beforeEach(() => {
    createChecklistItem.mockImplementation(echoCreate);
  });

  describe('paged away while filling (R37)', () => {
    it('should add the step to the traced milestone and jump back to its page', async () => {
      const user = userEvent.setup();
      // A full page of twelve and one more, so the new milestone lands on
      // page 2 beside t13.
      await renderTodo(topLevel(13));
      await startFilling(user);
      expect(counter().textContent).toBe('2 of 2');

      await user.click(screen.getByRole('button', { name: 'Previous page' }));
      expect(counter().textContent).toBe('1 of 2');
      expect(composerState()).toBe('filling');

      await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'Write copy',
        'plan-1',
        'longterm',
        { type: 'task', parentId: 'new1' }
      );
      expect(counter().textContent).toBe('2 of 2');
      expect(traced()).toEqual(['new1']);
      expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
      expect(document.activeElement).toBe(fillingInput('Launch'));
    });
  });

  describe('traced milestone lost (R38)', () => {
    it.each([
      ['archived', 'Archive'],
      ['deleted', 'Delete'],
      ['converted to a note', 'Make a note'],
    ])('should drop to naming with the text kept when it is %s', async (_, action) => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await cardAction(user, 'Launch', action);

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(traced()).toEqual([]);
      // Lost from under it, not moved by anyone: the cursor isn't pulled down.
      expect(composer().contains(document.activeElement)).toBe(false);
    });

    it('should drop to naming when the Notes filter hides it, and stay there once it is back', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await user.click(filterTab('Notes'));
      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');

      // Forgotten, not held: the milestone coming back doesn't re-trace it.
      await user.click(filterTab('All'));
      expect(composerState()).toBe('naming');
      expect(traced()).toEqual([]);
    });

    it('should name the milestone by its current title after a rename', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);

      await user.click(within(card('new1')!).getByText('Launch'));
      const rename = screen.getByRole('textbox', { name: 'Rename Launch' });
      await user.clear(rename);
      await user.type(rename, 'Go live{Enter}');

      expect(composerState()).toBe('filling');
      expect(fillingInput('Go live')).not.toBeNull();
      expect(composer().textContent).toContain('steps for Go live');
    });
  });

  describe('notes (R39)', () => {
    it('should rest as "+ New note" under the Notes filter and create top-level notes, staying in naming', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await user.click(filterTab('Notes'));

      expect(restingButton().textContent).toContain('New note');
      await user.click(restingButton());
      expect(namingInput()!.placeholder).toBe('Jot down a note…');

      await user.type(namingInput()!, 'An idea{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'An idea',
        'plan-1',
        'longterm',
        { type: 'note' }
      );
      expect(card('new1')).not.toBeNull();
      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('');
      expect(document.activeElement).toBe(namingInput());
      expect(traced()).toEqual([]);
    });

    it('should create notes with the icon on note under All, and milestones again once it is back on task', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await user.click(restingButton());

      await user.click(typeIcon()!);
      expect(namingInput()!.placeholder).toBe('Jot down a note…');
      await user.type(namingInput()!, 'An idea{Enter}');
      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'An idea',
        'plan-1',
        'longterm',
        { type: 'note' }
      );
      expect(composerState()).toBe('naming');

      await user.keyboard('{Escape}');
      expect(restingButton().textContent).toContain('New note');
      await user.click(restingButton());

      await user.click(typeIcon()!);
      expect(namingInput()!.placeholder).toBe('Name a new milestone…');
      await user.type(namingInput()!, 'Launch{Enter}');

      expect(await screen.findByRole('textbox', { name: 'Add a step to Launch' })).toBeTruthy();
      expect(composerState()).toBe('filling');
    });

    it('should let the icon override the Notes filter, a hidden milestone leaving it in naming', async () => {
      const user = userEvent.setup();
      await renderTodo([item('N0', { type: 'note' }), item('P0')]);
      await user.click(filterTab('Notes'));
      await user.click(restingButton());

      await user.click(typeIcon()!);
      // Re-choosing the tab you are on doesn't set the type again (R21).
      await user.click(filterTab('Notes'));
      expect(namingInput()!.placeholder).toBe('Name a new milestone…');

      await user.type(namingInput()!, 'Launch{Enter}');

      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'Launch',
        'plan-1',
        'longterm',
        { type: 'task' }
      );
      expect(composerState()).toBe('naming');
      expect(traced()).toEqual([]);
      expect(namingInput()!.value).toBe('');
    });

    it('should offer the type icon in naming only', async () => {
      const user = userEvent.setup();
      await renderTodo();

      expect(typeIcon()).toBeNull();
      await user.click(restingButton());
      expect(typeIcon()).not.toBeNull();
      await user.type(namingInput()!, 'Launch{Enter}');
      await screen.findByRole('textbox', { name: 'Add a step to Launch' });
      expect(typeIcon()).toBeNull();
    });
  });

  describe('view switch (R41)', () => {
    it('should leave the list bar at top level with the text kept, nothing nested', async () => {
      const user = userEvent.setup();
      window.localStorage.setItem('checklistView', 'list');
      await renderTodo();
      // Nested in the list before ever going to the grid.
      await user.click(listInput());
      await user.keyboard('{Tab}');
      expect(listInput().getAttribute('aria-label')).toContain('under');

      await user.click(viewButton('Milestones'));
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');
      await user.click(viewButton('Checklist'));

      expect(listInput().value).toBe('half typed');
      expect(listInput().getAttribute('aria-label')).toBe('New task');

      await user.type(listInput(), '{Enter}');
      expect(createChecklistItem).toHaveBeenLastCalledWith(
        'half typed',
        'plan-1',
        'longterm',
        { type: 'task', parentId: undefined }
      );
    });

    it('should leave a nest alone when the view already showing is chosen again', async () => {
      const user = userEvent.setup();
      window.localStorage.setItem('checklistView', 'list');
      await renderTodo();
      await user.click(listInput());
      await user.keyboard('{Tab}');
      const nested = listInput().getAttribute('aria-label');
      expect(nested).toContain('under');

      // Already in list view: nothing changes, so nothing is reset.
      await user.click(viewButton('Checklist'));

      expect(listInput().getAttribute('aria-label')).toBe(nested);
    });

    it('should come back to naming when text remains, resting when it does not', async () => {
      const user = userEvent.setup();
      await renderTodo();
      await startFilling(user);
      await user.type(fillingInput('Launch')!, 'half typed');

      await user.click(viewButton('Checklist'));
      await user.click(viewButton('Milestones'));

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(traced()).toEqual([]);

      await user.click(viewButton('Checklist'));
      await user.clear(listInput());
      await user.click(viewButton('Milestones'));

      expect(composerState()).toBe('resting');
    });
  });
});
