import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

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
 */
export type ChecklistItem = components["schemas"]["ChecklistResp"];
export type CreateChecklistRequest = components["schemas"]["CreateChecklistReq"];
export type UpdateChecklistRequest = components["schemas"]["UpdateChecklistReq"];
export type UpdateDatesRequest = components["schemas"]["UpdateDatesReq"];

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
  return data!;
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
};
