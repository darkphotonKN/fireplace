import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/react';
import { expectMockContract } from '@/test/mockContract';
import { REVEAL_MS } from '../bento/BentoTile';
import { DATABASE_MILESTONE, MICROSERVICES_MILESTONE, PLAN_MILESTONES, PLAN_TITLE, progressOf } from './content';
import MilestonesTileMock from './MilestonesTileMock';

function segments(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-milestone-segment]'));
}

function fillOf(segment: HTMLElement) {
  const fill = segment.querySelector<HTMLElement>('[data-reveal-part="fill"]');
  if (!fill) throw new Error(`no fill in ${segment.dataset.milestoneSegment}`);
  return {
    from: Number(fill.style.getPropertyValue('--reveal-from')),
    to: Number(fill.style.getPropertyValue('--reveal-to')),
    delay: delayOf(fill),
  };
}

/** A part's stagger, in ms (0 when it has none). */
function delayOf(el: Element) {
  return parseInt((el as HTMLElement).style.getPropertyValue('--reveal-delay') || '0', 10);
}

/** Present at start: not inside a part that only arrives on reveal. */
function atStart(el: Element) {
  return !el.closest('[data-reveal-part="enter"]');
}

const closed = PLAN_MILESTONES.filter((m) => m.total > 0 && m.done === m.total).length;

describe('MilestonesTileMock', () => {
  it('should honour the mock contract', () => {
    const { container } = render(<MilestonesTileMock />);

    expectMockContract(container);
  });

  it('should draw one segment per milestone, fourteen in all, under the plan title', () => {
    const { container, getByText } = render(<MilestonesTileMock />);

    expect(PLAN_MILESTONES).toHaveLength(14);
    expect(segments(container).map((s) => s.dataset.milestoneSegment)).toEqual(
      PLAN_MILESTONES.map((m) => m.title)
    );
    expect(atStart(getByText(PLAN_TITLE))).toBe(true);
  });

  it('should headline the closed milestones at start, counted from content.ts', () => {
    const { getByText } = render(<MilestonesTileMock />);

    expect(closed).toBeGreaterThan(0);
    expect(closed).toBeLessThan(14);
    expect(atStart(getByText(`${closed} of 14 milestones`))).toBe(true);
  });

  it('should fill Database and Microservices to the counts the PLAN tile shows', () => {
    const { container } = render(<MilestonesTileMock />);
    const byTitle = (title: string) =>
      segments(container).find((s) => s.dataset.milestoneSegment === title)!;

    for (const milestone of [DATABASE_MILESTONE, MICROSERVICES_MILESTONE]) {
      const { done, total } = progressOf(milestone);
      expect(fillOf(byTitle(milestone.title)).from).toBe(done / total);
    }
    expect(fillOf(byTitle('Database')).from).toBe(1 / 5);
    expect(fillOf(byTitle('Microservices')).from).toBe(6 / 8);
  });

  it('should start every segment at its own milestone progress', () => {
    const { container } = render(<MilestonesTileMock />);

    segments(container).forEach((segment, i) => {
      const { done, total } = PLAN_MILESTONES[i];
      expect(fillOf(segment).from).toBe(done / total);
    });
  });

  it('should fill every open segment one by one, left to right, on reveal', () => {
    const { container } = render(<MilestonesTileMock />);
    const fills = segments(container).map(fillOf);

    fills.forEach((fill) => expect(fill.to).toBe(1));
    const moving = fills.filter((fill) => fill.from < 1).map((fill) => fill.delay);
    expect(moving).toHaveLength(14 - closed);
    moving.slice(1).forEach((d, i) => expect(d).toBeGreaterThan(moving[i]));
  });

  it('should cross-fade the headline to 14 of 14 as the last segment fills', () => {
    const { container, getByText } = render(<MilestonesTileMock />);
    const next = getByText('14 of 14 milestones');
    const last = Math.max(...segments(container).map((s) => fillOf(s).delay));

    expect(atStart(next)).toBe(false);
    expect(delayOf(next.closest('[data-reveal-part="enter"]')!)).toBe(last);
  });

  it('should finish its longest stagger inside the tile reveal window', () => {
    const css = readFileSync(resolve(__dirname, '../bento/bento.module.css'), 'utf8');
    const revealMs = parseInt(/--reveal-ms:\s*(\d+)ms/.exec(css)![1], 10);
    const { container } = render(<MilestonesTileMock />);

    const longest = Math.max(...Array.from(container.querySelectorAll('[data-reveal-part]'), delayOf));
    expect(longest + revealMs).toBeLessThanOrEqual(REVEAL_MS);
  });

  it('should fill by scaleX, never by width, and import no product component, context or fetch', () => {
    const source = readFileSync(resolve(__dirname, 'MilestonesTileMock.tsx'), 'utf8');

    expect(source).not.toMatch(/\bwidth\s*:/);
    expect(source).not.toMatch(/from '@\/components\/(?!landing)/);
    expect(source).not.toMatch(/@\/(context|api|services|lib\/api)/);
    expect(source).not.toMatch(/\bfetch\(|useContext/);
  });
});
