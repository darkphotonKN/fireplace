'use client';

import Section from '../Section';
import ParallaxLayer from '../motion/ParallaxLayer';
import DailyReviewMock from '../mocks/DailyReviewMock';

export default function ReviewSection() {
  return (
    <Section>
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-16">
        <div>
          <h2 className="text-2xl font-bold md:text-3xl">Close the day honestly.</h2>
          <p className="mt-4 text-base leading-relaxed text-muted-foreground">
            Ticking boxes isn&apos;t the same as knowing how the day went. A short review at
            the end — what moved, what didn&apos;t, and why — is what makes tomorrow&apos;s
            list worth trusting.
          </p>
        </div>

        <ParallaxLayer speed={0.6} className="flex justify-center md:justify-end">
          <DailyReviewMock />
        </ParallaxLayer>
      </div>
    </Section>
  );
}
