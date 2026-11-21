import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent, within, act } from '@testing-library/react';
import { api, card, item, renderTodo } from '@/test/todoHarness';

/**
 * The completion nudge end to end (FS-FSFEC R8–R14): Close goes through the
 * page's own optimistic toggle, so a failure reverts and shows the existing
 * error, and the closed milestone stays where it was. Grid only.
 */

const { updateChecklistItem } = api;

// P1 is complete: both task steps done, itself still open.
const PLAN = [
  item('P0'),
  item('P1'),
  item('S1', { parentId: 'P1', done: true }),
  item('S2', { parentId: 'P1', done: true }),
  item('P2'),
];

const nudge = () => document.querySelector('[data-completion-nudge]');
const closeButton = () =>
  within(card('P1')).getByRole('button', { name: 'Close item P1' });

describe('Todo completion nudge', () => {
  it('should close the milestone on Close and keep it in place, its segment full', async () => {
    await renderTodo(PLAN);
    expect(screen.getByText('0 of 3 milestones')).toBeTruthy();

    await act(async () => fireEvent.click(closeButton()));

    expect(updateChecklistItem).toHaveBeenCalledWith(
      'P1',
      { done: true },
      'plan-1',
      'longterm'
    );
    expect(screen.getByText('1 of 3 milestones')).toBeTruthy();
    expect(nudge()).toBeNull();
    const order = Array.from(
      document.querySelectorAll('[data-checklist-card]')
    ).map((el) => el.getAttribute('data-checklist-card'));
    expect(order).toEqual(['P0', 'P1', 'P2']);
  });

  it('should revert a failed Close and show the existing error', async () => {
    let reject!: (e: Error) => void;
    updateChecklistItem.mockImplementation(
      () => new Promise((_, r) => (reject = r))
    );
    vi.spyOn(console, 'error').mockImplementation(() => {});
    await renderTodo(PLAN);

    fireEvent.click(closeButton());
    // Optimistic: closed before the request lands.
    expect(card('P1').className).toContain('opacity-60');

    await act(async () => reject(new Error('nope')));

    expect(card('P1').className).not.toContain('opacity-60');
    expect(nudge()).not.toBeNull();
    expect(
      screen.getByText('Failed to update task status. Please try again.')
    ).toBeTruthy();
  });

  it('should show no nudge in the checklist reading', async () => {
    await renderTodo(PLAN);
    expect(nudge()).not.toBeNull();

    fireEvent.click(
      within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
        name: 'Checklist',
      })
    );

    expect(nudge()).toBeNull();
    expect(screen.queryByRole('button', { name: 'Close item P1' })).toBeNull();
  });
});
