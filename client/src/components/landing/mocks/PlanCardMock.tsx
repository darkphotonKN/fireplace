'use client';

import Mock from './Mock';

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
          <li key={task.label} className="flex items-center gap-3">
            {/* A square div, deliberately not an <input type="checkbox">. */}
            <span
              className={
                task.done
                  ? 'flex h-4 w-4 items-center justify-center rounded-sm bg-primary'
                  : 'h-4 w-4 rounded-sm border border-border'
              }
            >
              {task.done && (
                <svg viewBox="0 0 12 12" className="h-3 w-3 text-primary-foreground">
                  <path
                    d="M2.5 6.2 4.8 8.5 9.5 3.8"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.6"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              )}
            </span>
            <span
              className={
                task.done
                  ? 'text-sm text-muted-foreground line-through'
                  : 'text-sm text-foreground'
              }
            >
              {task.label}
            </span>
          </li>
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
