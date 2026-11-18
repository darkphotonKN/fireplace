import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { ApiError } from '@/api/client';
import type { NoteComment } from '@/api/noteComments';

/**
 * A note's comments without the section that draws them (FS-65B3N R28, R30,
 * R33): the load, refreshing in place, and what each code means.
 */

const listNoteComments = vi.fn();
const createNoteComment = vi.fn();
const updateNoteComment = vi.fn();
const deleteNoteComment = vi.fn();
vi.mock('@/api/noteComments', () => ({
  listNoteComments: (...a: unknown[]) => listNoteComments(...a),
  createNoteComment: (...a: unknown[]) => createNoteComment(...a),
  updateNoteComment: (...a: unknown[]) => updateNoteComment(...a),
  deleteNoteComment: (...a: unknown[]) => deleteNoteComment(...a),
}));

import { useNoteComments } from './useNoteComments';

const comment = (id: string, extra: Partial<NoteComment> = {}): NoteComment => ({
  id,
  authorId: `u-${id}`,
  authorName: 'ada',
  body: `comment ${id}`,
  createdAt: new Date().toISOString(),
  editedAt: null,
  canEdit: true,
  canDelete: true,
  ...extra,
});
const refused = (code: string, status: number) =>
  new ApiError({ code, status } as never, status);

const onGone = vi.fn();
const onNotANote = vi.fn();
const setup = async (noteId = 'n1') => {
  const hook = renderHook(({ noteId }) => useNoteComments('plan-1', noteId, { onGone, onNotANote }), {
    initialProps: { noteId },
  });
  await act(async () => {});
  return hook;
};
const ids = (load: ReturnType<typeof useNoteComments>['load']) =>
  load.state === 'ready' ? load.comments.map((c) => c.id) : load.state;

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(console, 'error').mockImplementation(() => {});
  listNoteComments.mockReset().mockResolvedValue([comment('c1'), comment('c2')]);
});

describe('useNoteComments loading (R33)', () => {
  it('should load, then hold the comments oldest first', async () => {
    const { result } = await setup();
    expect(listNoteComments).toHaveBeenCalledWith('plan-1', 'n1');
    expect(ids(result.current.load)).toEqual(['c1', 'c2']);
  });

  it('should fail a first load, and load again on retry', async () => {
    listNoteComments.mockRejectedValueOnce(refused('SERVICE_UNAVAILABLE', 503));
    const { result } = await setup();
    expect(result.current.load.state).toBe('failed');
    await act(async () => result.current.retry());
    expect(ids(result.current.load)).toEqual(['c1', 'c2']);
  });

  it('should start from loading again for another note', async () => {
    const { result, rerender } = await setup('n1');
    listNoteComments.mockReturnValue(new Promise(() => {}));
    rerender({ noteId: 'n2' });
    expect(result.current.load.state).toBe('loading');
  });
});

describe('useNoteComments refreshing in place (R28)', () => {
  it('should keep the list shown while a FORBIDDEN refetch runs, then take the new one', async () => {
    deleteNoteComment.mockRejectedValue(refused('FORBIDDEN', 403));
    const { result } = await setup();
    let answer!: (c: NoteComment[]) => void;
    listNoteComments.mockReturnValue(new Promise((r) => (answer = r)));
    expect(await act(() => result.current.remove('c1'))).toBe('handled');
    expect(ids(result.current.load)).toEqual(['c1', 'c2']);
    await act(async () => answer([comment('c1', { canDelete: false }), comment('c2')]));
    expect(result.current.load.state === 'ready' && result.current.load.comments[0].canDelete).toBe(false);
  });

  it('should keep the list when the refetch fails', async () => {
    updateNoteComment.mockRejectedValue(refused('FORBIDDEN', 403));
    const { result } = await setup();
    listNoteComments.mockRejectedValue(refused('SERVICE_UNAVAILABLE', 503));
    await act(async () => {
      await result.current.edit('c1', 'new words');
    });
    expect(ids(result.current.load)).toEqual(['c1', 'c2']);
  });
});

describe('useNoteComments codes (R30, ADR-0004)', () => {
  it('should ask the page to reload when the load answers ITEM_NOT_A_NOTE', async () => {
    listNoteComments.mockRejectedValue(refused('ITEM_NOT_A_NOTE', 400));
    const { result } = await setup();
    expect(onNotANote).toHaveBeenCalledTimes(1);
    expect(result.current.load.state).toBe('loading');
  });

  it('should treat the retired NOT_A_NOTE as any other failure', async () => {
    listNoteComments.mockRejectedValue(refused('NOT_A_NOTE', 400));
    const { result } = await setup();
    expect(onNotANote).not.toHaveBeenCalled();
    expect(result.current.load.state).toBe('failed');
  });

  it('should tell the page the note is gone when the load answers NOT_FOUND', async () => {
    listNoteComments.mockRejectedValue(refused('NOT_FOUND', 404));
    await setup();
    expect(onGone).toHaveBeenCalledTimes(1);
  });

  it('should drop the comment with a notice when an edit answers NOT_FOUND', async () => {
    updateNoteComment.mockRejectedValue(refused('NOT_FOUND', 404));
    const { result } = await setup();
    expect(await act(() => result.current.edit('c1', 'x'))).toBe('handled');
    expect(ids(result.current.load)).toEqual(['c2']);
    expect(result.current.notice).toBe('This comment was deleted.');
    expect(onGone).not.toHaveBeenCalled();
  });

  it('should leave any other refusal to the caller', async () => {
    deleteNoteComment.mockRejectedValue(refused('SERVICE_UNAVAILABLE', 503));
    const { result } = await setup();
    expect(await act(() => result.current.remove('c1'))).toBe('failed');
    expect(ids(result.current.load)).toEqual(['c1', 'c2']);
  });

  it('should add a comment last, and hand an add refused for the note to the page', async () => {
    createNoteComment.mockResolvedValueOnce(comment('c3'));
    const { result } = await setup();
    expect(await act(() => result.current.add('third'))).toBe(true);
    expect(ids(result.current.load)).toEqual(['c1', 'c2', 'c3']);
    createNoteComment.mockRejectedValueOnce(refused('ITEM_NOT_A_NOTE', 400));
    expect(await act(() => result.current.add('fourth'))).toBe(false);
    expect(onNotANote).toHaveBeenCalledTimes(1);
  });
});
