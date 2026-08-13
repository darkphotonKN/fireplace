'use client';

import Section from '../Section';
import ParallaxLayer from '../motion/ParallaxLayer';
import SuggestionMock from '../mocks/SuggestionMock';

/**
 * Deliberately the smallest beat on the page: one card, centred, compact
 * spacing. The AI moment is a grace note at the end of the loop, not a pitch —
 * giving it parity with REVIEW would read as padding.
 */
export default function NudgeSection() {
  return (
    <Section size="compact" contentClassName="max-w-2xl text-center">
      <h2 className="text-2xl font-bold md:text-3xl">It notices what you keep skipping.</h2>

      <ParallaxLayer speed={0.6} className="mt-8 flex justify-center">
        <SuggestionMock />
      </ParallaxLayer>
    </Section>
  );
}
