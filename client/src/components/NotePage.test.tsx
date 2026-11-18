import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import Link from 'next/link';
import { ApiError } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';

/**
 * The note page (FS-PGG2N R16–R27, R36, R37, R49): one note read out of the
 * plan's long-term list, its body on paper, saved as it is typed.
 */

const fetchChecklist = vi.fn();
const updateChecklistItem = vi.fn();
const deleteChecklistItem = vi.fn();
vi.mock('@/services/api', () => ({
  fetchChecklist: (...a: unknown[]) => fetchChecklist(...a),
  updateChecklistItem: (...a: unknown[]) => updateChecklistItem(...a),
  deleteChecklistItem: (...a: unknown[]) => deleteChecklistItem(...a),
}));

const summarizeNote = vi.fn();
vi.mock('@/api/insights', () => ({
  summarizeNote: (...a: unknown[]) => summarizeNote(...a),
}));

const listNoteComments = vi.fn();
vi.mock('@/api/noteComments', () => ({
  listNoteComments: (...a: unknown[]) => listNoteComments(...a),
  createNoteComment: vi.fn(),
}));

const push = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, replace: vi.fn(), refresh: vi.fn() }),
}));

import NotePage from './NotePage';

const mk = (id: string, description: string, extra: Partial<ChecklistItem> = {}) =>
  ({
    id,
    description,
    done: false,
    scope: 'longterm',
    type: 'task',
    planId: 'plan-1',
    archived: false,
    updatedAt: new Date().toISOString(),
    ...extra,
  }) as ChecklistItem;

const ITEMS = [
  mk('m1', 'Learn Go'),
  mk('n1', 'Read the memory model\nthen the pipelines post', {
    parentId: 'm1',
    type: 'note',
  }),
  mk('n2', 'Launch ideas\nA waitlist', { type: 'note' }),
  mk('t1', 'Book flights'),
  mk('a1', 'old note', { type: 'note', archived: true }),
  mk('o1', 'someone else', { type: 'note', planId: 'plan-2' }),
];

const renderPage = (noteId = 'n2', planId = 'plan-1') =>
  render(<NotePage planId={planId} noteId={noteId} />);

/**
 * Clicks, then lets every promise the click started run out: a leave decides
 * over several awaits (flush, retry, the prompt), more than \`act\` waits for.
 */
const clickSettled = (el: Element) =>
  act(async () => {
    fireEvent.click(el);
    await new Promise((r) => setTimeout(r, 0));
  });

const sheet = () => document.querySelector('[data-note-sheet]') as HTMLElement;
const editor = () => screen.getByRole('textbox', { name: 'Note' });

beforeEach(() => {
  vi.clearAllMocks();
  fetchChecklist.mockReset().mockResolvedValue(ITEMS);
  updateChecklistItem.mockReset().mockImplementation(
    async (id: string, fields: Partial<ChecklistItem>) => ({
      ...ITEMS.find((i) => i.id === id)!,
      ...fields,
    })
  );
  deleteChecklistItem.mockReset().mockResolvedValue(undefined);
  summarizeNote.mockReset().mockResolvedValue('An AI gist');
  listNoteComments.mockReset().mockResolvedValue([]);
});

describe('NotePage Back (R17)', () => {
  it("should go back to the milestone's page, named for it, for a milestone's child note", async () => {
    renderPage('n1');
    const back = await screen.findByRole('link', { name: 'Back to Learn Go' });
    expect(back.getAttribute('href')).toBe('/plan/plan-1/m/m1');
  });

  it('should go back to the plan for a top-level note', async () => {
    renderPage('n2');
    const back = await screen.findByRole('link', { name: 'Back to plan' });
    expect(back.getAttribute('href')).toBe('/plan/plan-1');
  });
});

