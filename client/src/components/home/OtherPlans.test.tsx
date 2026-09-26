import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';

/**
 * FS-KSJFR slice 3 (I-KSJFR-3): Your other plans.
 *
 * The load-bearing assertions are the two the spec actually argued about:
 * disjointness from Continue (R39/D6 — a plan listed twice reads as a
 * rendering bug), and the absence of the block below four plans (R41 — no
 * heading over nothing). The cap and the link out are the third.
 *
 * The component takes plans as props and fetches nothing, which is what keeps
 * ADR-0013's four-request budget intact; the no-request test below is the
 * guard on that, since a progress number on these rows is exactly the change
 * that would break it. That guard watches the API layer rather than
 * `globalThis.fetch`: the generated client captures `fetch` when it is created,
 * so a spy installed later would never see a call and the guard could not fail.
 */

const listPlans = vi.fn();
const listChecklists = vi.fn();

vi.mock('@/api/plans', () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
}));
vi.mock('@/api/checklists', () => ({
  listChecklists: (...a: unknown[]) => listChecklists(...a),
}));

import OtherPlans from './OtherPlans';
import type { Plan } from '@/api/plans';

const plan = (id: string, extra: Partial<Plan> = {}): Plan =>
  ({
    id,
    name: `Plan ${id}`,
    focus: `focus of ${id}`,
    description: '',
    planType: 'project',
    dailyReset: false,
    userId: 'u1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...extra,
  }) as Plan;

/**
 * `count` plans in the order `useHomeData` hands them over: most recently
 * updated first. p0/p1/p2 are therefore the three Continue cards.
 */
const ordered = (count: number) =>
  Array.from({ length: count }, (_, i) => plan(`p${i}`));

beforeEach(() => {
  listPlans.mockReset();
  listChecklists.mockReset();
});

describe('OtherPlans — disjoint from Continue (R39)', () => {
  it('should list the 4th through 9th plans and none of the three Continue cards', () => {
    render(<OtherPlans plans={ordered(10)} />);

    ['Plan p3', 'Plan p4', 'Plan p5', 'Plan p6', 'Plan p7', 'Plan p8'].forEach(
      (name) => expect(screen.getByText(name)).toBeInTheDocument(),
    );
    ['Plan p0', 'Plan p1', 'Plan p2'].forEach((name) =>
      expect(screen.queryByText(name)).toBeNull(),
    );
    expect(screen.getAllByRole('listitem')).toHaveLength(6);
  });

  it('should link each row into its own plan', () => {
    render(<OtherPlans plans={ordered(5)} />);

    expect(screen.getByRole('link', { name: /plan p3/i })).toHaveAttribute(
      'href',
      '/plan/p3',
    );
    expect(screen.getByRole('link', { name: /plan p4/i })).toHaveAttribute(
      'href',
      '/plan/p4',
    );
  });
});

describe('OtherPlans — the cap and the way out (R40)', () => {
  it.each([
    { total: 4, rows: 1, link: false },
    { total: 9, rows: 6, link: false },
    { total: 10, rows: 6, link: true },
    { total: 40, rows: 6, link: true },
  ])(
    'should show $rows rows for $total plans, with the all-plans link: $link',
    ({ total, rows, link }) => {
      render(<OtherPlans plans={ordered(total)} />);

      expect(screen.getAllByRole('listitem')).toHaveLength(rows);
      const out = screen.queryByRole('link', { name: /all plans/i });
      if (link) {
        expect(out).toHaveAttribute('href', '/myplans');
      } else {
        // Nothing is hidden, so there is nothing to send the user elsewhere for.
        expect(out).toBeNull();
      }
    },
  );

  it('should carry name and type only — no progress, and no request to compute one', () => {
    render(
      <OtherPlans
        plans={[
          ...ordered(3),
          plan('p3', { name: 'Quiet one', planType: 'learning' }),
        ]}
      />,
    );

    expect(screen.getByText('Quiet one')).toBeInTheDocument();
    expect(screen.getByText(/learning/i)).toBeInTheDocument();
    // Progress would cost a per-plan item fetch and break ADR-0013's cap.
    expect(screen.queryByText(/\d+\s*\/\s*\d+/)).toBeNull();
    expect(screen.queryByText('focus of p3')).toBeNull();
    // Progress would come from a per-plan checklist fetch, which is the call
    // ADR-0013's cap forbids this block from making.
    expect(listChecklists).not.toHaveBeenCalled();
    expect(listPlans).not.toHaveBeenCalled();
  });
});

describe('OtherPlans — named, and labelled the way /myplans labels', () => {
  it('should give its section the heading as an accessible name', () => {
    render(<OtherPlans plans={ordered(10)} />);

    // A landmark with no name is one more unlabelled "region" to page past.
    expect(
      screen.getByRole('region', { name: 'Your other plans' }),
    ).toBeInTheDocument();
  });

  it.each([
    { planType: 'project', label: 'Project' },
    { planType: 'learning', label: 'Learning' },
  ])(
    'should show $planType as "$label" — the name the row links to',
    ({ planType, label }) => {
      render(
        <OtherPlans plans={[...ordered(3), plan('p3', { planType })]} />,
      );

      expect(screen.getByText(label)).toBeInTheDocument();
    },
  );

  it('should show no type label at all when the plan carries none', () => {
    render(
      <OtherPlans
        plans={[...ordered(3), plan('p3', { name: 'Untyped', planType: undefined })]}
      />,
    );

    const row = screen.getByRole('link', { name: /untyped/i });
    // An empty span would be a silent gap where /myplans shows nothing.
    expect(row.textContent).toBe('Untyped');
  });
});

describe('OtherPlans — absent, not empty (R41)', () => {
  it.each([0, 1, 2, 3])(
    'should render nothing at all for %i plans — no heading, no empty state',
    (total) => {
      const { container } = render(<OtherPlans plans={ordered(total)} />);

      expect(container).toBeEmptyDOMElement();
      expect(screen.queryByRole('heading')).toBeNull();
      expect(screen.queryByRole('list')).toBeNull();
    },
  );
});

describe('OtherPlans — visual conformance (R47–R51)', () => {
  it('should style from tokens only — no raw colour, no gray utility', () => {
    const { container } = render(<OtherPlans plans={ordered(10)} />);

    const markup = container.innerHTML;
    expect(markup).not.toMatch(/rgba?\(/);
    expect(markup).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(markup).not.toMatch(/-gray-/);
  });

  it('should stay neutral — the page\'s one coral moment is not this block (R49)', () => {
    const { container } = render(<OtherPlans plans={ordered(10)} />);

    expect(container.querySelector('[class*="bg-primary"]')).toBeNull();
    expect(container.querySelector('[class*="text-primary"]')).toBeNull();
    // No h1 either: the global base style paints those coral.
    expect(container.querySelector('h1')).toBeNull();
  });

  it('should not wear the /myplans header card (R18)', () => {
    const { container } = render(<OtherPlans plans={ordered(10)} />);

    expect(container.querySelector('.backdrop-blur-sm')).toBeNull();
    expect(container.querySelector('.rounded-2xl')).toBeNull();
    expect(container.querySelector('.shadow-lg')).toBeNull();
  });
});
