'use client';

import Section from '../Section';
import BentoTile from './BentoTile';
import PlanTileMock from '../mocks/PlanTileMock';
import TodayTileMock from '../mocks/TodayTileMock';
import ReviewTileMock from '../mocks/ReviewTileMock';
import MilestonesTileMock from '../mocks/MilestonesTileMock';
import CalendarTileMock from '../mocks/CalendarTileMock';

/**
 * The feature tiles, after the hero (FS-XCTMG R9, R10). One column, then two at
 * md, then three at lg. Placement is DOM order alone: no `grid-flow-dense`, no
 * `order-*`, so the tab order is the order the eye reads. Wide tiles say so
 * with BentoTile's `span`, not with placement here.
 *
 * Inside `Section`, so the grid rises in with `riseIn` like every other beat.
 * Never parallaxed (R21): nothing here sits in a `ParallaxLayer`.
 */
export default function BentoGrid() {
  return (
    <Section>
      <div data-bento-grid className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <BentoTile label="Plan" mock={<PlanTileMock />} />
        {/* The hero badge's target; scroll-mt clears the fixed chrome bar. */}
        <BentoTile id="today" label="Today" inverted mock={<TodayTileMock />} className="scroll-mt-28" />
        <BentoTile label="Review" mock={<ReviewTileMock />} />
        {/* Row 2 at lg: two columns of three; one of two at md. */}
        <BentoTile label="Milestones" span="lg" mock={<MilestonesTileMock />} />
        {/* Closes row 2 at lg; at md the fifth tile fills its own row, no orphan cell. */}
        <BentoTile label="Calendar" span="md" mock={<CalendarTileMock />} />
      </div>
    </Section>
  );
}
