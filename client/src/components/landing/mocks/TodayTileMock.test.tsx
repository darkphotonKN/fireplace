import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/react';
import { expectMockContract } from '@/test/mockContract';
import { REVEAL_MS } from '../bento/BentoTile';
import { DATABASE_MILESTONE, DAY_PLAN, MICROSERVICES_MILESTONE } from './content';
import TodayTileMock from './TodayTileMock';

const entries = DAY_PLAN.flatMap((group) => group.entries);
const tickedAtStart = entries.filter((e) => e.ticked).length;

function rows(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-entry]'));
}

/** Present at start: not inside a part that only arrives on reveal. */
function atStart(el: Element) {
  return !el.closest('[data-reveal-part="enter"]');
}

/** A part's stagger, in ms (0 when it has none). */
function delayOf(el: Element) {
  return parseInt((el as HTMLElement).style.getPropertyValue('--reveal-delay') || '0', 10);
}

describe('TodayTileMock', () => {
  it('should honour the mock contract', () => {
    const { container } = render(<TodayTileMock />);

    expectMockContract(container);
  });

  it('should render with no app context providers at all', () => {
    expect(() => render(<TodayTileMock />)).not.toThrow();
  });

  it('should draw every entry from an open step in content.ts, under its milestone', () => {
    const open = new Map(
      [DATABASE_MILESTONE, MICROSERVICES_MILESTONE].flatMap((m) =>
        m.steps.filter((s) => !s.done).map((s) => [s.label, m.title] as const)
      )
    );
    expect(entries.length).toBeGreaterThanOrEqual(3);

    const { container } = render(<TodayTileMock />);
    const shown = rows(container);

    expect(shown).toHaveLength(entries.length);
    entries.forEach((entry, i) => {
      expect(open.get(entry.step)).toBe(entry.milestone);
      expect(shown[i]).toHaveAttribute('data-entry', entry.step);
      expect(shown[i]).toHaveTextContent(entry.milestone);
      expect(shown[i]).toHaveTextContent(`~${entry.minutes} min`);
    });
  });

  it('should group the entries under short labels with a rough total each', () => {
    const { container } = render(<TodayTileMock />);
    const groups = Array.from(container.querySelectorAll<HTMLElement>('[data-day-group]'));

    expect(groups.map((g) => g.dataset.dayGroup)).toEqual(DAY_PLAN.map((g) => g.label));
    groups.forEach((group, i) => {
      expect(group.querySelectorAll('[data-entry]')).toHaveLength(DAY_PLAN[i].entries.length);
      expect(group).toHaveTextContent(/~(\d+h( \d+m)?|\d+ min)/);
    });
  });

  it('should open on a partly filled day: some entries ticked, the count and meter to match', () => {
    expect(tickedAtStart).toBeGreaterThan(0);
    expect(tickedAtStart).toBeLessThan(entries.length);

    const { container, getByText } = render(<TodayTileMock />);

    const ticks = rows(container).filter((row) =>
      Array.from(row.querySelectorAll('[data-tick]')).some(atStart)
    );
    expect(ticks).toHaveLength(tickedAtStart);
    expect(atStart(getByText(`${tickedAtStart} of ${entries.length} done today`))).toBe(true);
    container.querySelectorAll<HTMLElement>('[data-reveal-part="fill"]').forEach((fill) => {
      expect(fill.style.getPropertyValue('--reveal-from')).toBe(String(tickedAtStart / entries.length));
    });
  });

  it('should end with every entry ticked and the day meter full', () => {
    const { container, getByText } = render(<TodayTileMock />);

    rows(container).forEach((row) => {
      expect(row.querySelector('[data-tick]')).not.toBeNull();
    });
    const done = getByText(`${entries.length} of ${entries.length} done today`);
    expect(done.closest('[data-reveal-part="enter"]')).not.toBeNull();

    const tops = Array.from(
      container.querySelectorAll<HTMLElement>('[data-reveal-part="fill"]'),
      (fill) => Number(fill.style.getPropertyValue('--reveal-to'))
    );
    expect(Math.max(...tops)).toBe(1);
  });

  it('should land the ticks one after another, the meter stepping with each', () => {
    const { container } = render(<TodayTileMock />);

    const tickDelays = rows(container)
      .filter((row) => row.dataset.entryTicked === 'false')
      .map((row) => delayOf(row.querySelector('[data-reveal-part="enter"]')!));
    const fills = Array.from(container.querySelectorAll<HTMLElement>('[data-reveal-part="fill"]'));

    // Strictly later each time, in reading order.
    tickDelays.slice(1).forEach((d, i) => expect(d).toBeGreaterThan(tickDelays[i]));
    // One meter layer per tick, rising to that tick's share at that tick's moment.
    expect(fills.map(delayOf)).toEqual(tickDelays);
    expect(fills.map((f) => Number(f.style.getPropertyValue('--reveal-to')))).toEqual(
      tickDelays.map((_, k) => (tickedAtStart + k + 1) / entries.length)
    );
  });

  it('should finish its longest stagger inside the tile reveal window', () => {
    const css = readFileSync(resolve(__dirname, '../bento/bento.module.css'), 'utf8');
    const revealMs = parseInt(/--reveal-ms:\s*(\d+)ms/.exec(css)![1], 10);
    const { container } = render(<TodayTileMock />);

    const longest = Math.max(
      ...Array.from(container.querySelectorAll('[data-reveal-part]'), delayOf)
    );
    expect(longest + revealMs).toBeLessThanOrEqual(REVEAL_MS);
  });

  it('should say "entry", and import no product component, context, service layer or fetch', () => {
    const source = readFileSync(resolve(__dirname, 'TodayTileMock.tsx'), 'utf8');

    expect(source).not.toMatch(/\bmoves?\b/i);
    expect(source).not.toMatch(/from '@\/components\/(?!landing)/);
    expect(source).not.toMatch(/@\/(context|api|services|lib\/api)/);
    expect(source).not.toMatch(/\bfetch\(|useContext/);
  });
});
