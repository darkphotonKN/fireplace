'use client';

import Section from '../Section';
import EmberGlow from '../EmberGlow';
import CtaLink, { SIGN_IN_HREF, SIGN_UP_HREF } from '../CtaLink';

/**
 * Documented product voice (see `docs/design-guideline.md`). Rendered verbatim —
 * rewording it is a brand decision, not a page decision.
 */
const ANCHOR_PHRASE = ['Start your plan now.', 'Sit down by the fire.'] as const;

export default function HeroSection() {
  return (
    <Section
      backdrop={<EmberGlow />}
      className="flex min-h-[92vh] items-center"
      contentClassName="max-w-2xl text-center"
    >
      {/* Coral, per the global `h1` rule — the page's single coral moment.
          The explicit size utility is required: base styles pin `h1` to 32px. */}
      <h1 className="text-5xl font-bold tracking-tight md:text-6xl">Fireplace</h1>

      <p className="mt-8 text-xl font-light leading-relaxed text-foreground/80 md:text-2xl">
        {/* The explicit space is load-bearing: `<br />` contributes nothing to
            textContent, so without it the phrase is announced as one run-on
            sentence with no pause between the two halves. */}
        {ANCHOR_PHRASE[0]}{' '}
        <br />
        {ANCHOR_PHRASE[1]}
      </p>

      <p className="mx-auto mt-6 max-w-xl text-base text-muted-foreground">
        The whole shape of a project, and the four things you&apos;re doing about it today.
        When the day closes, you write down how it actually went.
      </p>

      <div className="mt-10 flex items-center justify-center gap-4">
        <CtaLink href={SIGN_UP_HREF}>Start your plan</CtaLink>
        <CtaLink href={SIGN_IN_HREF} variant="outline">
          Sign in
        </CtaLink>
      </div>

      <p className="mt-16 text-sm text-muted-foreground">See how a day here runs ↓</p>
    </Section>
  );
}