describe('NotePage layout and title (R19, R21, R49)', () => {
  it('should show a quiet loading line first', () => {
    fetchChecklist.mockReturnValue(new Promise(() => {}));
    renderPage();
    expect(screen.getByText('Loading note…')).toBeInTheDocument();
  });

  it('should read from the plan’s long-term list (R18)', async () => {
    renderPage();
    await screen.findByRole('link', { name: 'Back to plan' });
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
  });

  it('should lay out header, body sheet, then meta line, in that order', async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
    const header = document.querySelector('header')!;
    const meta = document.querySelector('[data-note-meta]')!;
    const follows = (a: Element, b: Element) =>
      Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
    expect(follows(header, sheet())).toBe(true);
    expect(follows(sheet(), meta)).toBe(true);
  });

  it('should set the body on paper, upright', async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
    for (const cls of ['bg-note-paper', 'rounded-xl', 'p-5', 'md:p-7', 'text-note', 'whitespace-pre-wrap', 'break-words']) {
      expect(sheet().className).toContain(cls);
    }
    expect(sheet().className).not.toContain('italic');
  });

  it("should name the tab after the card text's first line, up to 60 characters", async () => {
    fetchChecklist.mockResolvedValue([
      mk('n9', `${'x'.repeat(80)}\nsecond`, { type: 'note' }),
    ]);
    renderPage('n9');
    await screen.findByText(/xxx/);
    // The title follows the paint by an effect: wait for it, not the text.
    await waitFor(() => expect(document.title).toHaveLength(60));
    expect(document.title.startsWith('xxx')).toBe(true);
  });
});

describe('NotePage menu (R20)', () => {
  const openMenu = async () => {
    renderPage('n1');
    await screen.findByText(/Read the memory model/);
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Read the memory model' }));
  };

  it('should offer exactly Make a task and Delete', async () => {
    await openMenu();
    expect(screen.getAllByRole('menuitem').map((r) => r.textContent)).toEqual([
      'Make a task',
      'Delete',
    ]);
  });

  it('should make it a task, then go to the Back destination', async () => {
    await openMenu();
    await act(async () => fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' })));
    expect(updateChecklistItem).toHaveBeenCalledWith('n1', { type: 'task' }, 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plan/plan-1/m/m1');
  });

  it('should say the note has comments when Make a task is refused for them (FS-65B3N R31)', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValue(
      new ApiError({ code: 'NOTE_HAS_COMMENTS', status: 400, detail: 'anything' } as never, 400)
    );
    await openMenu();
    await act(async () => fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' })));
    expect(
      screen.getByText('This note has comments. Delete them to make it a task.')
    ).toBeInTheDocument();
    expect(screen.queryByText('Failed to change item type. Please try again.')).toBeNull();
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText(/Read the memory model/)).toBeInTheDocument();
  });

  it('should stay with the error when Make a task is refused', async () => {
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await openMenu();
    await act(async () => fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' })));
    expect(screen.getByText('Failed to change item type. Please try again.')).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
  });

  it('should delete after the confirmation, then go to the Back destination', async () => {
    await openMenu();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    expect(screen.getByText(/permanently delete "Read the memory model"\?/)).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })));
    expect(deleteChecklistItem).toHaveBeenCalledWith('n1', 'plan-1');
    expect(push).toHaveBeenCalledWith('/plan/plan-1/m/m1');
  });

  it('should stay with the error when the delete fails', async () => {
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    await openMenu();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })));
    expect(screen.getByText('Failed to delete task')).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
  });
});

describe('NotePage reading mode (R23, G4)', () => {
  it('should keep line breaks and link http(s) URLs safely, and leave other schemes as text', async () => {
    fetchChecklist.mockResolvedValue([
      mk('n5', 'Sources\nhttps://go.dev/ref/mem and javascript:alert(1)', { type: 'note' }),
    ]);
    renderPage('n5');
    const link = await screen.findByRole('link', { name: 'https://go.dev/ref/mem' });
    expect(link.getAttribute('href')).toBe('https://go.dev/ref/mem');
    expect(link.getAttribute('target')).toBe('_blank');
    expect(link.getAttribute('rel')).toBe('noopener noreferrer');
    expect(screen.queryByRole('link', { name: /javascript/ })).toBeNull();
    expect(sheet().textContent).toBe('Sources\nhttps://go.dev/ref/mem and javascript:alert(1)');
  });

  it('should not start writing when a link is clicked', async () => {
    fetchChecklist.mockResolvedValue([mk('n5', 'see https://go.dev', { type: 'note' })]);
    renderPage('n5');
    fireEvent.click(await screen.findByRole('link', { name: 'https://go.dev' }));
    expect(screen.queryByRole('textbox', { name: 'Note' })).toBeNull();
  });
});

