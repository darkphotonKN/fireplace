'use client';

import Section from '../Section';
import ParallaxLayer from '../motion/ParallaxLayer';
import PlanCardMock from '../mocks/PlanCardMock';

export default function PlanSection() {
  return (
    <Section>
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-16">
        {/* Copy rides at native scroll speed — text never moves relative to the
            reader, which is what keeps the depth from costing legibility. */}
        <div>
          <h2 className="text-2xl font-bold md:text-3xl">Every arc gets a plan.</h2>
          <p className="mt-4 text-base leading-relaxed text-muted-foreground">
            Not a flat list of things to do. A plan holds the whole shape of the work —
            what comes first, what waits on it, and roughly when each part lands.
          </p>
        </div>

        {/* Mock art drifts slower, so it sits a little behind the words. */}
        <ParallaxLayer speed={0.6} className="flex justify-center md:justify-end">
          <PlanCardMock />
        </ParallaxLayer>
      </div>
    </Section>
  );
}
