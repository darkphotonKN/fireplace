import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import { ApiError } from '@/api/client';
import type { NoteComment } from '@/api/noteComments';

/**
 * The note page's comments section (FS-65B3N R12–R13, R24–R28, R30, R33):
 * oldest first, with the add box at the bottom.
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

import NoteComments from './NoteComments';

const minutesAgo = (n: number) => new Date(Date.now() - n * 60_000).toISOString();
const mkComment = (id: string, extra: Partial<NoteComment> = {}): NoteComment => ({
  id,
  authorId: `u-${id}`,
  authorName: 'kranti',
  body: `comment ${id}`,
  createdAt: minutesAgo(5),
  editedAt: null,
  canEdit: false,
  canDelete: false,
  ...extra,
});

const onGone = vi.fn();
const onNotANote = vi.fn();
const renderSection = async () => {
  render(<NoteComments planId="plan-1" noteId="n1" onGone={onGone} onNotANote={onNotANote} />);
  await act(async () => {});
};
const box = () => screen.getByRole('textbox', { name: 'Add a comment' }) as HTMLTextAreaElement;
const commentButton = () => screen.getByRole('button', { name: 'Comment' });
const items = () => within(screen.getByRole('list', { name: 'Comments' })).queryAllByRole('listitem');

beforeEach(() => {
  vi.clearAllMocks();
  listNoteComments.mockReset().mockResolvedValue([
    mkComment('c1', { authorName: 'ada', body: 'first\nwith a second line', createdAt: minutesAgo(90) }),
    mkComment('c2', { authorName: null, body: 'second', createdAt: minutesAgo(5), editedAt: minutesAgo(1) }),
  ]);
  updateNoteComment.mockReset().mockImplementation(
    async (_p: string, _n: string, id: string, body: string) =>
      mkComment(id, { body: body.trim(), editedAt: new Date().toISOString(), canEdit: true, canDelete: true })
  );
  deleteNoteComment.mockReset().mockResolvedValue(undefined);
  createNoteComment.mockReset().mockImplementation(async (_p: string, _n: string, body: string) =>
    mkComment('c3', { authorName: 'me', body: body.trim(), createdAt: new Date().toISOString() })
  );
});

describe('NoteComments reading (R11–R13, R25–R27, R33)', () => {
  it('should load the note’s comments on open and show them oldest first', async () => {
    await renderSection();
    expect(listNoteComments).toHaveBeenCalledWith('plan-1', 'n1');
    expect(items().map((li) => li.querySelector('[data-comment-body]')!.textContent)).toEqual([
      'first\nwith a second line',
      'second',
    ]);
  });

  it('should show an initial avatar, "name · relative time", and keep line breaks', async () => {
    await renderSection();
    const first = items()[0];
    expect(first.querySelector('[data-avatar]')!.textContent).toBe('A');
    expect(first.querySelector('[data-avatar]')!.className).toContain('rounded-full');
    expect(first.querySelector('[data-comment-meta]')!.textContent).toBe('ada · 1 h ago');
    expect(first.querySelector('[data-comment-body]')!.className).toContain('whitespace-pre-wrap');
  });

  it('should show "Deleted user" with a muted avatar for a missing author, and " · edited"', async () => {
    await renderSection();
    const second = items()[1];
    expect(second.querySelector('[data-comment-meta]')!.textContent).toBe(
      'Deleted user · 5 min ago · edited'
    );
    const avatar = second.querySelector('[data-avatar]')!;
    expect(avatar.textContent).toBe('D');
    expect(avatar.className).toContain('bg-muted');
    expect(avatar.className).toContain('text-muted-foreground');
  });

  it('should give the full date and time as the tooltip and accessible text', async () => {
    const at = new Date(2026, 10, 3, 16, 5).toISOString();
    listNoteComments.mockResolvedValue([mkComment('c1', { createdAt: at })]);
    await renderSection();
    const time = items()[0].querySelector('time')!;
    expect(time.getAttribute('dateTime')).toBe(at);
    expect(time.getAttribute('title')).toBe('Nov 3, 2026, 4:05 PM');
    expect(time.getAttribute('aria-label')).toBe('Nov 3, 2026, 4:05 PM');
  });

  it('should link http(s) URLs in a new tab, nofollow, and keep markup literal', async () => {
    listNoteComments.mockResolvedValue([
      mkComment('c1', { body: '<b>x</b> see https://go.dev/ref/mem. javascript:alert(1)' }),
    ]);
    await renderSection();
    const link = screen.getByRole('link', { name: 'https://go.dev/ref/mem' });
    expect(link.getAttribute('href')).toBe('https://go.dev/ref/mem');
    expect(link.getAttribute('target')).toBe('_blank');
    expect(link.getAttribute('rel')).toBe('noopener noreferrer nofollow');
    expect(screen.queryByRole('link', { name: /javascript/ })).toBeNull();
    expect(items()[0].querySelector('[data-comment-body]')!.textContent).toBe(
      '<b>x</b> see https://go.dev/ref/mem. javascript:alert(1)'
    );
    expect(items()[0].querySelector('b')).toBeNull();
  });

  it('should show only the add box when there are no comments', async () => {
    listNoteComments.mockResolvedValue([]);
    await renderSection();
    expect(items()).toHaveLength(0);
    expect(box()).toBeInTheDocument();
  });

  it('should say "Couldn\'t load comments" with Retry on a failed load, and load again', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    listNoteComments.mockRejectedValueOnce(new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503));
    await renderSection();
    expect(screen.getByText("Couldn't load comments")).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Retry' })));
    expect(listNoteComments).toHaveBeenCalledTimes(2);
    expect(items()).toHaveLength(2);
  });

  it('should ask the page to reload on ITEM_NOT_A_NOTE, on the code alone', async () => {
    listNoteComments.mockRejectedValue(new ApiError({ code: 'ITEM_NOT_A_NOTE', status: 400 } as never, 400));
    await renderSection();
    expect(onNotANote).toHaveBeenCalledTimes(1);
  });

  it('should tell the page the note is gone on NOT_FOUND', async () => {
    listNoteComments.mockRejectedValue(new ApiError({ code: 'NOT_FOUND', status: 404 } as never, 404));
    await renderSection();
    expect(onGone).toHaveBeenCalledTimes(1);
  });
});

describe('NoteComments adding (R28)', () => {
  it('should sit below the last comment, with the "Add a comment…" placeholder', async () => {
    await renderSection();
    expect(box().placeholder).toBe('Add a comment…');
    const list = screen.getByRole('list', { name: 'Comments' });
    expect(list.compareDocumentPosition(box()) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it.each([
    ['empty', '', true],
    ['whitespace', '  \n ', true],
    ['one character', 'a', false],
    ['exactly 2000 code points, emoji included', '😀'.repeat(2000), false],
    ['over 2000', 'a'.repeat(2001), true],
    ['2000 once trimmed', `  ${'a'.repeat(2000)}  `, false],
  ])('should enable Comment for %s: disabled=%s', async (_why, text, disabled) => {
    await renderSection();
    fireEvent.change(box(), { target: { value: text } });
    expect((commentButton() as HTMLButtonElement).disabled).toBe(disabled);
  });

  it('should show the count only past 1800', async () => {
    await renderSection();
    fireEvent.change(box(), { target: { value: 'a'.repeat(1800) } });
    expect(screen.queryByText('1800/2000')).toBeNull();
    fireEvent.change(box(), { target: { value: 'a'.repeat(1801) } });
    expect(screen.getByText('1801/2000')).toBeInTheDocument();
    fireEvent.change(box(), { target: { value: '😀'.repeat(2001) } });
    expect(screen.getByText('2001/2000')).toBeInTheDocument();
  });

  it('should add on Comment: the new one last, the box cleared and still focused', async () => {
    await renderSection();
    box().focus();
    fireEvent.change(box(), { target: { value: '  a third  ' } });
    await act(async () => fireEvent.click(commentButton()));
    expect(createNoteComment).toHaveBeenCalledWith('plan-1', 'n1', 'a third');
    expect(items()).toHaveLength(3);
    expect(items()[2].querySelector('[data-comment-body]')!.textContent).toBe('a third');
    expect(box().value).toBe('');
    expect(document.activeElement).toBe(box());
  });

  it.each([
    ['Ctrl+Enter', { ctrlKey: true }],
    ['Cmd+Enter', { metaKey: true }],
  ])('should submit on %s', async (_how, mods) => {
    await renderSection();
    fireEvent.change(box(), { target: { value: 'quick one' } });
    await act(async () => fireEvent.keyDown(box(), { key: 'Enter', ...mods }));
    expect(createNoteComment).toHaveBeenCalledTimes(1);
  });

  it('should leave Enter alone, for a new line', async () => {
    await renderSection();
    fireEvent.change(box(), { target: { value: 'line one' } });
    const plain = fireEvent.keyDown(box(), { key: 'Enter' });
    expect(plain).toBe(true); // not prevented: the browser adds the line
    expect(createNoteComment).not.toHaveBeenCalled();
  });

  it('should not submit on Ctrl+Enter when the text is not sendable', async () => {
    await renderSection();
    fireEvent.change(box(), { target: { value: '   ' } });
    await act(async () => fireEvent.keyDown(box(), { key: 'Enter', ctrlKey: true }));
    expect(createNoteComment).not.toHaveBeenCalled();
  });

  it('should keep the text with an error when the add fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    createNoteComment.mockRejectedValue(new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503));
    await renderSection();
    fireEvent.change(box(), { target: { value: 'keep me' } });
    await act(async () => fireEvent.click(commentButton()));
    expect(box().value).toBe('keep me');
    expect(screen.getByText("Couldn't add your comment. Try again.")).toBeInTheDocument();
    expect(items()).toHaveLength(2);
  });

  it('should ask the page to reload when the add answers ITEM_NOT_A_NOTE', async () => {
    createNoteComment.mockRejectedValue(new ApiError({ code: 'ITEM_NOT_A_NOTE', status: 400 } as never, 400));
    await renderSection();
    fireEvent.change(box(), { target: { value: 'too late' } });
    await act(async () => fireEvent.click(commentButton()));
    expect(onNotANote).toHaveBeenCalledTimes(1);
  });

  it('should send one add for a double press', async () => {
    let finish!: (c: NoteComment) => void;
    createNoteComment.mockReturnValue(new Promise((res) => (finish = res)));
    await renderSection();
    fireEvent.change(box(), { target: { value: 'once' } });
    fireEvent.click(commentButton());
    fireEvent.click(commentButton());
    expect(createNoteComment).toHaveBeenCalledTimes(1);
    await act(async () => finish(mkComment('c3', { body: 'once' })));
  });
});

describe('NoteComments edit and delete (R16–R17, R29–R30)', () => {
  const withComments = (...comments: NoteComment[]) => listNoteComments.mockResolvedValue(comments);
  const row = (i: number) => within(items()[i]);
  const editBox = () => screen.getByRole('textbox', { name: 'Edit comment' }) as HTMLTextAreaElement;

  it.each([
    [true, true],
    [true, false],
    [false, true],
    [false, false],
  ])('should offer Edit only when canEdit=%s and Delete only when canDelete=%s', async (canEdit, canDelete) => {
    withComments(mkComment('c1', { canEdit, canDelete }));
    await renderSection();
    expect(Boolean(row(0).queryByRole('button', { name: 'Edit' }))).toBe(canEdit);
    expect(Boolean(row(0).queryByRole('button', { name: 'Delete' }))).toBe(canDelete);
  });

  it('should show the controls on hover and focus on pointer devices, and always on touch', async () => {
    withComments(mkComment('c1', { canEdit: true, canDelete: true }));
    await renderSection();
    const controls = items()[0].querySelector('[data-comment-actions]')!;
    for (const cls of [
      'opacity-0',
      'group-hover/comment:opacity-100',
      'group-focus-within/comment:opacity-100',
      '[@media(hover:none)]:opacity-100',
    ]) {
      expect(controls.className).toContain(cls);
    }
    expect(items()[0].className).toContain('group/comment');
    // Buttons, so Tab reaches them.
    expect(row(0).getByRole('button', { name: 'Edit' }).tagName).toBe('BUTTON');
  });

  it('should edit in place: prefilled, saved, then shown with " · edited"', async () => {
    withComments(mkComment('c1', { body: 'old words', canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    expect(editBox().value).toBe('old words');
    expect(document.activeElement).toBe(editBox());
    fireEvent.change(editBox(), { target: { value: ' new words ' } });
    await act(async () => fireEvent.click(row(0).getByRole('button', { name: 'Save' })));
    expect(updateNoteComment).toHaveBeenCalledWith('plan-1', 'n1', 'c1', 'new words');
    expect(screen.queryByRole('textbox', { name: 'Edit comment' })).toBeNull();
    expect(items()[0].querySelector('[data-comment-body]')!.textContent).toBe('new words');
    expect(items()[0].querySelector('[data-comment-meta]')!.textContent).toMatch(/ · edited$/);
  });

  it('should save on Ctrl/Cmd+Enter', async () => {
    withComments(mkComment('c1', { canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: 'keyboard edit' } });
    await act(async () => fireEvent.keyDown(editBox(), { key: 'Enter', metaKey: true }));
    expect(updateNoteComment).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['Escape', () => fireEvent.keyDown(editBox(), { key: 'Escape' })],
    ['Cancel', () => fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))],
  ])('should cancel on %s and restore the body', async (_how, cancel) => {
    withComments(mkComment('c1', { body: 'as it was', canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: 'never mind' } });
    act(() => cancel());
    expect(screen.queryByRole('textbox', { name: 'Edit comment' })).toBeNull();
    expect(items()[0].querySelector('[data-comment-body]')!.textContent).toBe('as it was');
    expect(updateNoteComment).not.toHaveBeenCalled();
  });

  it.each([
    ['empty', '  '],
    ['over 2000', 'a'.repeat(2001)],
  ])('should disable Save for %s', async (_why, text) => {
    withComments(mkComment('c1', { canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: text } });
    expect(row(0).getByRole('button', { name: 'Save' })).toBeDisabled();
    await act(async () => fireEvent.keyDown(editBox(), { key: 'Enter', ctrlKey: true }));
    expect(updateNoteComment).not.toHaveBeenCalled();
  });

  it('should send nothing for the same text, and stay unedited', async () => {
    withComments(mkComment('c1', { body: 'same', canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: '  same ' } });
    await act(async () => fireEvent.click(row(0).getByRole('button', { name: 'Save' })));
    expect(updateNoteComment).not.toHaveBeenCalled();
    expect(items()[0].querySelector('[data-comment-meta]')!.textContent).not.toMatch(/edited/);
  });

  it('should keep the edit text with an inline error when a save fails otherwise', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateNoteComment.mockRejectedValue(new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503));
    withComments(mkComment('c1', { canEdit: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: 'keep this' } });
    await act(async () => fireEvent.click(row(0).getByRole('button', { name: 'Save' })));
    expect(editBox().value).toBe('keep this');
    expect(row(0).getByText("Couldn't save your edit. Try again.")).toBeInTheDocument();
  });

  it('should ask before deleting, and remove the comment once confirmed', async () => {
    withComments(mkComment('c1', { body: 'doomed', canDelete: true }), mkComment('c2'));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Delete' }));
    expect(screen.getByText(/permanently delete "doomed"\?/)).toBeInTheDocument();
    expect(deleteNoteComment).not.toHaveBeenCalled();
    await act(async () =>
      fireEvent.click(within(screen.getByText(/permanently delete/).parentElement!).getByRole('button', { name: 'Delete' }))
    );
    expect(deleteNoteComment).toHaveBeenCalledWith('plan-1', 'n1', 'c1');
    expect(items()).toHaveLength(1);
    expect(items()[0].querySelector('[data-comment-body]')!.textContent).toBe('comment c2');
  });

  it('should keep the comment when the delete is cancelled', async () => {
    withComments(mkComment('c1', { canDelete: true }));
    await renderSection();
    fireEvent.click(row(0).getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(deleteNoteComment).not.toHaveBeenCalled();
    expect(items()).toHaveLength(1);
    expect(screen.queryByText(/permanently delete/)).toBeNull();
  });

  const confirmDelete = async () => {
    fireEvent.click(row(0).getByRole('button', { name: 'Delete' }));
    await act(async () =>
      fireEvent.click(within(screen.getByText(/permanently delete/).parentElement!).getByRole('button', { name: 'Delete' }))
    );
  };
  const saveEdit = async () => {
    fireEvent.click(row(0).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: 'too late' } });
    await act(async () => fireEvent.click(row(0).getByRole('button', { name: 'Save' })));
  };

  it.each([
    ['a save', 'update', saveEdit],
    ['a delete', 'delete', confirmDelete],
  ])('should remove the comment with "This comment was deleted." when %s answers NOT_FOUND', async (_why, op, act_) => {
    const gone = new ApiError({ code: 'NOT_FOUND', status: 404 } as never, 404);
    (op === 'update' ? updateNoteComment : deleteNoteComment).mockRejectedValue(gone);
    withComments(mkComment('c1', { canEdit: true, canDelete: true }), mkComment('c2'));
    await renderSection();
    await act_();
    expect(items()).toHaveLength(1);
    expect(screen.getByText('This comment was deleted.')).toBeInTheDocument();
    expect(onGone).not.toHaveBeenCalled();
  });

  it.each([
    ['a save', 'update', saveEdit],
    ['a delete', 'delete', confirmDelete],
  ])('should load the list again when %s answers FORBIDDEN', async (_why, op, act_) => {
    const stale = new ApiError({ code: 'FORBIDDEN', status: 403 } as never, 403);
    (op === 'update' ? updateNoteComment : deleteNoteComment).mockRejectedValue(stale);
    withComments(mkComment('c1', { canEdit: true, canDelete: true }));
    await renderSection();
    listNoteComments.mockResolvedValue([mkComment('c1')]);
    await act_();
    await act(async () => {});
    expect(listNoteComments).toHaveBeenCalledTimes(2);
    expect(row(0).queryByRole('button', { name: 'Edit' })).toBeNull();
  });

  it('should keep the list, the typed comment and other open edits while a FORBIDDEN refetch runs (R28)', async () => {
    deleteNoteComment.mockRejectedValue(new ApiError({ code: 'FORBIDDEN', status: 403 } as never, 403));
    withComments(
      mkComment('c1', { canEdit: true, canDelete: true }),
      mkComment('c2', { canEdit: true })
    );
    await renderSection();
    fireEvent.change(box(), { target: { value: 'half a thought' } });
    fireEvent.click(row(1).getByRole('button', { name: 'Edit' }));
    fireEvent.change(editBox(), { target: { value: 'an edit in progress' } });
    let answer!: (c: NoteComment[]) => void;
    listNoteComments.mockReturnValue(new Promise((r) => (answer = r)));
    await confirmDelete();
    expect(listNoteComments).toHaveBeenCalledTimes(2);
    expect(screen.queryByText('Loading comments…')).toBeNull();
    expect(box().value).toBe('half a thought');
    await act(async () =>
      answer([mkComment('c1'), mkComment('c2', { canEdit: true })])
    );
    expect(row(0).queryByRole('button', { name: 'Delete' })).toBeNull();
    expect(box().value).toBe('half a thought');
    expect(editBox().value).toBe('an edit in progress');
  });

  it('should keep the list when a refetch fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    deleteNoteComment.mockRejectedValue(new ApiError({ code: 'FORBIDDEN', status: 403 } as never, 403));
    withComments(mkComment('c1', { canDelete: true }), mkComment('c2'));
    await renderSection();
    fireEvent.change(box(), { target: { value: 'still here' } });
    listNoteComments.mockRejectedValue(new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503));
    await confirmDelete();
    await act(async () => {});
    expect(items()).toHaveLength(2);
    expect(box().value).toBe('still here');
  });

  it('should keep the comment with an inline error when a delete fails otherwise', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    deleteNoteComment.mockRejectedValue(new ApiError({ code: 'SERVICE_UNAVAILABLE', status: 503 } as never, 503));
    withComments(mkComment('c1', { canDelete: true }));
    await renderSection();
    await confirmDelete();
    expect(items()).toHaveLength(1);
    expect(row(0).getByText("Couldn't delete the comment. Try again.")).toBeInTheDocument();
  });
});
