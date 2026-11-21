import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import Link from 'next/link';
import { ApiError } from '@/api/client';
import type { ChecklistItem } from '@/api/checklists';
import { api, item as mk, quietErrors, router } from '@/test/pageHarness';
import { deferred, settle } from '@/test/async';

/**
 * The note page (FS-PGG2N R16–R27, R36, R37, R49): one note read out of the
 * plan's long-term list, its body on paper, saved as it is typed.
 */

const summarizeNote = vi.fn();
vi.mock('@/api/insights', () => ({
  summarizeNote: (...a: unknown[]) => summarizeNote(...a),
}));

const listNoteComments = vi.fn();
vi.mock('@/api/noteComments', () => ({
  listNoteComments: (...a: unknown[]) => listNoteComments(...a),
  createNoteComment: vi.fn(),
}));

import NotePage from './NotePage';

const { fetchChecklist, updateChecklistItem, deleteChecklistItem } = api;
const { push } = router;

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

/** The update answers with the stored item, the sent fields applied. */
const echoUpdate = async (id: string, fields: Partial<ChecklistItem>) => ({
  ...ITEMS.find((i) => i.id === id)!,
  ...fields,
});

beforeEach(() => {
  fetchChecklist.mockResolvedValue(ITEMS);
  updateChecklistItem.mockImplementation(echoUpdate);
  deleteChecklistItem.mockResolvedValue(undefined);
  summarizeNote.mockReset().mockResolvedValue('An AI gist');
  listNoteComments.mockReset().mockResolvedValue([]);
});

const renderPage = (noteId = 'n2', planId = 'plan-1') =>
  render(<NotePage planId={planId} noteId={noteId} />);

/** The milestone's child note, by the start of its body. */
const N1 = /Read the memory model/;

/** Renders the page and waits for the note's body, n2's by default. */
const openNote = async (noteId = 'n2', ready: RegExp = /Launch ideas/) => {
  const view = renderPage(noteId);
  await screen.findByText(ready);
  return view;
};

/** Opens the note and clicks its body into writing mode. */
const startWriting = async (noteId = 'n2', ready: RegExp = /Launch ideas/) => {
  await openNote(noteId, ready);
  fireEvent.click(screen.getByText(ready));
  return editor() as HTMLTextAreaElement;
};

/** Serves n2 alone, with `summary` on its card. */
const withSummary = (summary?: string) =>
  fetchChecklist.mockResolvedValue([
    mk('n2', 'Launch ideas\nA waitlist', { type: 'note', summary }),
  ]);

/**
 * Clicks, then lets every promise the click started run out: a leave decides
 * over several awaits (flush, retry, the prompt), more than \`act\` waits for.
 */
const clickSettled = async (el: Element) => {
  fireEvent.click(el);
  await settle(0);
};

const sheet = () => document.querySelector('[data-note-sheet]') as HTMLElement;
const editor = () => screen.getByRole('textbox', { name: 'Note' });
const meta = () => document.querySelector('[data-note-meta]')!;
const follows = (a: Element, b: Element) =>
  Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);

/** Opens the summary editor on the card block and saves `text` with Enter. */
const writeSummary = async (text: string, noteId = 'n2', ready: RegExp = /Launch ideas/) => {
  await openNote(noteId, ready);
  fireEvent.click(screen.getByRole('button', { name: /Write a summary|Edit summary/ }));
  const box = screen.getByRole('textbox', { name: 'Summary' });
  fireEvent.change(box, { target: { value: text } });
  await act(async () => fireEvent.keyDown(box, { key: 'Enter' }));
};

/** With the menu open: Delete, then the confirmation's Delete. */
const deleteFromMenu = async () => {
  fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
  await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })));
};

const pressSummarize = () =>
  act(async () =>
    fireEvent.click(screen.getByRole('button', { name: 'Summarize with AI' }))
  );

describe('NotePage Back (R17)', () => {
  it.each([
    ["a milestone's child note, to the milestone's page, named for it", 'n1', 'Back to Learn Go', '/plan/plan-1/m/m1'],
    ['a top-level note, to the plan', 'n2', 'Back to plan', '/plan/plan-1'],
  ])('should go back for %s', async (_why, id, name, href) => {
    renderPage(id);
    const back = await screen.findByRole('link', { name });
    expect(back.getAttribute('href')).toBe(href);
  });
});

