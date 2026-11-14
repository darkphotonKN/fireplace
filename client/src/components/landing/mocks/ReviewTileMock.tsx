'use client';

import { cn } from '@/lib/utils';
import Mock from './Mock';
import {
  DAY_LABEL,
  PERSONA_NAME,
  REVIEW_THREAD,
  type MockReviewMessage,
  type MockReviewSpeaker,
} from './content';
import { RevealIn } from '../bento/reveal';

const SENDER: Record<MockReviewSpeaker, string> = {
  persona: PERSONA_NAME,
  fireplace: 'Fireplace',
};

/** Consecutive messages from one speaker, shown under one name. */
interface Run {
  from: MockReviewSpeaker;
  messages: { message: MockReviewMessage; arrives: boolean }[];
}

/** The whole thread, start and arriving, grouped into runs by speaker. */
const RUNS: Run[] = [
  ...REVIEW_THREAD.start.map((message) => ({ message, arrives: false })),
  { message: REVIEW_THREAD.next, arrives: true },
].reduce<Run[]>((runs, entry) => {
  const last = runs[runs.length - 1];
  if (last?.from === entry.message.from) last.messages.push(entry);
  else runs.push({ from: entry.message.from, messages: [entry] });
  return runs;
}, []);

/**
 * The REVIEW tile (FS-XCTMG R15): the end of the Tuesday the TODAY tile plans,
 * as chat bubbles. Chris's note sits on the right, Fireplace's replies on the
 * left. On reveal the next reply slides up into the slot it held, invisibly,
 * from the start, so the tile's box never changes (R11).
 */
export default function ReviewTileMock() {
  return (
    <Mock className="space-y-3 rounded-none border-0 border-transparent bg-transparent p-0 shadow-none">
      <p className="text-sm font-medium text-foreground/60">{DAY_LABEL} · end of day</p>
      {RUNS.map((run, i) => (
        <div
          key={i}
          className={cn('flex flex-col gap-1.5', run.from === 'persona' ? 'items-end' : 'items-start')}
        >
          <span className="px-1 text-xs text-foreground/45">{SENDER[run.from]}</span>
          {run.messages.map(({ message, arrives }) =>
            arrives ? (
              // Full width, so the bubble's width cap reads off the tile, not itself.
              <RevealIn
                key={message.text}
                className={cn('flex w-full', run.from === 'persona' ? 'justify-end' : 'justify-start')}
              >
                <Bubble message={message} />
              </RevealIn>
            ) : (
              <Bubble key={message.text} message={message} />
            )
          )}
        </div>
      ))}
    </Mock>
  );
}

/**
 * One message. Two lines at most: the copy is capped in content.ts, and clamped
 * here. Below `sm` the tile is one narrow column, so the bubble steps down a text
 * size, trims its padding and takes the tile's full width; the cap is sized for
 * that bubble, the narrowest there is.
 */
function Bubble({ message }: { message: MockReviewMessage }) {
  const mine = message.from === 'persona';
  return (
    <p
      data-review-message={message.from}
      className={cn(
        'max-w-full rounded-2xl px-3 py-2 text-xs leading-5',
        'sm:max-w-[90%] sm:px-3.5 sm:text-sm sm:leading-5',
        mine
          ? 'rounded-br-md bg-foreground/[0.07] text-foreground'
          : 'rounded-bl-md border border-foreground/[0.08] bg-background text-foreground/80'
      )}
    >
      {/* Clamped inside the padding, so a third line can't show in it. */}
      <span className="line-clamp-2">{message.text}</span>
    </p>
  );
}
