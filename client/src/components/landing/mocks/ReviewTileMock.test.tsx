import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { DAY_PLAN, PERSONA_NAME, REVIEW_BUBBLE_MAX_CHARS, REVIEW_THREAD } from './content';
import ReviewTileMock from './ReviewTileMock';

function bubbles(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-review-message]'));
}

/** Present at start: not inside a part that only arrives on reveal. */
function atStart(el: Element) {
  return !el.closest('[data-reveal-part="enter"]');
}

describe('ReviewTileMock', () => {
  it("should open on Chris's note and one reply, both in place at start", () => {
    const { container } = render(<ReviewTileMock />);
    const [note, reply] = bubbles(container);

    expect(REVIEW_THREAD.start.map((m) => m.from)).toEqual(['persona', 'fireplace']);
    expect(note).toHaveTextContent(REVIEW_THREAD.start[0].text);
    expect(note).toHaveAttribute('data-review-message', 'persona');
    expect(reply).toHaveTextContent(REVIEW_THREAD.start[1].text);
    expect(reply).toHaveAttribute('data-review-message', 'fireplace');
    expect(atStart(note)).toBe(true);
    expect(atStart(reply)).toBe(true);
    expect(container).toHaveTextContent(PERSONA_NAME);
  });

  it('should end with the next reply, in the DOM at start but held hidden in its slot', () => {
    const { container } = render(<ReviewTileMock />);
    const all = bubbles(container);
    const next = all[all.length - 1];

    expect(all).toHaveLength(REVIEW_THREAD.start.length + 1);
    expect(next).toHaveTextContent(REVIEW_THREAD.next.text);
    // It arrives by the reveal's enter part (translateY + opacity), never by
    // mounting, so its slot is reserved and the tile's box never changes.
    const part = next.closest<HTMLElement>('[data-reveal-part="enter"]');
    expect(part).not.toBeNull();
    for (let el: HTMLElement | null = next; el && el !== container; el = el.parentElement) {
      expect(el.className).not.toMatch(/(^|\s)(hidden|absolute|fixed|invisible|h-0|max-h-0)(\s|$)/);
    }
  });

  it('should talk about the same Tuesday the TODAY tile plans', () => {
    const entries = DAY_PLAN.flatMap((g) => g.entries);
    const finished = entries.find((e) => e.ticked)!;
    const open = entries.filter((e) => !e.ticked).map((e) => e.step);
    const note = REVIEW_THREAD.start[0].text;

    expect(note).toContain(finished.step);
    expect(open.some((step) => note.includes(step))).toBe(true);
  });

  it('should keep every message inside the two-line bubble cap', () => {
    // The cap is sized for the narrowest bubble, the single-column tile at a
    // 320px phone; the wrap itself is checked in a browser.
    [...REVIEW_THREAD.start, REVIEW_THREAD.next].forEach((m) =>
      expect(m.text.length).toBeLessThanOrEqual(REVIEW_BUBBLE_MAX_CHARS)
    );
  });
});
