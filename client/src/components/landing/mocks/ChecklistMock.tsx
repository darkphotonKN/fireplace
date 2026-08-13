'use client';

import Mock from './Mock';
import CheckRow from './CheckRow';
import { GATEWAY_TASK } from './content';

/**
 * The dual-scope checklist. The two groups are the point of this mock: today's
 * list is short and resets, the long-term list sits where you left it.
 */
const TODAY = [
  { label: 'read ch.1', done: true },
  { label: 'stand-up notes', done: true },
  { label: GATEWAY_TASK, done: false },
] as const;

const LONG_TERM = [
  { label: 'ship the read model', done: false },
  { label: 'write the runbook', done: false },
] as const;

function GroupLabel({ children }: { children: string }) {
  return (
    <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground">{children}</p>
  );
}

export default function ChecklistMock() {
  return (
    <Mock className="w-full max-w-md">
      <GroupLabel>Today</GroupLabel>
      <ul className="mt-3 space-y-2">
        {TODAY.map((task) => (
          <CheckRow key={task.label} {...task} />
        ))}
      </ul>

      <div className="mt-6 border-t border-border pt-5">
        <GroupLabel>Long-term</GroupLabel>
        <ul className="mt-3 space-y-2">
          {LONG_TERM.map((task) => (
            // Dimmed: present, but not what today is about.
            <CheckRow key={task.label} {...task} dimmed />
          ))}
        </ul>
      </div>
    </Mock>
  );
}