describe('NotePage layout and title (R19, R21, R49)', () => {
  it('should show a quiet loading line first', () => {
    fetchChecklist.mockReturnValue(new Promise(() => {}));
    renderPage();
    expect(screen.getByText('Loading note…')).toBeInTheDocument();
  });

  it('should read the plan’s long-term list, and lay out header, body sheet, meta line, then "On the card", in that order (R18, R28)', async () => {
    await openNote();
    expect(fetchChecklist).toHaveBeenCalledWith('plan-1', 'longterm');
    const header = document.querySelector('header')!;
    const card = screen.getByRole('region', { name: 'On the card' });
    expect(follows(header, sheet())).toBe(true);
    expect(follows(sheet(), meta())).toBe(true);
    expect(follows(meta(), card)).toBe(true);
  });

  it("should name the tab after the card text's first line, up to 60 characters", async () => {
    fetchChecklist.mockResolvedValue([
      mk('n9', `${'x'.repeat(80)}\nsecond`, { type: 'note' }),
    ]);
    await openNote('n9', /xxx/);
    // The title follows the paint by an effect: wait for it, not the text.
    await waitFor(() => expect(document.title).toHaveLength(60));
    expect(document.title.startsWith('xxx')).toBe(true);
  });
});

describe('NotePage menu (R20)', () => {
  const openMenu = async () => {
    await openNote('n1', N1);
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Read the memory model' }));
  };
  const makeTask = () =>
    act(async () => fireEvent.click(screen.getByRole('menuitem', { name: 'Make a task' })));

  it('should offer exactly Make a task and Delete', async () => {
    await openMenu();
    expect(screen.getAllByRole('menuitem').map((r) => r.textContent)).toEqual([
      'Make a task',
      'Delete',
    ]);
  });

  it('should make it a task, then go to the Back destination', async () => {
    await openMenu();
    await makeTask();
    expect(updateChecklistItem).toHaveBeenCalledWith('n1', { type: 'task' }, 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plan/plan-1/m/m1');
  });

  it('should say the note has comments when Make a task is refused for them (FS-65B3N R31)', async () => {
    quietErrors();
    updateChecklistItem.mockRejectedValue(
      new ApiError({ code: 'NOTE_HAS_COMMENTS', status: 400, detail: 'anything' } as never, 400)
    );
    await openMenu();
    await makeTask();
    expect(
      screen.getByText('This note has comments. Delete them to make it a task.')
    ).toBeInTheDocument();
    expect(screen.queryByText('Failed to change item type. Please try again.')).toBeNull();
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByText(N1)).toBeInTheDocument();
  });

  it('should stay with the error when Make a task is refused', async () => {
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await openMenu();
    await makeTask();
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
    await deleteFromMenu();
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
    await openNote();
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

  it.each([
    ['unchanged text', ['Launch ideas\nA waitlist!', 'Launch ideas\nA waitlist']],
    ['an empty or whitespace body', ['   \n  ']],
  ])('should never send %s, and show the stored body again', async (_why, edits) => {
    const box = await startWriting();
    vi.useFakeTimers();
    for (const value of edits) fireEvent.change(box, { target: { value } });
    await act(async () => vi.advanceTimersByTime(1000));
    await act(async () => fireEvent.blur(box));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(sheet().textContent).toBe('Launch ideas\nA waitlist');
  });

  it('should save a pending edit when the page goes away (best effort)', async () => {
    const view = await openNote();
    fireEvent.click(screen.getByText(/Launch ideas/));
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
    const call = deferred<unknown>();
    updateChecklistItem.mockReturnValue(call.promise);
    const box = await startWriting();
    expect(meta().textContent).toMatch(/^4 words · edited/);
    fireEvent.change(box, { target: { value: 'Launch ideas now' } });
    await act(async () => fireEvent.blur(box));
    expect(meta().textContent).toBe('3 words · Saving…');
    await act(async () => call.resolve({ ...ITEMS[2], description: 'Launch ideas now' }));
    expect(meta().textContent).toMatch(/^3 words · edited/);
  });

  it("should keep the text and offer Try again when a save fails", async () => {
    updateChecklistItem.mockRejectedValueOnce(new Error('network'));
    const box = await startWriting();
    fireEvent.change(box, { target: { value: 'Launch ideas, kept' } });
    await act(async () => fireEvent.blur(box));
    expect(meta().textContent).toContain("Couldn't save");
    expect(sheet().textContent).toBe('Launch ideas, kept');

    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Try again' })));
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(updateChecklistItem.mock.calls[1][1]).toEqual({ description: 'Launch ideas, kept' });
    expect(meta().textContent).toMatch(/edited/);
  });
});

describe('NotePage deleted while open (R37)', () => {
  it.each([
    [
      'a body save',
      async () => {
        updateChecklistItem.mockRejectedValue(new ApiError(undefined, 404));
        fireEvent.change(await startWriting('n1', N1), { target: { value: 'into nothing' } });
        await act(async () => fireEvent.blur(editor()));
      },
    ],
    [
      'a summary save',
      async () => {
        updateChecklistItem.mockRejectedValue(new ApiError(undefined, 404));
        await writeSummary('into nothing', 'n1', N1);
      },
    ],
    [
      'Summarize with AI',
      async () => {
        summarizeNote.mockRejectedValue(new ApiError({ code: 'NOT_FOUND', status: 404 } as never, 404));
        await openNote('n1', N1);
        await pressSummarize();
      },
    ],
  ])('should replace the page with "This note was deleted" and Back when %s answers 404', async (_how, run) => {
    await run();
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
    quietErrors();
    fetchChecklist.mockRejectedValueOnce(err);
    renderPage();
    expect(await screen.findByText("Couldn't load this note")).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Try again' })));
    expect(await screen.findByText(/Launch ideas/)).toBeInTheDocument();
    expect(fetchChecklist).toHaveBeenCalledTimes(2);
  });
});

describe('NotePage "On the card" block (R28, R29)', () => {
  it('should save the summary through the item update and show it at once', async () => {
    await writeSummary('  the gist ');
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'the gist' }, 'plan-1', 'longterm');
    expect(screen.getByText('the gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
    // The tab follows the card text.
    expect(document.title).toBe('the gist');
  });

  it('should clear the summary when saved empty, and say there is none again', async () => {
    withSummary('old gist');
    await writeSummary('');
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: '' }, 'plan-1', 'longterm');
    expect(screen.getByText('No summary yet. The card shows the start of the note.')).toBeInTheDocument();
  });

  it('should put the old summary back with the existing error when the save fails', async () => {
    quietErrors();
    withSummary('old gist');
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await writeSummary('new gist');
    expect(screen.getByText('old gist')).toBeInTheDocument();
    expect(screen.queryByText('new gist')).toBeNull();
    expect(screen.getByText('Failed to update task. Please try again.')).toBeInTheDocument();
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
  });

  it('should open reading otherwise', async () => {
    window.history.replaceState(null, '', '/plan/plan-1/n/n2');
    await openNote();
    expect(screen.queryByRole('textbox', { name: 'Note' })).toBeNull();
  });
});

