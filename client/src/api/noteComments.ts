import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";
import { markTouchedToday } from "@/lib/touchedToday";

/**
 * A note's comments (FS-65B3N §API surface), through the generated client,
 * under the checklist item route. Open to the plan's owner and members; each
 * comment says what the caller may do with it (`canEdit`, `canDelete`), and
 * the client never works that out for itself (R12).
 *
 * Every mutation records the touched-today stamp once it lands, as the
 * mutations in `checklists.ts` do (FS-KSJFR R12). A read never does.
 *
 * THROWS an `ApiError`. Callers switch on its `code` (ADR-0004): `NOT_FOUND`
 * (the note, or the comment, is gone, or the plan can't be reached),
 * `ITEM_NOT_A_NOTE` (the item is a task now), `VALIDATION_FAILED` (a body empty
 * after trimming, or over 2000 characters).
 */
export type NoteComment = components["schemas"]["NoteCommentResp"];

const path = (planId: string, noteId: string) => ({
  params: { path: { id: planId, checklist_id: noteId } },
});

/** Every comment on the note, oldest first; [] when it has none (R11). */
export const listNoteComments = async (
  planId: string,
  noteId: string,
): Promise<NoteComment[]> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/checklists/{checklist_id}/comments",
    path(planId, noteId),
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data ?? [];
};

/** Adds a comment by the caller; the server trims it (R6, R8, R9). */
export const createNoteComment = async (
  planId: string,
  noteId: string,
  body: string,
): Promise<NoteComment> => {
  const { data, error, response } = await api.POST(
    "/api/plans/{id}/checklists/{checklist_id}/comments",
    { ...path(planId, noteId), body: { body } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

const onePath = (planId: string, noteId: string, commentId: string) => ({
  params: { path: { id: planId, checklist_id: noteId, comment_id: commentId } },
});

/**
 * Replaces a comment's body; its author only (R16). The same text as stored
 * changes nothing and comes back unedited. `FORBIDDEN` means the caller is
 * not the author; `NOT_FOUND`, that the comment is gone.
 */
export const updateNoteComment = async (
  planId: string,
  noteId: string,
  commentId: string,
  body: string,
): Promise<NoteComment> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}",
    { ...onePath(planId, noteId, commentId), body: { body } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

/**
 * Deletes a comment for good; its author or the plan's owner (R17).
 * `FORBIDDEN` means neither; `NOT_FOUND`, that it is already gone.
 */
export const deleteNoteComment = async (
  planId: string,
  noteId: string,
  commentId: string,
): Promise<void> => {
  const { error, response } = await api.DELETE(
    "/api/plans/{id}/checklists/{checklist_id}/comments/{comment_id}",
    onePath(planId, noteId, commentId),
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
};
