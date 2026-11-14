'use client';

import Section from '../Section';
import EmberGlow from '../EmberGlow';
import CtaLink, { SIGN_IN_HREF, SIGN_UP_HREF } from '../CtaLink';

/**
 * The close. Reprises the hero's hearth light so the scroll ends where it began
 * — the page is a loop, and it has to *look* like it returns, not merely say
 * so. Sized to sit with the hero (FS-XCTMG R22): x.ai scale, a step below the
 * `h1`, and in `foreground` — the underline on "fire" stays the page's one
 * coral moment.
 */
export default function ReturnSection() {
  return (
    <Section
      backdrop={<EmberGlow />}
      className="flex min-h-[80vh] items-center"
      contentClassName="max-w-4xl text-center"
    >
      {/* Explicit size and colour: base styles pin `h2` to 24px. */}
      <h2 className="text-4xl font-bold leading-[1.1] tracking-tight text-foreground md:text-5xl lg:text-6xl">
        Tomorrow, the same fire.
      </h2>

      <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-muted-foreground md:text-xl">
        Plan it, work it, close it honestly. What you wrote down is where tomorrow&apos;s
        list starts. Then you sit down and do it again.
      </p>

      {/* One primary action, not a matched pair. By here the visitor is warm,
          and two equal-weight buttons read as hesitation. */}
      <div className="mt-10 flex flex-col items-center gap-5">
        <CtaLink href={SIGN_UP_HREF}>Start your plan</CtaLink>
        <CtaLink href={SIGN_IN_HREF} variant="quiet">
          Sign in
        </CtaLink>
      </div>
    </Section>
  );
}
