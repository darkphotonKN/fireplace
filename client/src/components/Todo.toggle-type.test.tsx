import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { ApiError } from '@/api/client';

/**
 * Make a task on a note that has comments (FS-65B3N R31): the server refuses
 * with code NOTE_HAS_COMMENTS, the row rolls back to a note, and the user is
 * told why. Any other failure keeps the generic message. The branch is on
 * `code`, never on `detail` (ADR-0004).
 */

const fetchChecklist = vi.fn();
const updateChecklistItem = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ planId: 'plan-1' }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }),
}));

vi.mock('@/components/ui/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  fetchArchivedChecklist: vi.fn(async () => []),
  createChecklistItem: vi.fn(),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
  deleteChecklistItem: vi.fn(async () => undefined),
  updateChecklistDates: vi.fn(async () => ({})),
  archiveChecklistItem: vi.fn(async () => ({})),
  reorderChecklistItems: vi.fn(async () => []),
  scheduleChecklistItem: vi.fn(),
  scope: { DAILY: 'daily', LONGTERM: 'longterm' },
  ScopeEnum: { DAILY: 'daily', LONGTERM: 'longterm' },
}));

vi.mock('@/api/insights', () => ({
  getChecklistSuggestion: vi.fn(async () => ''),
}));

vi.mock('@/api/plans', () => ({
  getPlan: vi.fn(async () => ({ id: 'plan-1', dailyReset: false })),
}));

const NOTE = {
  id: 'N0',
  description: 'a note with comments',
  done: false,
  scope: 'longterm',
  type: 'note',
  parentId: null,
  planId: 'plan-1',
};

const R31 = 'This note has comments. Delete them to make it a task.';
const GENERIC = 'Failed to change item type. Please try again.';

async function makeATask() {
  fetchChecklist.mockImplementation(async () => [NOTE]);
  const Todo = (await import('./Todo')).default;
  render(<Todo fixedTaskType="longterm" />);
  await screen.findByText('a note with comments');
  fireEvent.click(
    within(screen.getByRole('group', { name: 'View' })).getByRole('button', {
      name: 'Checklist',
    })
  );
  fireEvent.click(screen.getByRole('button', { name: 'Actions for a note with comments' }));
  fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' }));
}

const isStillANote = () =>
  document.querySelector('[data-note-text="N0"]') !== null;

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
});

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
