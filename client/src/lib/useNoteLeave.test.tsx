import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import type { SummarizeOutcome } from './useSummarizeNote';
import { useNoteLeave, type BodySaveResult, type LeavingBody } from './useNoteLeave';

/**
 * Leaving a note's page (FS-PGG2N R27, R32–R35), decided without the page: the
 * body is flushed first, and a long note with no summary asks for one, once
 * per visit.
 */

const LONG = 'x'.repeat(281);

let flush: ReturnType<typeof vi.fn<() => Promise<BodySaveResult>>>;
let retry: ReturnType<typeof vi.fn<() => Promise<BodySaveResult>>>;
let text: string;
const summarize = vi.fn<() => Promise<SummarizeOutcome>>();
const openSummary = vi.fn();
const go = vi.fn();

function setup(summary: string | null = null) {
  const body: LeavingBody = { flush: () => flush(), retry: () => retry(), text: () => text };
  return renderHook(
    ({ summary }) =>
      useNoteLeave({
        body: { current: body },
        description: text,
        summary,
        summarize,
        openSummary,
        go,
      }),
    { initialProps: { summary } }
  );
}

const tick = () => new Promise<void>((r) => setTimeout(r, 0));

/**
 * Starts a leave and lets it run up to the prompt (or its end). The answer
 * comes back boxed: an async function returning it bare would wait on it.
 */
async function start(result: { current: ReturnType<typeof useNoteLeave> }, href = '/plan/p1') {
  let answer!: Promise<boolean>;
  await act(async () => {
    answer = result.current.decide(href);
    await tick();
  });
  return { answer };
}
/** A leave run to its end. */
const decide = async (result: { current: ReturnType<typeof useNoteLeave> }, href?: string) =>
  (await start(result, href)).answer;

const settled = async <T,>(p: Promise<T>) => {
  let done = false;
  let value: T | undefined;
  p.then((v) => {
    done = true;
    value = v;
  });
  await act(tick);
  return done ? value : 'pending';
};

beforeEach(() => {
  vi.clearAllMocks();
  flush = vi.fn(async () => 'saved' as const);
  retry = vi.fn(async () => 'saved' as const);
  text = 'a short note';
});

describe('useNoteLeave flush first (R27)', () => {
  it('should flush the body, then let a short note go', async () => {
    const { result } = setup();
    expect(await decide(result)).toBe(true);
    expect(flush).toHaveBeenCalledTimes(1);
    expect(retry).not.toHaveBeenCalled();
  });

  it('should try once more when the flush fails, and go when that lands', async () => {
    flush.mockResolvedValue('failed');
    const { result } = setup();
    expect(await decide(result)).toBe(true);
    expect(retry).toHaveBeenCalledTimes(1);
    expect(result.current.unsaved).toBeNull();
  });

  it('should stop when both fail, holding where it was going for Leave anyway', async () => {
    flush.mockResolvedValue('failed');
    retry.mockResolvedValue('failed');
    const { result } = setup();
    expect(await decide(result, '/plans')).toBe(false);
    expect(result.current.unsaved).toBe('/plans');
    act(() => result.current.leaveUnsaved());
    expect(go).toHaveBeenCalledWith('/plans');
    expect(result.current.unsaved).toBeNull();
  });

  it('should let a gone note go without asking, however long', async () => {
    flush.mockResolvedValue('gone');
    text = LONG;
    const { result } = setup();
    expect(await decide(result)).toBe(true);
    expect(result.current.prompt).toBeNull();
  });
});

describe('useNoteLeave when to ask (R32, R35)', () => {
  it('should not ask for a body of exactly 280 characters, trimmed', async () => {
    text = `  ${'x'.repeat(280)}  `;
    const { result } = setup();
    expect(await decide(result)).toBe(true);
    expect(result.current.prompt).toBeNull();
  });

  it('should ask for a body over 280 with no summary, holding the leave', async () => {
    text = LONG;
    const { result } = setup();
    const { answer } = await start(result);
    expect(result.current.prompt).toEqual({ problem: null });
    expect(await settled(answer)).toBe('pending');
  });

  it('should never ask when the note has a summary, read at the leave', async () => {
    text = LONG;
    const { result, rerender } = setup(null);
    rerender({ summary: 'The gist' });
    expect(await decide(result)).toBe(true);
    expect(result.current.prompt).toBeNull();
  });

  it('should ask at most once per visit, whatever was chosen', async () => {
    text = LONG;
    const { result } = setup();
    const { answer: first } = await start(result);
    act(() => result.current.choose.stay());
    expect(await first).toBe(false);
    expect(result.current.prompt).toBeNull();
    expect(await decide(result)).toBe(true);
    expect(result.current.prompt).toBeNull();
  });
});

describe('useNoteLeave the prompt’s choices (R33, R34)', () => {
  const asked = async () => {
    text = LONG;
    const hook = setup();
    const { answer } = await start(hook.result);
    return { ...hook, answer };
  };

  it('should go on Leave anyway', async () => {
    const { result, answer } = await asked();
    act(() => result.current.choose.leave());
    expect(await answer).toBe(true);
    expect(result.current.prompt).toBeNull();
  });

  it('should stay on Write a summary, opening the editor', async () => {
    const { result, answer } = await asked();
    act(() => result.current.choose.write());
    expect(await answer).toBe(false);
    expect(openSummary).toHaveBeenCalledWith();
  });

  it('should summarize, then go when the summary is saved', async () => {
    summarize.mockResolvedValue({ kind: 'saved' });
    const { result, answer } = await asked();
    await act(async () => result.current.choose.summarize());
    expect(await answer).toBe(true);
  });

  it('should go when the note turns out to be gone', async () => {
    summarize.mockResolvedValue({ kind: 'gone' });
    const { result, answer } = await asked();
    await act(async () => result.current.choose.summarize());
    expect(await answer).toBe(true);
  });

  it.each([
    ['the AI fails', { kind: 'failed', reason: 'failed' }, 'failed'],
    ['the budget is spent', { kind: 'failed', reason: 'paused' }, 'paused'],
    ['saving the summary fails', { kind: 'saveFailed' }, 'unsaved'],
  ] as const)('should keep asking, with the problem, when %s', async (_why, outcome, problem) => {
    summarize.mockResolvedValue(outcome as SummarizeOutcome);
    const { result, answer } = await asked();
    await act(async () => result.current.choose.summarize());
    expect(result.current.prompt).toEqual({ problem });
    expect(await settled(answer)).toBe('pending');
  });

  it('should stay and open the proposal when a summary arrived meanwhile', async () => {
    summarize.mockResolvedValue({ kind: 'proposed', text: 'An AI gist' });
    const { result, answer } = await asked();
    await act(async () => result.current.choose.summarize());
    expect(await answer).toBe(false);
    expect(openSummary).toHaveBeenCalledWith('An AI gist');
  });
});

describe('useNoteLeave the page’s own leave (R20, R27)', () => {
  it('should flush, then go, never asking', async () => {
    text = LONG;
    const { result } = setup();
    await act(async () => result.current.leave('/plan/p1'));
    expect(flush).toHaveBeenCalledTimes(1);
    expect(go).toHaveBeenCalledWith('/plan/p1');
    expect(result.current.prompt).toBeNull();
  });

  it('should stop when the body will not save', async () => {
    flush.mockResolvedValue('failed');
    retry.mockResolvedValue('failed');
    const { result } = setup();
    await act(async () => result.current.leave('/plan/p1'));
    expect(go).not.toHaveBeenCalled();
    expect(result.current.unsaved).toBe('/plan/p1');
  });
});
