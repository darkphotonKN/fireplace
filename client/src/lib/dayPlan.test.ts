import { describe, it, expect } from 'vitest';
import type { MilestoneSourceItem } from './milestones';
import {
  findAddedStep,
  resolveDayPlan,
  type DayPlanEntrySource,
} from './dayPlan';

const item = (
  id: string,
  extra: Partial<MilestoneSourceItem> = {}
): MilestoneSourceItem => ({
  id,
  description: `item ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  ...extra,
});

const existing = (id: string, itemId: string | undefined): DayPlanEntrySource => ({
  id,
  kind: 'existing_item',
  itemId,
  text: `drawn text ${id}`,
  reason: `why ${id}`,
});

const proposed = (id: string, parentItemId: string | undefined): DayPlanEntrySource => ({
  id,
  kind: 'new_step',
  parentItemId,
  text: `new step ${id}`,
  reason: `why ${id}`,
});

// A plan's long-term items as the section sees them at render time, with
// every way a drawn entry can go stale since Plan my day (R67).
const ITEMS = [
  item('M1', { description: 'Ship the beta' }),
  item('S1', { parentId: 'M1', description: 'Write the release notes' }),
  item('S2', { parentId: 'M1', done: true, description: 'Fix the login bug' }),
  item('S3', { parentId: 'M1', archived: true }),
  item('G1', { parentId: 'S1', description: 'Draft the changelog' }),
  item('MA', { archived: true }),
  item('MC', { done: true }),
];

const one = (entry: DayPlanEntrySource) =>
  resolveDayPlan([{ label: 'L', entries: [entry] }], ITEMS);

describe('resolveDayPlan', () => {
  const cases: {
    name: string;
    entry: DayPlanEntrySource;
    shown: boolean;
    done?: boolean;
  }[] = [
    { name: 'live item', entry: existing('a', 'S1'), shown: true, done: false },
    { name: 'done item stays, shown completed', entry: existing('b', 'S2'), shown: true, done: true },
    { name: 'done milestone stays, shown completed', entry: existing('c', 'MC'), shown: true, done: true },
    { name: 'archived item', entry: existing('d', 'S3'), shown: false },
    { name: 'deleted item', entry: existing('e', 'gone'), shown: false },
    { name: 'existing item with no item id', entry: existing('f', undefined), shown: false },
    { name: 'new step under a live parent', entry: proposed('g', 'M1'), shown: true, done: false },
    { name: 'new step under a live nested parent', entry: proposed('h', 'S1'), shown: true, done: false },
    { name: 'new step under a closed parent', entry: proposed('i', 'MC'), shown: false },
    { name: 'new step under an archived parent', entry: proposed('j', 'MA'), shown: false },
    { name: 'new step under a deleted parent', entry: proposed('k', 'gone'), shown: false },
    { name: 'new step with no parent id', entry: proposed('l', undefined), shown: false },
  ];

  it.each(cases)('$name', ({ entry, shown, done }) => {
    const groups = one(entry);
    if (!shown) {
      expect(groups).toEqual([]);
      return;
    }
    expect(groups).toHaveLength(1);
    expect(groups[0].entries).toHaveLength(1);
    expect(groups[0].entries[0].id).toBe(entry.id);
    expect(groups[0].entries[0].done).toBe(done);
  });

  it('names the top-level ancestor as the milestone, and reads the live text', () => {
    const [group] = resolveDayPlan(
      [
        {
          label: 'Beta',
          entries: [
            existing('deep', 'G1'),
            { ...proposed('step', 'S1'), minutes: 20 },
          ],
        },
      ],
      ITEMS
    );
    expect(group.entries[0]).toMatchObject({
      milestoneTitle: 'Ship the beta',
      itemId: 'G1',
      text: 'Draft the changelog', // the item as it reads now
      reason: 'why deep',
    });
    expect(group.entries[1]).toMatchObject({
      milestoneTitle: 'Ship the beta',
      parentItemId: 'S1',
      text: 'new step step', // nothing live to read: the proposal as drawn
      minutes: 20,
    });
  });

  it('gives no separate milestone name to an entry that is the milestone itself', () => {
    const [group] = one(existing('m', 'M1'));
    expect(group.entries[0].milestoneTitle).toBeNull();
    expect(group.entries[0].text).toBe('Ship the beta');
  });

  it('keeps groups and entries in order and drops a group left empty', () => {
    const groups = resolveDayPlan(
      [
        { label: 'First', entries: [existing('1', 'S1'), existing('x', 'S3')] },
        { label: 'Stale', entries: [proposed('y', 'MA')] },
        { label: 'Last', entries: [proposed('2', 'M1'), existing('3', 'S2')] },
      ],
      ITEMS
    );
    expect(groups.map((g) => g.label)).toEqual(['First', 'Last']);
    expect(groups.flatMap((g) => g.entries.map((e) => e.id))).toEqual([
      '1',
      '2',
      '3',
    ]);
  });

  it('survives a parent cycle without hanging', () => {
    const cyclic = [
      item('A', { parentId: 'B' }),
      item('B', { parentId: 'A', description: 'B' }),
    ];
    const [group] = resolveDayPlan(
      [{ label: 'L', entries: [existing('c', 'A')] }],
      cyclic
    );
    expect(group.entries[0].id).toBe('c');
  });
});

describe('ticked (R82)', () => {
  it('carries the day tick through, false when absent', () => {
    const [group] = resolveDayPlan(
      [
        {
          label: 'L',
          entries: [{ ...existing('t', 'S1'), ticked: true }, existing('u', 'S2')],
        },
      ],
      ITEMS
    );
    expect(group.entries.map((e) => e.ticked)).toEqual([true, false]);
  });
});

describe('revisions (R66, R80, R81)', () => {
  it('carries a stamped revision and the switch through, and nothing when absent', () => {
    const [group] = resolveDayPlan(
      [
        {
          label: 'L',
          entries: [
            { ...existing('r', 'S1'), revision: 2, revisionsEnabled: true },
            existing('p', 'S2'),
          ],
        },
      ],
      ITEMS
    );
    const [stamped, plain] = group.entries;
    expect(stamped.revision).toBe(2);
    expect(stamped.revisionsEnabled).toBe(true);
    expect(plain).not.toHaveProperty('revision');
    expect(plain).not.toHaveProperty('revisionsEnabled');
  });

  // Switching revisions off cancels the item's open revision (R47), but the
  // entry drawn earlier that day still carries its stamp: the label must not
  // name a revision that no longer exists.
  it('drops the revision once revisions are switched off for the item', () => {
    const [entry] = one({
      ...existing('r', 'S1'),
      revision: 2,
      revisionsEnabled: false,
    }).flatMap((g) => g.entries);
    expect(entry).not.toHaveProperty('revision');
    expect(entry.revisionsEnabled).toBe(false);
  });
});

// A step added from a new_step entry, found again after a reload (R76).
describe('findAddedStep', () => {
  const kids = [
    item('M1', { description: 'Ship the beta' }),
    item('K1', { parentId: 'M1', description: '  Book   a DEMO slot ' }),
    item('K2', { parentId: 'M2', description: 'Book a demo slot' }),
    item('K3', { parentId: 'M1', description: 'Book a demo slot', archived: true }),
    item('K4', { parentId: 'M1', description: 'Book a demo slot', type: 'note' }),
  ];

  it('matches a child of the parent case- and whitespace-insensitively', () => {
    expect(findAddedStep('M1', 'book a demo  slot', kids)).toBe('K1');
  });

  it.each([
    ['another parent', 'M3', 'Book a demo slot'],
    ['edited text', 'M1', 'Book a demo slot today'],
    ['no parent', undefined, 'Book a demo slot'],
  ])('finds nothing for %s', (_, parent, text) => {
    expect(findAddedStep(parent, text, kids)).toBeUndefined();
  });

  it('skips archived children and notes: only a live task can be ticked (R25)', () => {
    expect(
      findAddedStep('M1', 'Book a demo slot', kids.filter((k) => k.id !== 'K1'))
    ).toBeUndefined();
  });

  it('marks a new step as added through the resolver', () => {
    const [group] = resolveDayPlan(
      [{ label: 'L', entries: [{ ...proposed('n', 'M1'), text: 'Book a demo slot' }] }],
      kids
    );
    expect(group.entries[0].addedItemId).toBe('K1');
  });
});
