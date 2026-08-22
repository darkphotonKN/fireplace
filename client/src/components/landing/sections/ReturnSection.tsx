'use client';

import Section from '../Section';
import EmberGlow from '../EmberGlow';
import CtaLink, { SIGN_IN_HREF, SIGN_UP_HREF } from '../CtaLink';

/**
 * The close. Reprises the hero's hearth light so the scroll ends where it began
 * — the six sections are a loop, and the page has to *look* like it returns, not
 * merely say so.
 */
export default function ReturnSection() {
  return (
    <Section
      backdrop={<EmberGlow />}
      className="flex min-h-[70vh] items-center"
      contentClassName="max-w-2xl text-center"
    >
      <h2 className="text-2xl font-bold md:text-3xl">Tomorrow, the same fire.</h2>

      <p className="mx-auto mt-4 max-w-xl text-base leading-relaxed text-muted-foreground">
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