describe('NotePage writing mode and autosave (R23–R27)', () => {
  afterEach(() => vi.useRealTimers());

  const startWriting = async (noteId = 'n2') => {
    renderPage(noteId);
    fireEvent.click(await screen.findByText(/Launch ideas/));
    return editor() as HTMLTextAreaElement;
  };

  it('should enter writing on a click, with the caret at the end, and leave it on blur', async () => {
    const box = await startWriting();
    expect(box.value).toBe('Launch ideas\nA waitlist');
    expect(document.activeElement).toBe(box);
    expect(box.selectionStart).toBe(box.value.length);
    await act(async () => fireEvent.blur(box));
    expect(screen.queryByRole('textbox', { name: 'Note' })).toBeNull();
    expect(sheet()).toBeInTheDocument();
  });

  it('should enter writing on Enter on the focused sheet', async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
    sheet().focus();
    fireEvent.keyDown(sheet(), { key: 'Enter' });
    expect(editor()).toBeInTheDocument();
  });

  it('should save once, 800 ms after the last keystroke', async () => {
    const box = await startWriting();
    vi.useFakeTimers();
    fireEvent.change(box, { target: { value: 'Launch ideas\nA waitlist\nA' } });
    act(() => vi.advanceTimersByTime(500));
    fireEvent.change(box, { target: { value: 'Launch ideas\nA waitlist\nA demo' } });
    act(() => vi.advanceTimersByTime(799));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    await act(async () => vi.advanceTimersByTime(1));
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'n2',
      { description: 'Launch ideas\nA waitlist\nA demo' },
      'plan-1',
      'longterm'
    );
  });

  it('should save on blur without waiting', async () => {
    const box = await startWriting();
    fireEvent.change(box, { target: { value: 'Launch ideas, revised' } });
    await act(async () => fireEvent.blur(box));
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem.mock.calls[0][1]).toEqual({ description: 'Launch ideas, revised' });
  });

  it('should send nothing for unchanged text', async () => {
    const box = await startWriting();
    vi.useFakeTimers();
    fireEvent.change(box, { target: { value: 'Launch ideas\nA waitlist!' } });
    fireEvent.change(box, { target: { value: 'Launch ideas\nA waitlist' } });
    await act(async () => vi.advanceTimersByTime(1000));
    await act(async () => fireEvent.blur(box));
    expect(updateChecklistItem).not.toHaveBeenCalled();
  });

  it('should never send an empty or whitespace body, and show the stored one again', async () => {
    const box = await startWriting();
    vi.useFakeTimers();
    fireEvent.change(box, { target: { value: '   \n  ' } });
    await act(async () => vi.advanceTimersByTime(1000));
    await act(async () => fireEvent.blur(box));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(sheet().textContent).toBe('Launch ideas\nA waitlist');
  });

  it('should save a pending edit when the page goes away (best effort)', async () => {
    const view = renderPage();
    fireEvent.click(await screen.findByText(/Launch ideas/));
    vi.useFakeTimers();
    fireEvent.change(editor(), { target: { value: 'Launch ideas, last words' } });
    view.unmount();
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem.mock.calls[0][1]).toEqual({ description: 'Launch ideas, last words' });
    // The debounce it replaced never fires a second one.
    await act(async () => vi.advanceTimersByTime(1000));
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
  });

  it('should show the word count and the edited time, then Saving…', async () => {
    let finish!: (v: unknown) => void;
    updateChecklistItem.mockReturnValue(new Promise((res) => (finish = res)));
    const box = await startWriting();
    const meta = () => document.querySelector('[data-note-meta]')!.textContent;
    expect(meta()).toMatch(/^4 words · edited/);
    fireEvent.change(box, { target: { value: 'Launch ideas now' } });
    await act(async () => fireEvent.blur(box));
    expect(meta()).toBe('3 words · Saving…');
    await act(async () => finish({ ...ITEMS[2], description: 'Launch ideas now' }));
    expect(meta()).toMatch(/^3 words · edited/);
  });

  it("should keep the text and offer Try again when a save fails", async () => {
    updateChecklistItem.mockRejectedValueOnce(new Error('network'));
    const box = await startWriting();
    fireEvent.change(box, { target: { value: 'Launch ideas, kept' } });
    await act(async () => fireEvent.blur(box));
    expect(document.querySelector('[data-note-meta]')!.textContent).toContain("Couldn't save");
    expect(sheet().textContent).toBe('Launch ideas, kept');

    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Try again' })));
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(updateChecklistItem.mock.calls[1][1]).toEqual({ description: 'Launch ideas, kept' });
    expect(document.querySelector('[data-note-meta]')!.textContent).toMatch(/edited/);
  });

  it('should replace the page with "This note was deleted" and Back when a save answers 404 (R37)', async () => {
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 404));
    renderPage('n1');
    fireEvent.click(await screen.findByText(/Read the memory model/));
    fireEvent.change(editor(), { target: { value: 'into nothing' } });
    await act(async () => fireEvent.blur(editor()));
    expect(screen.getByText('This note was deleted')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to Learn Go' }).getAttribute('href')).toBe(
      '/plan/plan-1/m/m1'
    );
    expect(document.querySelector('[data-note-sheet]')).toBeNull();
  });
});

