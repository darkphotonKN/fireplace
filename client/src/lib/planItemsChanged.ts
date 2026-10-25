/**
 * "This plan's items changed somewhere else on the page": a one-way nudge
 * between siblings that keep their own copy of a plan's items.
 *
 * The plan page's Day plan section and its middle card's `<Todo>` each read
 * the items themselves and share no state. When the section writes an item
 * (Also mark done, Add), it announces; the `<Todo>` listens and re-reads, so
 * its progress strip moves without a reload (FS-02JBZ R77).
 *
 * A window event rather than a context: the two never share a provider short
 * of the layout, and the signal carries nothing but the plan id. Listeners
 * re-read from the server; nothing is passed through the event.
 */

const EVENT = 'fireplace:plan-items-changed';

type Detail = { planId: string };

export function announcePlanItemsChanged(planId: string): void {
  window.dispatchEvent(new CustomEvent<Detail>(EVENT, { detail: { planId } }));
}

/** Calls `listener` whenever `planId`'s items are announced. Returns the unsubscribe. */
export function onPlanItemsChanged(
  planId: string,
  listener: () => void
): () => void {
  const handle = (event: Event) => {
    if ((event as CustomEvent<Detail>).detail?.planId === planId) listener();
  };
  window.addEventListener(EVENT, handle);
  return () => window.removeEventListener(EVENT, handle);
}
