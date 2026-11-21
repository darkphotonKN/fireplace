import { describe, it, expect } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { ApiError } from '@/api/client';
import { api, chooseAction, item, renderTodo, viewButton } from '@/test/todoHarness';

/**
 * Make a task on a note that has comments (FS-65B3N R31): the server refuses
 * with code NOTE_HAS_COMMENTS, the row rolls back to a note, and the user is
 * told why. Any other failure keeps the generic message. The branch is on
 * `code`, never on `detail` (ADR-0004).
 */

const { updateChecklistItem } = api;

const NOTE = item('N0', { description: 'a note with comments', type: 'note' });

const R31 = 'This note has comments. Delete them to make it a task.';
const GENERIC = 'Failed to change item type. Please try again.';

async function makeATask() {
  await renderTodo([NOTE], { props: { enableTypeFilter: false } });
  fireEvent.click(viewButton('Checklist'));
  chooseAction('a note with comments', 'Make a task');
}

const isStillANote = () =>
  document.querySelector('[data-note-text="N0"]') !== null;

describe('Todo — Make a task on a note with comments (FS-65B3N R31)', () => {
  it('should roll back and say why when the code is NOTE_HAS_COMMENTS', async () => {
    updateChecklistItem.mockRejectedValue(
      new ApiError(
        { code: 'NOTE_HAS_COMMENTS', status: 400, detail: 'anything at all' } as never,
        400
      )
    );
    await makeATask();
    expect(await screen.findByText(R31)).toBeInTheDocument();
    expect(screen.queryByText(GENERIC)).toBeNull();
    expect(isStillANote()).toBe(true);
  });

  it('should branch on code, not detail: another code keeps the generic message', async () => {
    updateChecklistItem.mockRejectedValue(
      new ApiError(
        { code: 'VALIDATION_FAILED', status: 400, detail: 'NOTE_HAS_COMMENTS' } as never,
        400
      )
    );
    await makeATask();
    expect(await screen.findByText(GENERIC)).toBeInTheDocument();
    expect(screen.queryByText(R31)).toBeNull();
    expect(isStillANote()).toBe(true);
  });
});