describe('NotePage not here and load failure (R36)', () => {
  it.each([
    ['unknown', 'nope'],
    ['archived', 'a1'],
    ['a task', 't1'],
    ['a milestone', 'm1'],
    ["another plan's", 'o1'],
  ])('should say "This note isn\'t here" for an id that is %s', async (_why, id) => {
    renderPage(id);
    expect(await screen.findByText("This note isn't here")).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Return to the plan' }).getAttribute('href')).toBe(
      '/plan/plan-1'
    );
  });

  it.each([403, 404, 422])('should say "not here" when the list answers %i', async (status) => {
    fetchChecklist.mockRejectedValue(new ApiError(undefined, status));
    renderPage();
    expect(await screen.findByText("This note isn't here")).toBeInTheDocument();
  });

  it.each([
    ['a network error', new TypeError('Failed to fetch')],
    ['a 5xx', new ApiError(undefined, 503)],
  ])('should offer Try again on %s, and load again on it', async (_why, err) => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    fetchChecklist.mockRejectedValueOnce(err);
    renderPage();
    expect(await screen.findByText("Couldn't load this note")).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Try again' })));
    expect(await screen.findByText(/Launch ideas/)).toBeInTheDocument();
    expect(fetchChecklist).toHaveBeenCalledTimes(2);
  });
});

describe('NotePage "On the card" block (R28, R29, R37)', () => {
  const writeSummary = async (text: string, noteId = 'n2') => {
    renderPage(noteId);
    await screen.findByText(/Launch ideas/);
    fireEvent.click(screen.getByRole('button', { name: /Write a summary|Edit summary/ }));
    const box = screen.getByRole('textbox', { name: 'Summary' });
    fireEvent.change(box, { target: { value: text } });
    await act(async () => fireEvent.keyDown(box, { key: 'Enter' }));
  };

  it('should sit below the meta line', async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
    const meta = document.querySelector('[data-note-meta]')!;
    const block = screen.getByRole('region', { name: 'On the card' });
    expect(meta.compareDocumentPosition(block) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('should save the summary through the item update and show it at once', async () => {
    await writeSummary('  the gist ');
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'the gist' }, 'plan-1', 'longterm');
    expect(screen.getByText('the gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
    // The tab follows the card text.
    expect(document.title).toBe('the gist');
  });

  it('should clear the summary when saved empty, and say there is none again', async () => {
    fetchChecklist.mockResolvedValue([mk('n2', 'Launch ideas\nA waitlist', { type: 'note', summary: 'old gist' })]);
    await writeSummary('');
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: '' }, 'plan-1', 'longterm');
    expect(screen.getByText('No summary yet. The card shows the start of the note.')).toBeInTheDocument();
  });

  it('should put the old summary back with the existing error when the save fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    fetchChecklist.mockResolvedValue([mk('n2', 'Launch ideas\nA waitlist', { type: 'note', summary: 'old gist' })]);
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await writeSummary('new gist');
    expect(screen.getByText('old gist')).toBeInTheDocument();
    expect(screen.queryByText('new gist')).toBeNull();
    expect(screen.getByText('Failed to update task. Please try again.')).toBeInTheDocument();
  });

  it('should show "This note was deleted" when the summary save answers 404', async () => {
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 404));
    await writeSummary('into nothing');
    expect(screen.getByText('This note was deleted')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to plan' })).toBeInTheDocument();
  });
});

describe('NotePage opened from Rename (R14, G8)', () => {
  afterEach(() => window.history.replaceState(null, '', '/'));

  it('should open in writing mode when asked, and drop the ask from the address', async () => {
    window.history.replaceState(null, '', '/plan/plan-1/n/n2?edit=1');
    renderPage();
    const box = (await screen.findByRole('textbox', { name: 'Note' })) as HTMLTextAreaElement;
    expect(box.value).toBe('Launch ideas\nA waitlist');
    await waitFor(() => expect(document.activeElement).toBe(box));
    await waitFor(() => expect(window.location.search).toBe(''));
    expect(window.location.pathname).toBe('/plan/plan-1/n/n2');
    expect(window.location.search).toBe('');
  });

  it('should open reading otherwise', async () => {
    window.history.replaceState(null, '', '/plan/plan-1/n/n2');
    renderPage();
    await screen.findByText(/Launch ideas/);
    expect(screen.queryByRole('textbox', { name: 'Note' })).toBeNull();
  });
});

