import { describe, it, expect, vi } from 'vitest';
import { act, fireEvent, render, renderHook, screen } from '@testing-library/react';
import CompletionNudge, { useCompletionNudges } from './CompletionNudge';
import type { Milestone } from '@/lib/milestones';

/**
 * The completion nudge (FS-FSFEC R10–R14, FS-043WP R21): one offer, drawn the
 * same on a grid card and on a milestone's page, with one memory of being
 * waved off. Pinned here: the offer itself, and how long a dismissal lasts.
 */

const ms = (
  id: string,
  done: number,
  total: number,
  closed = false
): Milestone => ({
  id,
  title: id,
  closed,
  steps: { done, total },
  progress: total ? done / total : 0,
});

describe('CompletionNudge', () => {
  it('should offer to close, through the caller, and to be waved off', () => {
    const onClose = vi.fn();
    const onDismiss = vi.fn();
    render(
      <CompletionNudge title="Ship v1" onClose={onClose} onDismiss={onDismiss} />
    );

    expect(screen.getByText('All steps done. Close this milestone?')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Close Ship v1' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onDismiss).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Not now, Ship v1' }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});

describe('useCompletionNudges', () => {
  const hook = (initial: Milestone[]) =>
    renderHook(({ list }) => useCompletionNudges(list), {
      initialProps: { list: initial },
    });

  it('should nudge only a complete milestone: all task steps done, still open', () => {
    const { result } = hook([]);
    expect(result.current.shows(ms('m', 2, 2))).toBe(true);
    expect(result.current.shows(ms('m', 1, 2))).toBe(false);
    expect(result.current.shows(ms('m', 0, 0))).toBe(false);
    expect(result.current.shows(ms('m', 2, 2, true))).toBe(false);
    expect(result.current.shows(undefined)).toBe(false);
  });

  it('should hold a dismissal while the progress pair holds, and drop it once it moves', () => {
    const { result, rerender } = hook([ms('m', 2, 2)]);
    act(() => result.current.dismiss(ms('m', 2, 2)));
    expect(result.current.shows(ms('m', 2, 2))).toBe(false);

    // Closed and reopened at the same pair: still waved off.
    rerender({ list: [ms('m', 2, 2, true)] });
    rerender({ list: [ms('m', 2, 2)] });
    expect(result.current.shows(ms('m', 2, 2))).toBe(false);

    // Untick, re-tick: the pair moved, so the offer comes back.
    rerender({ list: [ms('m', 1, 2)] });
    rerender({ list: [ms('m', 2, 2)] });
    expect(result.current.shows(ms('m', 2, 2))).toBe(true);
  });

  it("should keep the dismissal of a milestone it isn't shown, which nothing here can tick", () => {
    const { result, rerender } = hook([ms('m', 2, 2)]);
    act(() => result.current.dismiss(ms('m', 2, 2)));
    rerender({ list: [] });
    rerender({ list: [ms('m', 2, 2)] });
    expect(result.current.shows(ms('m', 2, 2))).toBe(false);
  });

  it('should forget every dismissal on remount: memory only', () => {
    const first = hook([ms('m', 2, 2)]);
    act(() => first.result.current.dismiss(ms('m', 2, 2)));
    first.unmount();
    expect(hook([ms('m', 2, 2)]).result.current.shows(ms('m', 2, 2))).toBe(true);
  });
});
