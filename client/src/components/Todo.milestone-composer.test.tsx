import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, within, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
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
  startFilling,
  toast,
  topLevel,
  traced,
  viewButton,
} from '@/test/todoHarness';

/**
 * The grid's new-milestone composer (FS-FSFEC R29–R36, R40, R42–R45): the
 * pinned bar rests as "+ New milestone", names a new top-level task, then
 * fills it with steps while the grid traces the card it is filling. List view
 * keeps its add bar untouched.
 */

const { createChecklistItem } = api;
// Queried by name rather than inside the composer: the list view has none.
const resting = () => screen.queryByRole('button', { name: 'New milestone' });
const renderTodo = (items: unknown[] = [item('P0')]) => renderHarness(items);

describe('Todo new-milestone composer', () => {
  beforeEach(() => {
    createChecklistItem.mockImplementation(echoCreate);
  });

  it('should name a milestone, then fill it with a step, tracing its card', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await startFilling(user, 'Launch');
    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    expect(createChecklistItem).toHaveBeenCalledTimes(2);
    const [first, second] = createChecklistItem.mock.calls;
    expect(first.slice(0, 3)).toEqual(['Launch', 'plan-1', 'longterm']);
    const opts = first[3] as { type?: string; parentId?: string } | undefined;
    expect(opts?.parentId).toBeUndefined();
    expect(opts?.type).toBe('task');
    expect(second).toEqual([
      'Write copy',
      'plan-1',
      'longterm',
      { type: 'task', parentId: 'new1' },
    ]);
    expect(traced()).toEqual(['new1']);
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
  });

  it('should rest as "+ New milestone" in grid view, in place of the add bar', async () => {
    await renderTodo();

    expect(resting()).toBeTruthy();
    expect(composerState()).toBe('resting');
    expect(screen.queryByRole('textbox', { name: /^New (task|note)/ })).toBeNull();
  });

  it('should leave the list view its add bar, with no composer', async () => {
    window.localStorage.setItem('checklistView', 'list');
    await renderTodo();

    expect(composer()).toBeNull();
    expect(resting()).toBeNull();
    const input = screen.getByRole('textbox', { name: 'New task' }) as HTMLInputElement;
    expect(input.placeholder).toBe('Add something to do…');
  });

  it('should open naming with focus on click, Enter or Space', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(resting()!);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.placeholder).toBe('Name a new milestone…');

    for (const key of ['{Enter}', ' ']) {
      await user.keyboard('{Escape}');
      expect(composerState()).toBe('resting');
      resting()!.focus();
      await user.keyboard(key);
      expect(composerState()).toBe('naming');
      expect(document.activeElement).toBe(namingInput());
    }
  });

  it('should take no focus on load', async () => {
    await renderTodo();
    expect(composer().contains(document.activeElement)).toBe(false);
  });

  it('should follow a new milestone to the last page, traced and named in the input', async () => {
    const user = userEvent.setup();
    await renderTodo(topLevel(24));
    expect(counter().textContent).toBe('1 of 2');

    await startFilling(user, 'Launch');

    // 25 milestones is three pages of twelve, and the new one opens the third.
    expect(counter().textContent).toBe('3 of 3');
    expect(card('new1')).not.toBeNull();
    expect(traced()).toEqual(['new1']);
    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.placeholder).toBe('Add a step…');
    expect(composer().textContent).toContain('steps for Launch');
    expect(composer().textContent).not.toContain('↳');
    expect(document.activeElement).toBe(fillingInput('Launch'));
  });

  it('should do nothing on an empty Enter in naming', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);

    await user.type(namingInput()!, '   {Enter}');

    expect(createChecklistItem).not.toHaveBeenCalled();
    expect(composerState()).toBe('naming');
  });

  it('should stay naming with the text kept when the create fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    createChecklistItem.mockRejectedValueOnce(new Error('nope'));
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);

    await user.type(namingInput()!, 'Launch{Enter}');

    expect(composerState()).toBe('naming');
    expect(namingInput()!.value).toBe('Launch');
    expect(traced()).toEqual([]);
    expect(
      screen.getByText('Failed to add milestone. Please try again.')
    ).toBeTruthy();
  });

  it('should add a step, clear the input and keep focus, staying in filling', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');
    await user.type(fillingInput('Launch')!, 'Ship it{Enter}');

    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.value).toBe('');
    expect(document.activeElement).toBe(fillingInput('Launch'));
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
    expect(within(card('new1')!).getByText('Ship it')).toBeTruthy();
  });

  it('should keep the step text when the step create fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    createChecklistItem.mockRejectedValueOnce(new Error('nope'));

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    expect(composerState()).toBe('filling');
    expect(fillingInput('Launch')!.value).toBe('Write copy');
    expect(traced()).toEqual(['new1']);
  });

  it('should open a collapsed traced milestone when a step arrives', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');

    await user.click(
      within(card('new1')!).getByRole('button', { name: 'Collapse Launch' })
    );
    expect(within(card('new1')!).queryByText('Write copy')).toBeNull();

    await user.type(fillingInput('Launch')!, 'Ship it{Enter}');

    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
    expect(within(card('new1')!).getByText('Ship it')).toBeTruthy();
  });

  it('should go from filling to naming on Tab or Shift+Tab, text kept and trace cleared', async () => {
    const user = userEvent.setup();
    await renderTodo();

    for (const key of ['{Tab}', '{Shift>}{Tab}{/Shift}']) {
      await user.keyboard('{Escape}');
      await startFilling(user, `Launch ${key.length}`);
      await user.type(fillingInput(`Launch ${key.length}`)!, 'half typed');

      await user.keyboard(key);

      expect(composerState()).toBe('naming');
      expect(namingInput()!.value).toBe('half typed');
      expect(document.activeElement).toBe(namingInput());
      expect(traced()).toEqual([]);
    }
  });

  it('should swallow Tab and Shift+Tab in naming, focus staying put', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch');

    await user.keyboard('{Tab}');
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());

    await user.keyboard('{Shift>}{Tab}{/Shift}');
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());
    expect(namingInput()!.value).toBe('Launch');
  });

  it('should rest on Esc from naming or filling, text gone and nothing traced', async () => {
    const user = userEvent.setup();
    await renderTodo();

    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch');
    await user.keyboard('{Escape}');
    expect(composerState()).toBe('resting');
    expect(document.activeElement).toBe(resting());

    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'half');
    await user.keyboard('{Escape}');
    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);

    await user.click(resting()!);
    expect(namingInput()!.value).toBe('');
  });

  it('should keep filling and the trace when a card is clicked or a step ticked', async () => {
    const user = userEvent.setup();
    await renderTodo([
      item('P0'),
      item('S0', { parentId: 'P0' }),
    ]);
    await startFilling(user);

    await user.click(card('P0')!);
    await user.click(
      within(card('P0')!).getByRole('checkbox', { name: 'Mark item S0 done' })
    );

    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
    expect(composer().contains(document.activeElement)).toBe(false);

    await user.type(fillingInput('Launch')!, 'Write copy{Enter}');
    expect(within(card('new1')!).getByText('Write copy')).toBeTruthy();
  });

  it('should start resting again after a reload mid-fill', async () => {
    const user = userEvent.setup();
    const { unmount } = await renderTodo();
    await startFilling(user);
    unmount();

    await renderTodo();

    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);
  });

  it('should ignore Enter while a create is in flight, focus kept', async () => {
    let resolve!: (v: unknown) => void;
    createChecklistItem.mockImplementationOnce(
      () => new Promise((r) => (resolve = r))
    );
    const user = userEvent.setup();
    await renderTodo();
    await user.click(resting()!);
    await user.type(namingInput()!, 'Launch{Enter}');

    expect(namingInput()!.readOnly).toBe(true);
    await user.keyboard('{Enter}');
    await user.keyboard('{Tab}');
    await user.keyboard('{Escape}');

    expect(createChecklistItem).toHaveBeenCalledTimes(1);
    expect(composerState()).toBe('naming');
    expect(document.activeElement).toBe(namingInput());

    await act(async () => resolve(item('new1', { description: 'Launch' })));
    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
  });

  it('should never offer the "Tab to nest" tip', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.keyboard('{Tab}');
    await user.keyboard('{Tab}');

    expect(toast).not.toHaveBeenCalledWith(
      expect.objectContaining({ title: 'Tip: Tab to nest' })
    );
  });

  it("should let a card's own add step add to that card, leaving the composer alone", async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);
    await user.type(fillingInput('Launch')!, 'half');

    await user.click(within(card('P0')!).getByRole('button', { name: 'add step' }));
    await user.type(
      within(card('P0')!).getByRole('textbox', { name: 'Add a step to item P0' }),
      'Old step{Enter}'
    );

    expect(createChecklistItem).toHaveBeenLastCalledWith(
      'Old step',
      'plan-1',
      'longterm',
      { type: 'task', parentId: 'P0' }
    );
    expect(within(card('P0')!).getByText('Old step')).toBeTruthy();
    expect(composerState()).toBe('filling');
    expect(traced()).toEqual(['new1']);
    expect(fillingInput('Launch')!.value).toBe('half');
  });

  it('should clear the trace when the view switches to the list', async () => {
    const user = userEvent.setup();
    await renderTodo();
    await startFilling(user);

    await user.click(viewButton('Checklist'));
    await user.click(viewButton('Milestones'));

    expect(composerState()).toBe('resting');
    expect(traced()).toEqual([]);
  });
});