describe('NotePage Summarize with AI (R30, R31, R37, R44)', () => {
  const withSummary = (summary?: string) =>
    fetchChecklist.mockResolvedValue([
      mk('n2', 'Launch ideas\nA waitlist', { type: 'note', summary }),
    ]);
  const press = async () => {
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: 'Summarize with AI' }))
    );
  };
  const open = async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
  };

  it('should save the returned text at once when there is no summary, and show it', async () => {
    await open();
    await press();
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    expect(summarizeNote).toHaveBeenCalledWith('plan-1', 'n2');
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'An AI gist' }, 'plan-1', 'longterm');
    expect(screen.getByText('An AI gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
  });

  it('should offer the result as a proposal when a summary exists, which Save keeps', async () => {
    withSummary('old gist');
    await open();
    await press();
    expect(updateChecklistItem).not.toHaveBeenCalled();
    const box = screen.getByRole('textbox', { name: 'Summary' }) as HTMLTextAreaElement;
    expect(box.value).toBe('An AI gist');
    expect(document.activeElement).toBe(box);
    // Blur alone keeps nothing: only Save does (ADR-0007).
    fireEvent.blur(box);
    expect(updateChecklistItem).not.toHaveBeenCalled();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Save' })));
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'An AI gist' }, 'plan-1', 'longterm');
    expect(screen.getByText('An AI gist')).toBeInTheDocument();
  });

  it.each([
    ['Cancel', () => fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))],
    ['Esc', () => fireEvent.keyDown(screen.getByRole('textbox', { name: 'Summary' }), { key: 'Escape' })],
  ])('should keep the old summary on %s', async (_how, cancel) => {
    withSummary('old gist');
    await open();
    await press();
    act(() => cancel());
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(screen.getByText('old gist')).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: 'Summary' })).toBeNull();
  });

  it('should read "Summarizing…", disabled, while it runs, so a second press makes no second call', async () => {
    let finish!: (v: string) => void;
    summarizeNote.mockReturnValue(new Promise((res) => (finish = res)));
    await open();
    fireEvent.click(screen.getByRole('button', { name: 'Summarize with AI' }));
    const busy = screen.getByRole('button', { name: 'Summarizing…' });
    expect(busy).toBeDisabled();
    fireEvent.click(busy);
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    await act(async () => finish('An AI gist'));
    expect(screen.getByRole('button', { name: 'Summarize with AI' })).not.toBeDisabled();
  });

  it.each([
    ['NOTE_SUMMARY_FAILED', 503],
    ['SERVICE_UNAVAILABLE', 503],
    ['ITEM_NOT_A_NOTE', 400],
  ])('should say "Couldn\'t summarize" on %s, keep the summary, and keep the other options', async (code, status) => {
    withSummary('old gist');
    summarizeNote.mockRejectedValue(new ApiError({ code, status } as never, status));
    await open();
    await press();
    expect(screen.getByText("Couldn't summarize")).toBeInTheDocument();
    expect(screen.queryByText('AI summaries are paused until tomorrow')).toBeNull();
    expect(screen.getByText('old gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
    expect(updateChecklistItem).not.toHaveBeenCalled();
  });

  it('should add the paused line when the AI budget is spent, on the code alone', async () => {
    summarizeNote.mockRejectedValue(
      new ApiError({ code: 'LLM_BUDGET_EXHAUSTED', status: 429, detail: 'whatever' } as never, 429)
    );
    await open();
    await press();
    expect(screen.getByText("Couldn't summarize")).toBeInTheDocument();
    expect(screen.getByText('AI summaries are paused until tomorrow')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Write a summary' })).toBeInTheDocument();
  });

  it('should show "This note was deleted" when the call answers NOT_FOUND', async () => {
    summarizeNote.mockRejectedValue(new ApiError({ code: 'NOT_FOUND', status: 404 } as never, 404));
    await open();
    await press();
    expect(screen.getByText('This note was deleted')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to plan' })).toBeInTheDocument();
  });

  it('should revert with the existing error when saving the returned text fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await open();
    await press();
    expect(screen.getByText('No summary yet. The card shows the start of the note.')).toBeInTheDocument();
    expect(screen.queryByText('An AI gist')).toBeNull();
    expect(screen.getByText('Failed to update task. Please try again.')).toBeInTheDocument();
  });

  it('should make no model call without a press: not on load, autosave or a summary edit', async () => {
    await open();
    fireEvent.click(screen.getByText(/Launch ideas/));
    fireEvent.change(editor(), { target: { value: 'Launch ideas, more' } });
    await act(async () => fireEvent.blur(editor()));
    fireEvent.click(screen.getByRole('button', { name: 'Write a summary' }));
    const box = screen.getByRole('textbox', { name: 'Summary' });
    fireEvent.change(box, { target: { value: 'by hand' } });
    await act(async () => fireEvent.keyDown(box, { key: 'Enter' }));
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(summarizeNote).not.toHaveBeenCalled();
  });
});

describe('NotePage leave prompt (R32–R35)', () => {
  const LONG = 'x'.repeat(281);
  const withBody = (description: string, summary?: string) =>
    fetchChecklist.mockResolvedValue([
      mk('n2', description, { type: 'note', summary }),
      mk('t1', 'Book flights'),
    ]);
  const open = async () => {
    render(
      <>
        {/* Stands in for the sidebar: any other link in the app. */}
        <Link href="/plans">Plans</Link>
        <NotePage planId="plan-1" noteId="n2" />
      </>
    );
    await screen.findByRole('region', { name: 'On the card' });
  };
  const clickBack = async () =>
    clickSettled(screen.getByRole('link', { name: 'Back to plan' }));
  const PROMPT = { name: 'Add a summary for the card?' };
  const prompt = () => screen.queryByRole('dialog', PROMPT);
  /** The prompt, once it has opened: waits on the page, not on a tick. */
  const opened = () => screen.findByRole('dialog', PROMPT);
  const choose = async (name: string) =>
    clickSettled(within(await opened()).getByRole('button', { name }));

  it('should open on Back for a body over 280 characters with no summary, holding the navigation', async () => {
    withBody(LONG);
    await open();
    await clickBack();
    expect(await opened()).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
    expect(
      Array.from(prompt()!.querySelectorAll('button')).map((b) => b.textContent)
    ).toEqual(['Summarize with AI', 'Write a summary', 'Leave anyway']);
  });

  it('should open on any other in-app link too, such as the sidebar', async () => {
    withBody(LONG);
    await open();
    await clickSettled(screen.getByRole('link', { name: 'Plans' }));
    expect(await opened()).toBeInTheDocument();
    await choose('Leave anyway');
    expect(push).toHaveBeenCalledWith('/plans');
  });

  it('should not open for a body of exactly 280 characters: it just leaves', async () => {
    withBody('x'.repeat(280));
    await open();
    await clickBack();
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should judge the length trimmed', async () => {
    withBody(`  ${'x'.repeat(280)}  \n`);
    await open();
    await clickBack();
    expect(prompt()).toBeNull();
  });

  it('should never open for a note with a summary', async () => {
    withBody(LONG, 'the gist');
    await open();
    await clickBack();
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should open at most once per visit, whatever was chosen', async () => {
    withBody(LONG);
    await open();
    await clickBack();
    await act(async () => fireEvent.keyDown(await opened(), { key: 'Escape' }));
    expect(prompt()).toBeNull();
    await clickBack();
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should leave on Leave anyway', async () => {
    withBody(LONG);
    await open();
    await clickBack();
    await choose('Leave anyway');
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
    expect(summarizeNote).not.toHaveBeenCalled();
  });

  it.each([
    ['Esc', () => fireEvent.keyDown(prompt()!, { key: 'Escape' })],
    ['a click outside', () => fireEvent.click(document.querySelector('[data-leave-scrim]')!)],
  ])('should stay on %s, dropping the navigation', async (_how, dismiss) => {
    withBody(LONG);
    await open();
    await clickBack();
    await opened();
    act(() => dismiss());
    expect(prompt()).toBeNull();
    expect(push).not.toHaveBeenCalled();
  });

  it('should stay on Write a summary, with the summary editor open and focused', async () => {
    withBody(LONG);
    await open();
    await clickBack();
    await choose('Write a summary');
    expect(prompt()).toBeNull();
    expect(push).not.toHaveBeenCalled();
    const box = screen.getByRole('textbox', { name: 'Summary' });
    expect(document.activeElement).toBe(box);
  });

  it('should summarize, save, then go where the person was going (G12)', async () => {
    withBody(LONG);
    await open();
    await clickSettled(screen.getByRole('link', { name: 'Plans' }));
    await choose('Summarize with AI');
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'An AI gist' }, 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plans');
    expect(prompt()).toBeNull();
  });

  it('should read "Summarizing…" while it runs, and take no second press', async () => {
    let finish!: (v: string) => void;
    summarizeNote.mockReturnValue(new Promise((res) => (finish = res)));
    withBody(LONG);
    await open();
    await clickBack();
    fireEvent.click(within(await opened()).getByRole('button', { name: 'Summarize with AI' }));
    const busy = within(prompt()!).getByRole('button', { name: 'Summarizing…' });
    expect(busy).toBeDisabled();
    fireEvent.click(busy);
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    await act(async () => finish('An AI gist'));
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should stay open with "Couldn\'t summarize" and only the other two choices when the AI fails', async () => {
    summarizeNote.mockRejectedValue(new ApiError({ code: 'NOTE_SUMMARY_FAILED', status: 503 } as never, 503));
    withBody(LONG);
    await open();
    await clickBack();
    await choose('Summarize with AI');
    const dialog = await opened();
    expect(await within(dialog).findByText("Couldn't summarize")).toBeInTheDocument();
    expect(within(dialog).queryByText('AI summaries are paused until tomorrow')).toBeNull();
    expect(Array.from(dialog.querySelectorAll('button')).map((b) => b.textContent)).toEqual([
      'Write a summary',
      'Leave anyway',
    ]);
    expect(push).not.toHaveBeenCalled();
  });

  it('should add the paused line when the budget is spent', async () => {
    summarizeNote.mockRejectedValue(new ApiError({ code: 'LLM_BUDGET_EXHAUSTED', status: 429 } as never, 429));
    withBody(LONG);
    await open();
    await clickBack();
    await choose('Summarize with AI');
    expect(
      await within(await opened()).findByText('AI summaries are paused until tomorrow')
    ).toBeInTheDocument();
  });

  it('should say "Couldn\'t save the summary" when saving it fails, with the same two choices', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    withBody(LONG);
    await open();
    await clickBack();
    await choose('Summarize with AI');
    const dialog = await opened();
    expect(await within(dialog).findByText("Couldn't save the summary")).toBeInTheDocument();
    expect(Array.from(dialog.querySelectorAll('button')).map((b) => b.textContent)).toEqual([
      'Write a summary',
      'Leave anyway',
    ]);
    await choose('Leave anyway');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should be a labelled modal dialog that takes focus', async () => {
    withBody(LONG);
    await open();
    await clickBack();
    expect((await opened()).getAttribute('aria-modal')).toBe('true');
    expect(prompt()!.contains(document.activeElement)).toBe(true);
  });
});

