import { describe, it, expect, vi, beforeEach } from 'vitest';

const api = vi.hoisted(() => ({ POST: vi.fn(), PATCH: vi.fn() }));

vi.mock('./client', () => ({
  api,
  apiErrorFrom: (error: { code: string }, status: number) =>
    Object.assign(new Error(`request failed with ${status}`), { code: error.code, status }),
}));

import { duplicateChecklistItem, updateChecklistItem } from './checklists';

/**
 * FS-F6AEE R19: duplicating goes through the generated client only, and hands
 * back the created items in the server's order — the copy, then its children.
 */
describe('duplicateChecklistItem', () => {
  beforeEach(() => api.POST.mockReset());

  it('should post to the duplicate operation for the item in its plan', async () => {
    api.POST.mockResolvedValue({ data: [], error: undefined, response: { status: 201 } });

    await duplicateChecklistItem('plan-1', 'item-1');

    expect(api.POST).toHaveBeenCalledWith(
      '/api/plans/{id}/checklists/{checklist_id}/duplicate',
      { params: { path: { id: 'plan-1', checklist_id: 'item-1' } } },
    );
  });

  it('should return the created items in the order the server sent them', async () => {
    const created = [{ id: 'copy' }, { id: 'child', parentId: 'copy' }];
    api.POST.mockResolvedValue({ data: created, error: undefined, response: { status: 201 } });

    await expect(duplicateChecklistItem('plan-1', 'item-1')).resolves.toEqual(created);
  });

  it('should return an empty list when the server answers null', async () => {
    api.POST.mockResolvedValue({ data: null, error: undefined, response: { status: 201 } });

    await expect(duplicateChecklistItem('plan-1', 'item-1')).resolves.toEqual([]);
  });

  it('should throw the problem as an ApiError when the server refuses', async () => {
    api.POST.mockResolvedValue({
      data: undefined,
      error: { code: 'VALIDATION_FAILED' },
      response: { status: 400 },
    });

    await expect(duplicateChecklistItem('plan-1', 'item-1')).rejects.toMatchObject({
      code: 'VALIDATION_FAILED',
      status: 400,
    });
  });
});

/**
 * FS-PGG2N R2, R5: a note's summary is set only through the item PATCH, and the
 * item that comes back carries it.
 */
describe('updateChecklistItem', () => {
  beforeEach(() => api.PATCH.mockReset());

  it('should send summary in the PATCH body and return the item with it', async () => {
    const item = { id: 'item-1', type: 'note', summary: 'gist' };
    api.PATCH.mockResolvedValue({ data: item, error: undefined, response: { status: 200 } });

    await expect(
      updateChecklistItem('plan-1', 'item-1', { summary: '  gist  ' }),
    ).resolves.toEqual(item);

    expect(api.PATCH).toHaveBeenCalledWith('/api/plans/{id}/checklists/{checklist_id}', {
      params: { path: { id: 'plan-1', checklist_id: 'item-1' } },
      body: { summary: '  gist  ' },
    });
  });
});
