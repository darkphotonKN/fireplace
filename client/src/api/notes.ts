import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The notes surface through the generated client (FS-0004, I-0018).
 *
 * SHAPE CHANGE from the hand-written versions in services/notesService.ts:
 * responses are bare resources. `listNotes` returns the array itself, not
 * `{statusCode, message, result}`, and `deleteNote` answers 204 with no body
 * where it used to answer 200 with `result: "success"`.
 *
 * `tags` is repeatable — `?tags=a&tags=b` filters on both. That is declared in
 * the contract as `explode: true`, so the generated client serializes an array
 * as repeated params rather than a comma-joined string.
 *
 * `isRead`/`isDismissed` are STRINGS, not booleans, because that is what the
 * endpoint has always accepted: presence means "filter", and any value other
 * than "true" reads as false. Typing them as booleans here would invent a
 * stricter contract than the server enforces.
 */
export type Note = components["schemas"]["NoteResponse"];
export type CreateNoteBody = components["schemas"]["CreateNoteRequest"];
export type UpdateNoteBody = components["schemas"]["UpdateNoteRequest"];
export type AIMetadata = components["schemas"]["AIMetadataPayload"];

export type NoteFilters = {
  type?: string;
  priority?: string;
  isRead?: string;
  isDismissed?: string;
  relatedTaskId?: string;
  tags?: string[];
};

export const listNotes = async (
  planId: string,
  filters: NoteFilters = {},
): Promise<Note[]> => {
  const { data, error, response } = await api.GET("/api/plans/{id}/notes", {
    params: { path: { id: planId }, query: filters },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const getNote = async (
  planId: string,
  noteId: string,
): Promise<Note> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/notes/{noteId}",
    { params: { path: { id: planId, noteId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const createNote = async (
  planId: string,
  note: CreateNoteBody,
): Promise<Note> => {
  const { data, error, response } = await api.POST("/api/plans/{id}/notes", {
    params: { path: { id: planId } },
    body: note,
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/** Partial update. Omitted fields are left unchanged. */
export const updateNote = async (
  planId: string,
  noteId: string,
  updates: UpdateNoteBody,
): Promise<Note> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/notes/{noteId}",
    { params: { path: { id: planId, noteId } }, body: updates },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const deleteNote = async (
  planId: string,
  noteId: string,
): Promise<void> => {
  const { error, response } = await api.DELETE(
    "/api/plans/{id}/notes/{noteId}",
    { params: { path: { id: planId, noteId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
};

/**
 * Generates notes from the plan's focus and checklist.
 *
 * The body is REQUIRED even when every field is optional: the endpoint rejects
 * a request with no body at all. `{}` means "generate all kinds".
 */
export const generateAINotes = async (
  planId: string,
  requestType?: string,
): Promise<Note[]> => {
  const { data, error, response } = await api.POST(
    "/api/plans/{id}/notes/generate-ai",
    { params: { path: { id: planId } }, body: { requestType } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};