describe('NotePage flush before leaving (R24, R27)', () => {
  const typeThenBack = async (text: string) => {
    renderPage();
    fireEvent.click(await screen.findByText(/Launch ideas/));
    vi.useFakeTimers();
    fireEvent.change(editor(), { target: { value: text } });
    vi.useRealTimers();
    await clickSettled(screen.getByRole('link', { name: 'Back to plan' }));
  };
  afterEach(() => vi.useRealTimers());

  it('should send a pending body save before leaving', async () => {
    await typeThenBack('Launch ideas, before I go');
    expect(updateChecklistItem).toHaveBeenCalledWith(
      'n2',
      { description: 'Launch ideas, before I go' },
      'plan-1',
      'longterm'
    );
    const saved = updateChecklistItem.mock.invocationCallOrder[0];
    expect(saved).toBeLessThan(push.mock.invocationCallOrder[0]);
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should try once more when the flush fails, and leave when that one lands', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValueOnce(new Error('network'));
    await typeThenBack('Launch ideas, second time lucky');
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should stop with "Couldn\'t save your changes" and Leave anyway when both fail', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockRejectedValue(new Error('network'));
    await typeThenBack('Launch ideas, unsaved');
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText("Couldn't save your changes")).toBeInTheDocument();
    await clickSettled(screen.getByRole('button', { name: 'Leave anyway' }));
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
  });

  it('should leave at once when there is nothing to save', async () => {
    renderPage();
    await screen.findByText(/Launch ideas/);
    await clickSettled(screen.getByRole('link', { name: 'Back to plan' }));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });
});

