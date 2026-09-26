import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";
import { markTouchedToday } from "@/lib/touchedToday";

/**
 * The checklists surface through the generated client (FS-0004, I-0017).
 *
 * SHAPE CHANGE from the hand-written versions in services/api.ts: responses are
 * bare resources. `listChecklists` returns the array itself, not
 * `{statusCode, message, result}`.
 *
 * TWO THINGS THE OLD SIGNATURES CARRIED THAT THE ENDPOINTS NEVER DID:
 *
 * - `fetchChecklist` sent an `archived` query param. The gateway only ever read
 *   `scope` and `type`, so it was ignored — passing archived: true returned
 *   NON-archived items. Archived items have their own endpoint, and this module
 *   exposes it as `listArchivedChecklists` rather than a flag that does nothing.
 * - Several calls sent `?scope=` on paths that ignore it entirely (update,
 *   delete, archive). Dropped here: a parameter the server does not read is not
 *   part of the contract.
 *
 * `scope` and `type` are now validated at the boundary. Sending a value outside
 * the enum returns 422 instead of travelling to plan-service — see the note in
 * typed_checklists.go.
 *
 * TOUCHED TODAY (FS-KSJFR R12): every mutating function below records the
 * touched-today stamp once its call has resolved successfully. It lives here
 * rather than in the callers because this is *almost* the one seam every
 * mutation crosses — add a mutation to this file, add the line. A read never
 * gets one (R13), and a rejected call never reaches it (R14).
 *
 * "Almost", and the exception is load-bearing: `deleteChecklistItem` below is
 * imported by `components/Todo.tsx:8` and never called. Both delete paths there
 * (:707, :1373) use a hand-written `fetch` that skips this file entirely — and,
 * carrying no Authorization header and an undefined env var, does not work at
 * all. See **I-0058**. So a delete records no stamp today, and this comment
 * says so rather than describing an invariant the product does not have. The
 * claim becomes true when I-0058 lands; until then FS-KSJFR R11 is correct
 * about what *should* stamp and the product is what is wrong.
 */
export type ChecklistItem = components["schemas"]["ChecklistResp"];
export type CreateChecklistRequest = components["schemas"]["CreateChecklistReq"];
export type UpdateChecklistRequest = components["schemas"]["UpdateChecklistReq"];
export type UpdateDatesRequest = components["schemas"]["UpdateDatesReq"];
export type ReorderChecklistRequest =
  components["schemas"]["ReorderChecklistReq"];

export type Scope = "daily" | "longterm";
export type ItemType = "task" | "note";

export const listChecklists = async (
  planId: string,
  scope?: Scope,
  type?: ItemType,
): Promise<ChecklistItem[]> => {
  const { data, error, response } = await api.GET("/api/plans/{id}/checklists", {
    params: { path: { id: planId }, query: { scope, type } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const listArchivedChecklists = async (
  planId: string,
): Promise<ChecklistItem[]> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/checklists/archived",
    { params: { path: { id: planId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const listUpcomingChecklists = async (
  planId: string,
): Promise<ChecklistItem[]> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/checklists/upcoming",
    { params: { path: { id: planId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const getChecklist = async (
  planId: string,
  checklistId: string,
): Promise<ChecklistItem> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/checklists/{checklist_id}",
    { params: { path: { id: planId, checklist_id: checklistId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const createChecklistItem = async (
  planId: string,
  item: CreateChecklistRequest,
): Promise<ChecklistItem> => {
  const { data, error, response } = await api.POST(
    "/api/plans/{id}/checklists",
    { params: { path: { id: planId } }, body: item },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

/**
 * Partial update.
 *
 * `parentId` is three-state and the distinction is load-bearing for
 * indent/outdent: omit it to leave the parent alone, send null to outdent to
 * top level, send an id to re-parent. Passing undefined is NOT the same as
 * passing null.
 */
export const updateChecklistItem = async (
  planId: string,
  checklistId: string,
  updates: UpdateChecklistRequest,
): Promise<ChecklistItem> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/checklists/{checklist_id}",
    {
      params: { path: { id: planId, checklist_id: checklistId } },
      body: updates,
    },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

/** Sets or clears dates. Same three-state rule as parentId. */
export const updateChecklistDates = async (
  planId: string,
  checklistId: string,
  dates: UpdateDatesRequest,
): Promise<ChecklistItem> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/checklists/{checklist_id}/dates",
    {
      params: { path: { id: planId, checklist_id: checklistId } },
      body: dates,
    },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

/**
 * Writes the order of ONE sibling set (I-0055).
 *
 * `ids` must be exactly a permutation of that set — the top-level items of a
 * (plan, scope) when `parentId` is null, or one parent's children when it is an
 * id. Not the visible subset: a set sent short of the items a filter happens to
 * be hiding is not a permutation and is refused whole.
 *
 * Sending the order a set already has is a no-op, not an error. Returns the
 * reordered siblings; the endpoint may answer with null rather than an empty
 * array, which is flattened here so callers always get a list.
 */
export const reorderChecklists = async (
  planId: string,
  body: ReorderChecklistRequest,
): Promise<ChecklistItem[]> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/checklists/order",
    { params: { path: { id: planId } }, body },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data ?? [];
};

/** A SETTER, not a toggle — pass false to unarchive. */
export const archiveChecklistItem = async (
  planId: string,
  checklistId: string,
  archived: boolean,
): Promise<ChecklistItem> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/checklists/{checklist_id}/archive",
    {
      params: { path: { id: planId, checklist_id: checklistId } },
      body: { archived },
    },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
  return data!;
};

export const deleteChecklistItem = async (
  planId: string,
  checklistId: string,
): Promise<void> => {
  const { error, response } = await api.DELETE(
    "/api/plans/{id}/checklists/{checklist_id}",
    { params: { path: { id: planId, checklist_id: checklistId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  markTouchedToday();
};
