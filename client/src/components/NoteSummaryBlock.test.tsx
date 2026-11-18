import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createRef } from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import NoteSummaryBlock, { type NoteSummaryHandle } from './NoteSummaryBlock';

/**
 * The "On the card" block (FS-PGG2N R28, R29, G10): what the card shows, and
 * a hand-written summary in one short paragraph.
 */

const onSave = vi.fn();
const renderBlock = (summary?: string | null) => {
  const ref = createRef<NoteSummaryHandle>();
  const view = render(<NoteSummaryBlock ref={ref} summary={summary} onSave={onSave} />);
  return { ...view, ref };
};
const editor = () => screen.getByRole('textbox', { name: 'Summary' }) as HTMLTextAreaElement;

beforeEach(() => onSave.mockReset());

describe('NoteSummaryBlock showing (R28)', () => {
  it('should say there is no summary yet, and offer Write a summary', () => {
    renderBlock(null);
    expect(screen.getByRole('region', { name: 'On the card' })).toBeInTheDocument();
    expect(
      screen.getByText('No summary yet. The card shows the start of the note.')
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Write a summary' })).toBeInTheDocument();
  });

  it('should show the summary, and offer Edit summary', () => {
    renderBlock('The gist');
    expect(screen.getByText('The gist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Edit summary' })).toBeInTheDocument();
    expect(screen.queryByText(/No summary yet/)).toBeNull();
  });
});

describe('NoteSummaryBlock editing (R29, G10)', () => {
  it('should save the trimmed summary once on Enter', () => {
    renderBlock(null);
    fireEvent.click(screen.getByRole('button', { name: 'Write a summary' }));
    fireEvent.change(editor(), { target: { value: '  gist ' } });
    fireEvent.keyDown(editor(), { key: 'Enter' });
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(onSave).toHaveBeenCalledWith('gist');
    expect(screen.queryByRole('textbox', { name: 'Summary' })).toBeNull();
  });

  it('should open on the current summary, with focus', () => {
    renderBlock('The gist');
    fireEvent.click(screen.getByRole('button', { name: 'Edit summary' }));
    expect(editor().value).toBe('The gist');
    expect(document.activeElement).toBe(editor());
  });

  it('should save on blur', () => {
    renderBlock('The gist');
    fireEvent.click(screen.getByRole('button', { name: 'Edit summary' }));
    fireEvent.change(editor(), { target: { value: 'A better gist' } });
    fireEvent.blur(editor());
    expect(onSave).toHaveBeenCalledWith('A better gist');
  });

  it('should cancel on Esc, saving nothing, even as the editor blurs away', () => {
    renderBlock('The gist');
    fireEvent.click(screen.getByRole('button', { name: 'Edit summary' }));
    const box = editor();
    fireEvent.change(box, { target: { value: 'Something else' } });
    fireEvent.keyDown(box, { key: 'Escape' });
    fireEvent.blur(box);
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText('The gist')).toBeInTheDocument();
  });

  it('should add no line on Shift+Enter: a summary is one paragraph', () => {
    renderBlock(null);
    fireEvent.click(screen.getByRole('button', { name: 'Write a summary' }));
    fireEvent.change(editor(), { target: { value: 'one line' } });
    const shiftEnter = fireEvent.keyDown(editor(), { key: 'Enter', shiftKey: true });
    expect(shiftEnter).toBe(false); // default prevented: no line break
    expect(onSave).not.toHaveBeenCalled();
    expect(editor().value).toBe('one line');
  });

  it('should stop input at 280 and count it as n/280', () => {
    renderBlock(null);
    fireEvent.click(screen.getByRole('button', { name: 'Write a summary' }));
    expect(screen.getByText('0/280')).toBeInTheDocument();
    fireEvent.change(editor(), { target: { value: 'gist' } });
    expect(screen.getByText('4/280')).toBeInTheDocument();
    fireEvent.change(editor(), { target: { value: 'x'.repeat(300) } });
    expect(editor().value).toBe('x'.repeat(280));
    expect(screen.getByText('280/280')).toBeInTheDocument();
  });

  // The server counts characters (runes), not UTF-16 units: an emoji is one.
  it('should count and stop by character, so 280 emoji fit and none is split', () => {
    renderBlock(null);
    fireEvent.click(screen.getByRole('button', { name: 'Write a summary' }));
    fireEvent.change(editor(), { target: { value: '🔥'.repeat(140) } });
    expect(screen.getByText('140/280')).toBeInTheDocument();
    fireEvent.change(editor(), { target: { value: '🔥'.repeat(281) } });
    expect(editor().value).toBe('🔥'.repeat(280));
    expect(screen.getByText('280/280')).toBeInTheDocument();
  });

  it('should cut a too-long proposal at 280 characters without splitting one', () => {
    const { ref } = renderBlock(null);
    act(() => ref.current!.open('🔥'.repeat(300)));
    expect(editor().value).toBe('🔥'.repeat(280));
    expect(screen.getByText('280/280')).toBeInTheDocument();
  });

  it('should clear the summary when saved empty', () => {
    renderBlock('The gist');
    fireEvent.click(screen.getByRole('button', { name: 'Edit summary' }));
    fireEvent.change(editor(), { target: { value: '   ' } });
    fireEvent.keyDown(editor(), { key: 'Enter' });
    expect(onSave).toHaveBeenCalledWith('');
  });

  it.each([
    ['the same summary', 'The gist', '  The gist  '],
    ['still no summary', null, '   '],
  ])('should send nothing for %s', (_why, summary, typed) => {
    renderBlock(summary);
    fireEvent.click(screen.getByRole('button', { name: summary ? 'Edit summary' : 'Write a summary' }));
    fireEvent.change(editor(), { target: { value: typed } });
    fireEvent.keyDown(editor(), { key: 'Enter' });
    expect(onSave).not.toHaveBeenCalled();
  });
});