describe('NotePage comments slot (FS-PGG2N R19, FS-65B3N R24, R30, R33)', () => {
  const follows = (a: Element, b: Element) =>
    Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);

  it.each([
    ['a top-level note', 'n2'],
    ["a milestone's child note", 'n1'],
  ])('should show the comments section below the "On the card" block on %s', async (_why, id) => {
    renderPage(id);
    const comments = await screen.findByRole('region', { name: 'Comments' });
    expect(follows(screen.getByRole('region', { name: 'On the card' }), comments)).toBe(true);
    await waitFor(() => expect(listNoteComments).toHaveBeenCalledWith('plan-1', id));
  });

  it('should keep the body editable while the comments are still loading or failed', async () => {
    listNoteComments.mockReturnValue(new Promise(() => {}));
    renderPage();
    fireEvent.click(await screen.findByText(/Launch ideas/));
    expect(editor()).toBeInTheDocument();
    expect(screen.getByText('Loading comments…')).toBeInTheDocument();
  });

  it('should load the page again when the comments answer ITEM_NOT_A_NOTE', async () => {
    listNoteComments.mockRejectedValueOnce(
      new ApiError({ code: 'ITEM_NOT_A_NOTE', status: 400 } as never, 400)
    );
    renderPage();
    await screen.findByRole('region', { name: 'Comments' });
    await waitFor(() => expect(fetchChecklist).toHaveBeenCalledTimes(2));
  });
});

