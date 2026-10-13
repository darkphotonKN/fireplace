import type { Plan } from "@/api/plans";

/**
 * The quiet note on a plan someone else owns and shared with you (FS-12ZMF
 * R26). Owned plans carry no mark in v1, and neither does anything without a
 * `relation` (search hits are owned-only, D17), so only `member` shows it.
 *
 * Deliberately not a badge: lowercase, small, muted, no fill or border. It
 * sits beside a plan's name the way a pencilled note would, and inherits the
 * surrounding colour so it reads on the sidebar, a card or a list row alike.
 */
export default function SharedMark({
  plan,
  className = "",
}: {
  plan: Pick<Partial<Plan>, "relation">;
  className?: string;
}) {
  if (plan.relation !== "member") return null;
  return (
    <span
      title="Shared with you"
      className={`shrink-0 text-xs font-normal lowercase tracking-wide opacity-50 ${className}`}
    >
      shared
    </span>
  );
}