describe('NoteSummaryBlock opener (for slices 7 and 8)', () => {
  it('should open the editor with focus from outside, prefilled when asked', () => {
    const { ref } = renderBlock('The gist');
    act(() => ref.current!.open('A proposal'));
    expect(editor().value).toBe('A proposal');
    expect(document.activeElement).toBe(editor());
  });

  it('should open on the current summary when not prefilled', () => {
    const { ref } = renderBlock(null);
    act(() => ref.current!.open());
    expect(editor().value).toBe('');
  });
});

describe('NoteSummaryBlock an AI proposal (R30, G11, ADR-0007)', () => {
  it('should keep a proposal only on Save, not on blur', () => {
    const { ref } = renderBlock('The gist');
    act(() => ref.current!.open('A proposal'));
    fireEvent.blur(editor());
    expect(onSave).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSave).toHaveBeenCalledWith('A proposal');
  });

  it('should keep a proposal on Enter too, a deliberate keep', () => {
    const { ref } = renderBlock('The gist');
    act(() => ref.current!.open('A proposal'));
    fireEvent.keyDown(editor(), { key: 'Enter' });
    expect(onSave).toHaveBeenCalledWith('A proposal');
  });

  it('should drop a proposal on Cancel and show the old summary', () => {
    const { ref } = renderBlock('The gist');
    act(() => ref.current!.open('A proposal'));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText('The gist')).toBeInTheDocument();
  });

  it('should offer no Save or Cancel when editing by hand', () => {
    renderBlock('The gist');
    fireEvent.click(screen.getByRole('button', { name: 'Edit summary' }));
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull();
  });
});

describe('NoteSummaryBlock Summarize with AI (R30, R31)', () => {
  it('should call onSummarize on a press', () => {
    const onSummarize = vi.fn();
    render(<NoteSummaryBlock summary={null} onSave={onSave} onSummarize={onSummarize} />);
    fireEvent.click(screen.getByRole('button', { name: 'Summarize with AI' }));
    expect(onSummarize).toHaveBeenCalledTimes(1);
  });

  it('should read "Summarizing…" and be disabled while running', () => {
    render(<NoteSummaryBlock summary={null} onSave={onSave} onSummarize={vi.fn()} summarizing />);
    expect(screen.getByRole('button', { name: 'Summarizing…' })).toBeDisabled();
  });

  it.each([
    ['failed', false],
    ['paused', true],
  ] as const)('should say it could not summarize (%s), with the paused line only when paused', (why, paused) => {
    render(<NoteSummaryBlock summary={null} onSave={onSave} onSummarize={vi.fn()} summarizeFailure={why} />);
    expect(screen.getByText("Couldn't summarize")).toBeInTheDocument();
    expect(Boolean(screen.queryByText('AI summaries are paused until tomorrow'))).toBe(paused);
    expect(screen.getByRole('button', { name: 'Write a summary' })).toBeInTheDocument();
  });
});
