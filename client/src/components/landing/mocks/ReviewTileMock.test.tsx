import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/react';
import { expectMockContract } from '@/test/mockContract';
import { REVEAL_MS } from '../bento/BentoTile';
import { DAY_PLAN, PERSONA_NAME, REVIEW_BUBBLE_MAX_CHARS, REVIEW_THREAD } from './content';
import ReviewTileMock from './ReviewTileMock';

function bubbles(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-review-message]'));
}

/** Present at start: not inside a part that only arrives on reveal. */
function atStart(el: Element) {
  return !el.closest('[data-reveal-part="enter"]');
}

/** A part's stagger, in ms (0 when it has none). */
function delayOf(el: Element) {
  return parseInt((el as HTMLElement).style.getPropertyValue('--reveal-delay') || '0', 10);
}

describe('ReviewTileMock', () => {
  it('should honour the mock contract', () => {
    const { container } = render(<ReviewTileMock />);

    expectMockContract(container);
  });

  it('should render with no app context providers at all', () => {
    expect(() => render(<ReviewTileMock />)).not.toThrow();
  });

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

  it('should keep every bubble to two lines at most, down to a 320px phone', () => {
    const { container } = render(<ReviewTileMock />);
    const thread = [...REVIEW_THREAD.start, REVIEW_THREAD.next];

    // The cap is sized for the narrowest REVIEW bubble: the single-column tile
    // at 320px, below `sm`. jsdom has no layout, so pin the geometry the cap
    // assumes there (14px text, 12px side padding, the tile's full width) and
    // step back up to the roomier bubble from `sm`.
    thread.forEach((m) => expect(m.text.length).toBeLessThanOrEqual(REVIEW_BUBBLE_MAX_CHARS));
    bubbles(container).forEach((b) => {
      const classes = b.className.split(/\s+/);
      expect(classes).toEqual(
        expect.arrayContaining(['text-xs', 'px-3', 'max-w-full', 'sm:text-sm', 'sm:px-3.5', 'sm:max-w-[90%]'])
      );
      expect(b.querySelector('.line-clamp-2')).toHaveTextContent(/.+/);
    });
  });

  it('should finish its longest stagger inside the tile reveal window', () => {
    const css = readFileSync(resolve(__dirname, '../bento/bento.module.css'), 'utf8');
    const revealMs = parseInt(/--reveal-ms:\s*(\d+)ms/.exec(css)![1], 10);
    const { container } = render(<ReviewTileMock />);

    const longest = Math.max(
      0,
      ...Array.from(container.querySelectorAll('[data-reveal-part]'), delayOf)
    );
    expect(longest + revealMs).toBeLessThanOrEqual(REVEAL_MS);
  });

  it('should import no product component, context, service layer or fetch', () => {
    const source = readFileSync(resolve(__dirname, 'ReviewTileMock.tsx'), 'utf8');

    expect(source).not.toMatch(/from '@\/components\/(?!landing)/);
    expect(source).not.toMatch(/@\/(context|api|services|lib\/api)/);
    expect(source).not.toMatch(/\bfetch\(|useContext/);
  });
});