describe('NotePage menu with an edit still waiting to save (R20, R27)', () => {
  afterEach(() => vi.useRealTimers());

  /** Types into the body and opens the menu before the autosave timer runs out. */
  const typeThenOpenMenu = async (text: string) => {
    const view = renderPage();
    fireEvent.click(await screen.findByText(/Launch ideas/));
    vi.useFakeTimers();
    fireEvent.change(editor(), { target: { value: text } });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Launch ideas' }));
    return view;
  };
  const bodySaves = () =>
    updateChecklistItem.mock.calls.filter(([, fields]) => 'description' in fields);

  it('should drop the waiting save on Delete, so no write follows the delete', async () => {
    const view = await typeThenOpenMenu('Launch ideas, then deleted');
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })));
    expect(deleteChecklistItem).toHaveBeenCalledWith('n2', 'plan-1');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
    const deletedAt = deleteChecklistItem.mock.invocationCallOrder[0];
    await act(async () => vi.advanceTimersByTime(2000));
    view.unmount();
    const savesAfter = updateChecklistItem.mock.calls.filter(
      ([, fields], i) =>
        'description' in fields && updateChecklistItem.mock.invocationCallOrder[i] > deletedAt
    );
    expect(savesAfter).toEqual([]);
  });

  it('should still save the waiting edit when the delete fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    await typeThenOpenMenu('Launch ideas, kept');
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })));
    await act(async () => vi.advanceTimersByTime(2000));
    expect(screen.getByText('Failed to delete task')).toBeInTheDocument();
    expect(bodySaves()).toEqual([
      ['n2', { description: 'Launch ideas, kept' }, 'plan-1', 'longterm'],
    ]);
  });

  it('should send the waiting save before Make a task leaves', async () => {
    await typeThenOpenMenu('Launch ideas, now a task');
    vi.useRealTimers();
    await clickSettled(screen.getByRole('menuitem', { name: 'Make a task' }));
    expect(bodySaves()).toEqual([
      ['n2', { description: 'Launch ideas, now a task' }, 'plan-1', 'longterm'],
    ]);
    const saveCall = updateChecklistItem.mock.calls.findIndex(([, f]) => 'description' in f);
    expect(updateChecklistItem.mock.invocationCallOrder[saveCall]).toBeLessThan(
      push.mock.invocationCallOrder[0]
    );
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should stop with "Couldn\'t save your changes" when Make a task lands but the body will not save', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    updateChecklistItem.mockImplementation(async (id: string, fields: Partial<ChecklistItem>) => {
      if ('description' in fields) throw new Error('network');
      return { ...ITEMS.find((i) => i.id === id)!, ...fields };
    });
    await typeThenOpenMenu('Launch ideas, unsaved task');
    vi.useRealTimers();
    await clickSettled(screen.getByRole('menuitem', { name: 'Make a task' }));
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText("Couldn't save your changes")).toBeInTheDocument();
    await clickSettled(screen.getByRole('button', { name: 'Leave anyway' }));
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });
});
