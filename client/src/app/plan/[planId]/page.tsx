"use client";

import Todo from "@/components/Todo";

/**
 * The plan's middle card at `/plan/<id>`: its long-term Items. Everything
 * around it, title card to videos, is the plan layout's (FS-043WP R2). The
 * dedicated Note block is hidden in this iteration; `NotesContainer` stays in
 * the codebase for a future feature.
 */
export default function PlanItems() {
  return <Todo fixedTaskType="longterm" enableTypeFilter />;
}
