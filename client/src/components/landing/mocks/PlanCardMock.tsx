'use client';

import Mock from './Mock';
import CheckRow from './CheckRow';

/** Invented content. Concrete enough to be recognisable, vague enough to age well. */
const PLAN_TITLE = 'Learn microservices';

const TASKS = [
  { label: 'read ch.1', done: true },
  { label: 'sketch the diagram', done: true },
  { label: 'wire up the gateway', done: false },
] as const;

/** Soft Gantt bars: [left offset %, width %]. Staggered to read as a sequence. */
const BARS = [
  [0, 38],
  [14, 52],
  [40, 34],
] as const;

export default function PlanCardMock() {
  return (
    <Mock className="w-full max-w-md">
      <p className="text-sm font-bold text-foreground">{PLAN_TITLE}</p>

      <ul className="mt-4 space-y-2">
        {TASKS.map((task) => (
          <CheckRow key={task.label} {...task} />
        ))}
      </ul>

      <div className="mt-6 space-y-2 border-t border-border pt-5">
        {BARS.map(([offset, width], i) => (
          <div key={`${offset}-${width}`} className="h-2 w-full">
            <div
              className="h-2 rounded-full bg-primary"
              style={{
                marginLeft: `${offset}%`,
                width: `${width}%`,
                // Descending emphasis: the near term is solid, later work fades.
                opacity: 0.55 - i * 0.15,
              }}
            />
          </div>
        ))}
      </div>
    </Mock>
  );
}
