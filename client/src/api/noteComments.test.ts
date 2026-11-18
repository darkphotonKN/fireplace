import { describe, it, expect, vi, beforeEach } from 'vitest';

const api = vi.hoisted(() => ({ GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn(), DELETE: vi.fn() }));
const markTouchedToday = vi.hoisted(() => vi.fn());

vi.mock('./client', () => ({
  api,
  apiErrorFrom: (error: { code: string }, status: number) =>
    Object.assign(new Error(`request failed with ${status}`), { code: error.code, status }),
}));
vi.mock('@/lib/touchedToday', () => ({ markTouchedToday }));

import {
  createNoteComment,
  deleteNoteComment,
  listNoteComments,
  updateNoteComment,
} from './noteComments';

/** FS-65B3N §API surface: a note's comments, through the generated client only. */
const PATH = '/api/plans/{id}/checklists/{checklist_id}/comments';
const params = { params: { path: { id: 'plan-1', checklist_id: 'note-1' } } };

beforeEach(() => {
  api.GET.mockReset();
  api.POST.mockReset();
  api.PATCH.mockReset();
  api.DELETE.mockReset();
  markTouchedToday.mockReset();
});

describe('listNoteComments', () => {
  it('should read the note’s comments, as the server orders them', async () => {
    const comments = [{ id: 'c1' }, { id: 'c2' }];
    api.GET.mockResolvedValue({ data: comments, error: undefined, response: { status: 200 } });
    await expect(listNoteComments('plan-1', 'note-1')).resolves.toEqual(comments);
    expect(api.GET).toHaveBeenCalledWith(PATH, params);
    expect(markTouchedToday).not.toHaveBeenCalled();
  });

  it('should give an empty list when the server answers null', async () => {
    api.GET.mockResolvedValue({ data: null, error: undefined, response: { status: 200 } });
    await expect(listNoteComments('plan-1', 'note-1')).resolves.toEqual([]);
  });

  it('should throw the problem with its code', async () => {
    api.GET.mockResolvedValue({ data: undefined, error: { code: 'ITEM_NOT_A_NOTE' }, response: { status: 400 } });
    await expect(listNoteComments('plan-1', 'note-1')).rejects.toMatchObject({ code: 'ITEM_NOT_A_NOTE', status: 400 });
  });
});

describe('createNoteComment', () => {
  it('should post the body and stamp touched-today once it lands', async () => {
    const made = { id: 'c3', body: 'hi' };
    api.POST.mockResolvedValue({ data: made, error: undefined, response: { status: 201 } });
    await expect(createNoteComment('plan-1', 'note-1', 'hi')).resolves.toEqual(made);
    expect(api.POST).toHaveBeenCalledWith(PATH, { ...params, body: { body: 'hi' } });
    expect(markTouchedToday).toHaveBeenCalledTimes(1);
  });

  it('should throw without a stamp when refused', async () => {
    api.POST.mockResolvedValue({ data: undefined, error: { code: 'VALIDATION_FAILED' }, response: { status: 400 } });
    await expect(createNoteComment('plan-1', 'note-1', ' ')).rejects.toMatchObject({ code: 'VALIDATION_FAILED' });
    expect(markTouchedToday).not.toHaveBeenCalled();
  });
});

const ONE = '/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}';
const oneParams = { params: { path: { id: 'plan-1', checklist_id: 'note-1', comment_id: 'c1' } } };

describe('updateNoteComment', () => {
  it('should patch the body and stamp touched-today once it lands', async () => {
    const edited = { id: 'c1', body: 'better', editedAt: '2026-11-19T10:00:00Z' };
    api.PATCH.mockResolvedValue({ data: edited, error: undefined, response: { status: 200 } });
    await expect(updateNoteComment('plan-1', 'note-1', 'c1', 'better')).resolves.toEqual(edited);
    expect(api.PATCH).toHaveBeenCalledWith(ONE, { ...oneParams, body: { body: 'better' } });
    expect(markTouchedToday).toHaveBeenCalledTimes(1);
  });

  it('should throw the code without a stamp when refused', async () => {
    api.PATCH.mockResolvedValue({ data: undefined, error: { code: 'FORBIDDEN' }, response: { status: 403 } });
    await expect(updateNoteComment('plan-1', 'note-1', 'c1', 'x')).rejects.toMatchObject({ code: 'FORBIDDEN', status: 403 });
    expect(markTouchedToday).not.toHaveBeenCalled();
  });
});

describe('deleteNoteComment', () => {
  it('should delete the comment and stamp touched-today once it lands', async () => {
    api.DELETE.mockResolvedValue({ data: undefined, error: undefined, response: { status: 204 } });
    await expect(deleteNoteComment('plan-1', 'note-1', 'c1')).resolves.toBeUndefined();
    expect(api.DELETE).toHaveBeenCalledWith(ONE, oneParams);
    expect(markTouchedToday).toHaveBeenCalledTimes(1);
  });

  it('should throw the code without a stamp when refused', async () => {
    api.DELETE.mockResolvedValue({ data: undefined, error: { code: 'NOT_FOUND' }, response: { status: 404 } });
    await expect(deleteNoteComment('plan-1', 'note-1', 'c1')).rejects.toMatchObject({ code: 'NOT_FOUND' });
    expect(markTouchedToday).not.toHaveBeenCalled();
  });
});
