import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, within } from '@testing-library/react';
import { expectMockContract } from '@/test/mockContract';
import { DATABASE_MILESTONE, MICROSERVICES_MILESTONE, progressOf } from './content';
import PlanTileMock from './PlanTileMock';

function card(container: HTMLElement, title: string) {
  const el = container.querySelector<HTMLElement>(`[data-milestone="${title}"]`);
  if (!el) throw new Error(`no ${title} card`);
  return el;
}

/** What a part shows at start: anything not waiting to arrive. */
function atStart(el: HTMLElement) {
  return !el.closest('[data-reveal-part="enter"]');
}

describe('PlanTileMock', () => {
  it('should honour the mock contract', () => {
    const { container } = render(<PlanTileMock />);

    expectMockContract(container);
  });

  it('should render with no app context providers at all', () => {
    expect(() => render(<PlanTileMock />)).not.toThrow();
  });

  it('should open on Database 1/5 and Microservices 6/8, as content.ts counts them', () => {
    expect(progressOf(DATABASE_MILESTONE)).toEqual({ done: 1, total: 5 });
    expect(progressOf(MICROSERVICES_MILESTONE)).toEqual({ done: 6, total: 8 });

    const { container } = render(<PlanTileMock />);

    const database = within(card(container, 'Database')).getByText('1/5');
    const micro = within(card(container, 'Microservices')).getByText('6/8');
    expect(atStart(database)).toBe(true);
    expect(atStart(micro)).toBe(true);
  });

  it('should cross-fade the Database count to 2/5 and fill its bar to match', () => {
    const { container } = render(<PlanTileMock />);
    const database = card(container, 'Database');

    const next = within(database).getByText('2/5');
    expect(next.closest('[data-reveal-part="enter"]')).not.toBeNull();

    const fill = database.querySelector<HTMLElement>('[data-reveal-part="fill"]')!;
    expect(fill.style.getPropertyValue('--reveal-from')).toBe(String(1 / 5));
    expect(fill.style.getPropertyValue('--reveal-to')).toBe(String(2 / 5));
  });

  it('should hold Microservices still on reveal: its bar starts and ends at 6/8', () => {
    const { container } = render(<PlanTileMock />);
    const fill = card(container, 'Microservices').querySelector<HTMLElement>(
      '[data-reveal-part="fill"]'
    )!;

    expect(fill.style.getPropertyValue('--reveal-from')).toBe(String(6 / 8));
    expect(fill.style.getPropertyValue('--reveal-to')).toBe(String(6 / 8));
  });

  it('should tick the first open Database step on reveal', () => {
    const { container } = render(<PlanTileMock />);
    const row = within(card(container, 'Database'))
      .getAllByText('SQL Tools Drill Practice')
      .map((el) => el.closest('[data-step]') as HTMLElement)[0];

    expect(row).toHaveAttribute('data-step-done', 'false');
    expect(row.querySelector('[data-reveal-part="enter"] [data-tick]')).not.toBeNull();
  });

  it('should show a few steps per card at start, legibly, and reveal more inside the cards', () => {
    const { container } = render(<PlanTileMock />);

    for (const title of ['Database', 'Microservices']) {
      const rows = Array.from(card(container, title).querySelectorAll<HTMLElement>('[data-step]'));
      const shown = rows.filter(atStart);
      const arriving = rows.filter((r) => !atStart(r));

      expect(shown.length).toBeGreaterThanOrEqual(2);
      expect(arriving.length).toBeGreaterThanOrEqual(1);
      // Every shown row carries its label, in its content.ts order.
      const milestone = title === 'Database' ? DATABASE_MILESTONE : MICROSERVICES_MILESTONE;
      rows.forEach((row, i) => {
        expect(row).toHaveAttribute('data-step-done', String(milestone.steps[i].done));
        expect(row).toHaveTextContent(milestone.steps[i].label);
      });
    }
  });

  it('should import no product component, context, service layer or fetch', () => {
    const source = readFileSync(resolve(__dirname, 'PlanTileMock.tsx'), 'utf8');

    expect(source).not.toMatch(/from '@\/components\/(?!landing)/);
    expect(source).not.toMatch(/@\/(context|api|services|lib\/api)/);
    expect(source).not.toMatch(/\bfetch\(|useContext/);
  });
});
