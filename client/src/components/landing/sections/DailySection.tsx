'use client';

import Section from '../Section';
import ParallaxLayer from '../motion/ParallaxLayer';
import ChecklistMock from '../mocks/ChecklistMock';

export default function DailySection() {
  return (
    <Section>
      {/* Mock leads on desktop this time — alternating sides keeps the scroll
          from reading as one repeated template. Stacked order on mobile still
          puts the words first. */}
      <div className="grid items-center gap-12 md:grid-cols-2 md:gap-16">
        <ParallaxLayer
          speed={0.6}
          className="flex justify-center md:order-first md:justify-start"
        >
          <ChecklistMock />
        </ParallaxLayer>

        <div className="md:order-last">
          <h2 className="text-2xl font-bold md:text-3xl">Today is a short list.</h2>
          <p className="mt-4 text-base leading-relaxed text-muted-foreground">
            Long-term goals sit where you left them. Today&apos;s list is short on purpose — a
            few things picked this morning, and a clean slate tomorrow.
          </p>
        </div>
      </div>
    </Section>
  );
}
