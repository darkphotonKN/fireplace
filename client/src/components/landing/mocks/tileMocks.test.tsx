import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import type { ComponentType } from 'react';
import { afterEach, describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { expectMockContract } from '@/test/mockContract';
import { REVEAL_MS } from '../bento/BentoTile';
import CalendarTileMock from './CalendarTileMock';
import MilestonesTileMock from './MilestonesTileMock';
import PlanTileMock from './PlanTileMock';
import ReviewTileMock from './ReviewTileMock';
import TodayTileMock from './TodayTileMock';

/**
 * What every landing tile mock owes the page, checked once for all five:
 * decorative and inert (the mock contract), drawable with no app around it,
 * and never touching the network. Each mock's own file covers what it shows.
 */

const MOCKS: [string, ComponentType][] = [
  ['PlanTileMock', PlanTileMock],
  ['TodayTileMock', TodayTileMock],
  ['ReviewTileMock', ReviewTileMock],
  ['MilestonesTileMock', MilestonesTileMock],
  ['CalendarTileMock', CalendarTileMock],
];

/** A part's stagger, in ms (0 when it has none). */
const delayOf = (el: Element) =>
  parseInt((el as HTMLElement).style.getPropertyValue('--reveal-delay') || '0', 10);

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('landing tile mocks', () => {
  it.each(MOCKS)('%s honours the mock contract with no providers and no fetch', (_name, Mock) => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);

    const { container } = render(<Mock />);

    expectMockContract(container);
    expect(fetch).not.toHaveBeenCalled();
  });

  // The transition length is a CSS token jsdom can't evaluate, so it is read
  // from the module once; the staggers it is checked against are rendered.
  const revealMs = parseInt(
    /--reveal-ms:\s*(\d+)ms/.exec(
      readFileSync(resolve(__dirname, '../bento/bento.module.css'), 'utf8')
    )![1],
    10
  );

  it.each(MOCKS)('%s finishes its longest stagger inside the tile reveal window', (_name, Mock) => {
    const { container } = render(<Mock />);

    const delays = Array.from(container.querySelectorAll('[data-reveal-part]'), delayOf);
    expect(Math.max(0, ...delays) + revealMs).toBeLessThanOrEqual(REVEAL_MS);
  });
});
