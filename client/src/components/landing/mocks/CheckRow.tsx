'use client';

/**
 * One checklist row inside a mock: a fake checkbox and a label.
 *
 * The box is a `<span>`, never an `<input type="checkbox">` — mocks are
 * decorative and must expose nothing interactive. Shared by every mock that
 * shows tasks so the fake control exists in exactly one place.
 */
export default function CheckRow({
  label,
  done,
  dimmed,
}: {
  label: string;
  done: boolean;
  /** Present, but not what this beat is about. */
  dimmed?: boolean;
}) {
  const labelTone = done
    ? 'text-muted-foreground line-through'
    : dimmed
      ? 'text-muted-foreground'
      : 'text-foreground';

  return (
    <li className="flex items-center gap-3">
      <span
        className={
          done
            ? 'flex h-4 w-4 items-center justify-center rounded-sm bg-primary'
            : 'h-4 w-4 rounded-sm border border-border'
        }
      >
        {done && (
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
      <span className={`text-sm ${labelTone}`}>{label}</span>
    </li>
  );
}
