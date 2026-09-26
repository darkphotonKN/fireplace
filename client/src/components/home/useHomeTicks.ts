"use client";

import { useCallback, useMemo, useState } from "react";
import { updateChecklistItem } from "@/api/checklists";
import { useToast } from "@/components/ui/use-toast";
import { withTicks, type HomeWork, type WorkRow } from "@/lib/homeWork";

export type HomeTicks = {
  /** The same `HomeWork` that went in, with this visit's ticks painted on. */
  work: HomeWork;
  /** Tick a row off. Optimistic; rolls itself back if the write is refused. */
  tick: (row: WorkRow) => void;
  /**
   * The ids ticked during this visit. Exposed because the Continue cards
   * compute their own progress from the fetched items and would otherwise
   * keep showing a pre-tick number for a plan whose row Today has already
   * struck through.
   */
  tickedIds: readonly string[];
};

/**
 * Ticking a row off home (FS-KSJFR R33–R35, D9).
 *
 * **One derivation, two readers.** `Dashboard` derives `selectHomeWork` once
 * precisely so the status line's sentence and the block's list can never
 * disagree about what is due (R16). Ticking preserves that: the ticked ids are
 * overlaid onto that one object, and both components go on reading the one
 * object. Giving the sentence and the list their own tick state would put the
 * disagreement back the moment a write failed under one of them.
 *
 * **The state is the id set, not the rows.** Everything visible — done rows,
 * decremented counts — is derived from it by `withTicks`, which is what makes
 * the failure path a one-liner and concurrency free: two ticks in flight are two
 * ids, and a refusal removes only its own (§Edge States). See `withTicks` for
 * why this is an overlay rather than a re-run of the selection.
 *
 * **The write goes through the API layer.** `updateChecklistItem` is where
 * I-KSJFR-1's touched-today stamp lives (R12), so ticking from home stamps the
 * day without this file knowing about it — which is what stops the day gate
 * firing again after the user has already acted here. A hand-rolled `fetch`
 * would silently cost that, exactly as I-0058 describes for delete.
 *
 * A refused write is told plainly (R34): a user must never be left believing
 * they finished something they didn't.
 */
export function useHomeTicks(work: HomeWork): HomeTicks {
  const [tickedIds, setTickedIds] = useState<readonly string[]>([]);
  const { toast } = useToast();

  const tick = useCallback(
    (row: WorkRow) => {
      if (row.done) return;
      setTickedIds((prev) => (prev.includes(row.id) ? prev : [...prev, row.id]));

      void updateChecklistItem(row.planId, row.id, { done: true }).catch(() => {
        // Its own id only: a sibling tick still in flight keeps its row and its
        // share of the count.
        setTickedIds((prev) => prev.filter((id) => id !== row.id));
        toast({
          title: "Couldn’t tick that off",
          description: `“${row.description}” has been put back. Please try again.`,
          position: "bottom-left",
        });
      });
    },
    [toast],
  );

  const ticked = useMemo(() => withTicks(work, tickedIds), [work, tickedIds]);

  return { work: ticked, tick, tickedIds };
}