describe('NotePage Summarize with AI (R30, R31, R44)', () => {
  it('should save the returned text at once when there is no summary, and show it', async () => {
    await openNote();
    await pressSummarize();
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    expect(summarizeNote).toHaveBeenCalledWith('plan-1', 'n2');
    expect(updateChecklistItem).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'An AI gist' }, 'plan-1', 'longterm');
    expect(screen.getByText('An AI gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
  });

  it('should offer the result as a proposal when a summary exists, which Save keeps', async () => {
    withSummary('old gist');
    await openNote();
    await pressSummarize();
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
    await openNote();
    await pressSummarize();
    act(() => cancel());
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(screen.getByText('old gist')).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: 'Summary' })).toBeNull();
  });

  it('should read "Summarizing…", disabled, while it runs, so a second press makes no second call', async () => {
    const call = deferred<string>();
    summarizeNote.mockReturnValue(call.promise);
    await openNote();
    fireEvent.click(screen.getByRole('button', { name: 'Summarize with AI' }));
    const busy = screen.getByRole('button', { name: 'Summarizing…' });
    expect(busy).toBeDisabled();
    fireEvent.click(busy);
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    await act(async () => call.resolve('An AI gist'));
    expect(screen.getByRole('button', { name: 'Summarize with AI' })).not.toBeDisabled();
  });

  it.each([
    ['NOTE_SUMMARY_FAILED', 503, false],
    ['SERVICE_UNAVAILABLE', 503, false],
    ['ITEM_NOT_A_NOTE', 400, false],
    ['LLM_BUDGET_EXHAUSTED', 429, true],
  ])(
    'should say "Couldn\'t summarize" on %s, keep the summary and the other options, adding the paused line on the budget code alone',
    async (code, status, paused) => {
      withSummary('old gist');
      summarizeNote.mockRejectedValue(
        new ApiError({ code, status, detail: 'whatever' } as never, status)
      );
      await openNote();
      await pressSummarize();
      expect(screen.getByText("Couldn't summarize")).toBeInTheDocument();
      expect(Boolean(screen.queryByText('AI summaries are paused until tomorrow'))).toBe(paused);
      expect(screen.getByText('old gist')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
      expect(updateChecklistItem).not.toHaveBeenCalled();
    }
  );

  it('should revert with the existing error when saving the returned text fails', async () => {
    quietErrors();
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await openNote();
    await pressSummarize();
    expect(screen.getByText('No summary yet. The card shows the start of the note.')).toBeInTheDocument();
    expect(screen.queryByText('An AI gist')).toBeNull();
    expect(screen.getByText('Failed to update task. Please try again.')).toBeInTheDocument();
  });

  it('should make no model call without a press: not on load, autosave or a summary edit', async () => {
    await startWriting();
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
  const clickLink = (name: string) => clickSettled(screen.getByRole('link', { name }));
  const clickBack = () => clickLink('Back to plan');
  /** A long body with no summary, opened, then a click on `link`: the prompt's way in. */
  const leaveLong = async (link = 'Back to plan') => {
    withBody(LONG);
    await open();
    await clickLink(link);
  };
  const PROMPT = { name: 'Add a summary for the card?' };
  const prompt = () => screen.queryByRole('dialog', PROMPT);
  /** The prompt, once it has opened: waits on the page, not on a tick. */
  const opened = () => screen.findByRole('dialog', PROMPT);
  const choose = async (name: string) =>
    clickSettled(within(await opened()).getByRole('button', { name }));
  const choices = (dialog: HTMLElement) =>
    Array.from(dialog.querySelectorAll('button')).map((b) => b.textContent);

  it('should open on Back for a body over 280 characters with no summary, as a labelled modal that takes focus, holding the navigation', async () => {
    await leaveLong();
    const dialog = await opened();
    expect(push).not.toHaveBeenCalled();
    expect(choices(dialog)).toEqual(['Summarize with AI', 'Write a summary', 'Leave anyway']);
    expect(dialog.getAttribute('aria-modal')).toBe('true');
    expect(dialog.contains(document.activeElement)).toBe(true);
  });

  it.each([
    ['Back', 'Back to plan', '/plan/plan-1'],
    ['any other in-app link, such as the sidebar', 'Plans', '/plans'],
  ])('should open on %s and leave for it on Leave anyway', async (_how, link, href) => {
    await leaveLong(link);
    expect(await opened()).toBeInTheDocument();
    await choose('Leave anyway');
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith(href);
    expect(summarizeNote).not.toHaveBeenCalled();
  });

  it.each([
    ['a body of exactly 280 characters', 'x'.repeat(280), undefined],
    ['a body of 280 once trimmed', `  ${'x'.repeat(280)}  \n`, undefined],
    ['a note with a summary', LONG, 'the gist'],
  ])('should not open for %s: it just leaves', async (_why, body, summary) => {
    withBody(body, summary);
    await open();
    await clickBack();
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should open at most once per visit, whatever was chosen', async () => {
    await leaveLong();
    await act(async () => fireEvent.keyDown(await opened(), { key: 'Escape' }));
    expect(prompt()).toBeNull();
    await clickBack();
    expect(prompt()).toBeNull();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it.each([
    ['Esc', () => fireEvent.keyDown(prompt()!, { key: 'Escape' })],
    ['a click outside', () => fireEvent.click(document.querySelector('[data-leave-scrim]')!)],
  ])('should stay on %s, dropping the navigation', async (_how, dismiss) => {
    await leaveLong();
    await opened();
    act(() => dismiss());
    expect(prompt()).toBeNull();
    expect(push).not.toHaveBeenCalled();
  });

  it('should stay on Write a summary, with the summary editor open and focused', async () => {
    await leaveLong();
    await choose('Write a summary');
    expect(prompt()).toBeNull();
    expect(push).not.toHaveBeenCalled();
    const box = screen.getByRole('textbox', { name: 'Summary' });
    expect(document.activeElement).toBe(box);
  });

  it('should summarize, save, then go where the person was going (G12)', async () => {
    await leaveLong('Plans');
    await choose('Summarize with AI');
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    expect(updateChecklistItem).toHaveBeenCalledWith('n2', { summary: 'An AI gist' }, 'plan-1', 'longterm');
    expect(push).toHaveBeenCalledWith('/plans');
    expect(prompt()).toBeNull();
  });

  it('should read "Summarizing…" while it runs, and take no second press', async () => {
    const call = deferred<string>();
    summarizeNote.mockReturnValue(call.promise);
    await leaveLong();
    fireEvent.click(within(await opened()).getByRole('button', { name: 'Summarize with AI' }));
    const busy = within(prompt()!).getByRole('button', { name: 'Summarizing…' });
    expect(busy).toBeDisabled();
    fireEvent.click(busy);
    expect(summarizeNote).toHaveBeenCalledTimes(1);
    await act(async () => call.resolve('An AI gist'));
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it.each([
    ['the AI fails', 'NOTE_SUMMARY_FAILED', 503, false],
    ['the budget is spent, adding the paused line', 'LLM_BUDGET_EXHAUSTED', 429, true],
  ])(
    'should stay open with "Couldn\'t summarize" and only the other two choices when %s',
    async (_why, code, status, paused) => {
      summarizeNote.mockRejectedValue(new ApiError({ code, status } as never, status));
      await leaveLong();
      await choose('Summarize with AI');
      const dialog = await opened();
      expect(await within(dialog).findByText("Couldn't summarize")).toBeInTheDocument();
      expect(
        Boolean(within(dialog).queryByText('AI summaries are paused until tomorrow'))
      ).toBe(paused);
      expect(choices(dialog)).toEqual(['Write a summary', 'Leave anyway']);
      expect(push).not.toHaveBeenCalled();
    }
  );

  it('should say "Couldn\'t save the summary" when saving it fails, with the same two choices', async () => {
    quietErrors();
    updateChecklistItem.mockRejectedValue(new ApiError(undefined, 400));
    await leaveLong();
    await choose('Summarize with AI');
    const dialog = await opened();
    expect(await within(dialog).findByText("Couldn't save the summary")).toBeInTheDocument();
    expect(choices(dialog)).toEqual(['Write a summary', 'Leave anyway']);
    await choose('Leave anyway');
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });
});

describe('NotePage flush before leaving (R24, R27)', () => {
  const typeThenBack = async (text: string) => {
    await startWriting();
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
    quietErrors();
    updateChecklistItem.mockRejectedValueOnce(new Error('network'));
    await typeThenBack('Launch ideas, second time lucky');
    expect(updateChecklistItem).toHaveBeenCalledTimes(2);
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });

  it('should stop with "Couldn\'t save your changes" and Leave anyway when both fail', async () => {
    quietErrors();
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
    await openNote();
    await clickSettled(screen.getByRole('link', { name: 'Back to plan' }));
    expect(updateChecklistItem).not.toHaveBeenCalled();
    expect(push).toHaveBeenCalledWith('/plan/plan-1');
  });
});

describe('NotePage comments slot (FS-PGG2N R19, FS-65B3N R24, R30, R33)', () => {
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
    await startWriting();
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
    const view = await openNote();
    fireEvent.click(screen.getByText(/Launch ideas/));
    vi.useFakeTimers();
    fireEvent.change(editor(), { target: { value: text } });
    fireEvent.click(screen.getByRole('button', { name: 'Actions for Launch ideas' }));
    return view;
  };
  const bodySaves = () =>
    updateChecklistItem.mock.calls.filter(([, fields]) => 'description' in fields);

  it('should drop the waiting save on Delete, so no write follows the delete', async () => {
    const view = await typeThenOpenMenu('Launch ideas, then deleted');
    await deleteFromMenu();
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
    quietErrors();
    deleteChecklistItem.mockRejectedValue(new Error('nope'));
    await typeThenOpenMenu('Launch ideas, kept');
    await deleteFromMenu();
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
    quietErrors();
    updateChecklistItem.mockImplementation(async (id: string, fields: Partial<ChecklistItem>) => {
      if ('description' in fields) throw new Error('network');
      return echoUpdate(id, fields);
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
