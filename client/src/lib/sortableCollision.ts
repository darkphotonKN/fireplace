import { closestCenter, type CollisionDetection } from '@dnd-kit/core';

/**
 * Which drop target a drag is over, chosen only among the dragged item's own
 * set: its fellow cards when a card is dragged, its fellow steps when a step
 * is (FS-0009 R8.1).
 *
 * One `DndContext` holds every set on the page, so plain `closestCenter`
 * weighs a step against the cards around it too. A card's centre sits in the
 * middle of its steps, so a step dragged past it lands "over" its own card,
 * and a card dragged across a neighbour lands over one of that neighbour's
 * steps. Over a target outside its own set, dnd-kit stops displacing the item
 * being dragged: it drops back to where it started until the pointer finds a
 * sibling again, and the drag stutters and lags the pointer. A drop across
 * sets is refused anyway (`moveOnto`), so those targets were only ever noise.
 */
export const closestInSameSet: CollisionDetection = (args) => {
  const set = args.active.data.current?.sortable?.containerId;
  if (set === undefined) return closestCenter(args);
  return closestCenter({
    ...args,
    droppableContainers: args.droppableContainers.filter(
      (target) => target.data.current?.sortable?.containerId === set
    ),
  });
};
