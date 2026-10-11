import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MilestoneStrip from './MilestoneStrip';
import { readMilestones, type MilestoneSourceItem } from '@/lib/milestones';

const item = (
  id: string,
  extra: Partial<MilestoneSourceItem> = {}
): MilestoneSourceItem => ({
  id,
  description: `Milestone ${id}`,
  done: false,
  scope: 'longterm',
  type: 'task',
  parentId: null,
  ...extra,
});

const many = (n: number) =>
  readMilestones(Array.from({ length: n }, (_, i) => item(`M${i}`)));

const segments = () =>
  Array.from(document.querySelectorAll<HTMLElement>('[data-milestone-segment]'));
const fills = () =>
  segments().map(
    (s) => s.querySelector<HTMLElement>('[data-milestone-fill]')!.style.width
  );
const track = () =>
  document.querySelector<HTMLElement>('[data-milestone-track]')!;

describe('MilestoneStrip', () => {
  it('should draw one segment per milestone, each filled to its own progress', () => {
    render(
      <MilestoneStrip
        reading={readMilestones([
          item('A'),
          item('A1', { parentId: 'A', done: true }),
          item('A2', { parentId: 'A', done: true }),
          item('A3', { parentId: 'A', done: true }),
          item('A4', { parentId: 'A' }),
          item('B', { done: true }),
          item('C'),
        ])}
      />
    );

    expect(screen.getByText('1 of 3 milestones')).toBeTruthy();
    expect(fills()).toEqual(['75%', '100%', '0%']);
  });

  it('should title each segment with its milestone and its progress', () => {
    render(
      <MilestoneStrip
        reading={readMilestones([
          item('A'),
          item('A1', { parentId: 'A', done: true }),
          item('A2', { parentId: 'A' }),
          item('B', { done: true }),
          item('C'),
        ])}
      />
    );

    expect(segments().map((s) => s.title)).toEqual([
      'Milestone A: 1/2 steps',
      'Milestone B: done',
      'Milestone C: not done',
    ]);
  });

  it('should render nothing at all for a plan with no milestones', () => {
    const { container } = render(<MilestoneStrip reading={many(0)} />);
    expect(container.innerHTML).toBe('');
  });

  it('should keep thirty milestones on one row of equal widths, gaps tighter than at twelve', () => {
    const { unmount } = render(<MilestoneStrip reading={many(12)} />);
    const atTwelve = parseFloat(track().style.columnGap);
    unmount();

    render(<MilestoneStrip reading={many(30)} />);

    expect(segments().length).toBe(30);
    // One column per milestone, each a share of the width that can shrink to
    // nothing: never a second row, never a horizontal scroll.
    expect(track().style.gridTemplateColumns).toBe(
      'repeat(30, minmax(0, 1fr))'
    );
    // At most 2px, less still once the track's share runs short.
    const atThirty = track().style.columnGap.match(/^min\(([\d.]+)px, /);
    expect(parseFloat(atThirty![1])).toBeLessThan(atTwelve);
  });

  it('should hold all the gaps to a share of the track, so even 150 milestones fit a phone-width card', () => {
    render(<MilestoneStrip reading={many(150)} layout="inline" />);

    // A fixed px gap past twelve is (M−1)×2px whatever the width: 298px of
    // gap alone at 150, wider than the track on a phone. Each gap is instead
    // the smaller of its px size and a slice of a fixed share of the track.
    const gap = getComputedStyle(track()).columnGap;
    const share = gap.match(/^min\(2px, ([\d.]+)%\)$/);
    expect(share).not.toBeNull();
    expect(149 * parseFloat(share![1])).toBeLessThanOrEqual(25);
  });

  it('should say "milestone" for a plan of one', () => {
    render(<MilestoneStrip reading={many(1)} />);
    expect(screen.getByText('0 of 1 milestone')).toBeTruthy();
  });
});
