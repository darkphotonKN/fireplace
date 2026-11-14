'use client';

import Logo from '@/components/Logo';
import Section from '../Section';
import EmberGlow from '../EmberGlow';
import CtaLink, { SIGN_IN_HREF, SIGN_UP_HREF } from '../CtaLink';

/**
 * Documented product voice (see `docs/design-guideline.md`). Rendered verbatim —
 * rewording it is a brand decision, not a page decision.
 */
const ANCHOR_PHRASE = ['Start your plan now.', 'Sit down by the fire.'] as const;

/** The one word that carries the coral underline: the page's single coral moment. */
const CORAL_WORD = 'fire';
const [BEFORE_CORAL, AFTER_CORAL] = ANCHOR_PHRASE[1].split(CORAL_WORD);

/** Locked marketing copy (FS-XCTMG D4); the glossary term in code is "entry". */
const BADGE_COPY = 'New · Day plan with AI moves ↗';

/** The Today tile's id. Owned by the tile; the badge only points at it. */
const TODAY_ANCHOR = '#today';

export default function HeroSection() {
  return (
    <Section
      backdrop={<EmberGlow />}
      className="flex min-h-[92vh] items-center"
      contentClassName="max-w-6xl text-center"
    >
      {/* Eyebrow: the mark and wordmark, small. Not a heading — the page has
          exactly one, and it is the anchor phrase below. */}
      <p className="flex items-center justify-center gap-2 text-base font-bold text-foreground">
        <Logo />
        <span>Fireplace</span>
      </p>

      {/* An in-page jump to the Today tile. A plain anchor, not `next/link`:
          it is a fragment, not a route, and the native jump already honours
          reduced motion (no smooth scroll is set). Copy is locked marketing
          text, used verbatim. */}
      <a
        href={TODAY_ANCHOR}
        className="mt-8 inline-flex items-center rounded-control border border-border bg-muted px-4 py-1.5 text-sm text-foreground transition-colors hover:bg-background"
      >
        {BADGE_COPY}
      </a>

      {/* Foreground, overriding the global coral `h1`: the coral moment is the
          underline on "fire", and a coral underline under coral glyphs would not
          read. Explicit size and colour utilities are required, since base styles
          pin `h1` to 32px coral. */}
      <h1 className="mt-6 text-5xl font-bold leading-[1.05] tracking-tight text-foreground md:text-6xl lg:text-7xl xl:text-8xl">
        {/* The explicit space is load-bearing: `<br />` contributes nothing to
            textContent, so without it the phrase is announced as one run-on
            sentence with no pause between the two halves. */}
        {ANCHOR_PHRASE[0]}{' '}
        <br />
        {BEFORE_CORAL}
        <span className="underline decoration-primary decoration-[0.08em] underline-offset-[0.12em]">
          {CORAL_WORD}
        </span>
        {AFTER_CORAL}
      </h1>

      <p className="mx-auto mt-8 max-w-2xl text-xl leading-relaxed text-muted-foreground md:text-2xl">
        The whole shape of a project, and the four things you&apos;re doing about it today.
        When the day closes, you write down how it actually went.
      </p>

      <div className="mt-10 flex flex-wrap items-center justify-center gap-3">
        <CtaLink href={SIGN_UP_HREF}>Start your plan</CtaLink>
        <CtaLink href={SIGN_IN_HREF} variant="muted">
          Sign in
        </CtaLink>
      </div>

      <p className="mt-16 text-sm text-muted-foreground">See how a day here runs ↓</p>
    </Section>
  );
}
