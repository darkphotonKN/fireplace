import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The insights surface through the generated client (FS-0004, I-0019).
 *
 * SHAPE CHANGE from the hand-written versions in services/api.ts: responses are
 * bare. `getChecklistSuggestion` resolves to the suggestion string itself —
 * no `{statusCode, message, result}` wrapper.
 *
 * `plan_id` stays snake_case here while notes uses camelCase. Both are
 * transcribed as they ship; normalizing is a later feature's job.
 *
 * NOTE: these three endpoints are served in-process by the gateway today and
 * are about to be repointed at insights-service over gRPC. This module is the
 * consumer half of the contract that makes that move observable.
 */
export type VideoSuggestion = components["schemas"]["VideoSuggestionResponse"];
export type DayPlan = components["schemas"]["DayPlanResponse"];
export type DayPlanGroup = components["schemas"]["DayPlanGroupResponse"];
export type DayPlanEntry = components["schemas"]["DayPlanEntryResponse"];

/** Fallback preserved from services/api.ts: a failed suggestion shows copy
 *  rather than nothing. Masks the error by design — pre-existing behaviour. */
const SUGGESTION_FALLBACK = "Review your current project priorities";

export const getChecklistSuggestion = async (
  planId: string,
): Promise<string> => {
  try {
    const { data, error, response } = await api.GET(
      "/api/insights/checklist-suggestion",
      { params: { query: { plan_id: planId } } },
    );
    if (error) throw apiErrorFrom(error, response.status);
    return data!;
  } catch (err) {
    console.error("Failed to get checklist suggestion:", err);
    return SUGGESTION_FALLBACK;
  }
};

/**
 * The caller's day plan for a plan and a browser-local date (FS-02JBZ R14,
 * R64). A read NEVER generates (ADR-0019, D7): a day nobody has planned comes
 * back as `status: not_planned` with no groups.
 *
 * THROWS an `ApiError` on failure: the section shows a quiet retry (R71),
 * which covers `BRIEF_DATE_OUT_OF_RANGE` (a client clock off by more than a
 * day) and 503. A 404 means the plan is not the caller's to see (R70).
 */
export const getDayPlan = async (
  planId: string,
  date: string,
): Promise<DayPlan> => {
  const { data, error, response } = await api.GET("/api/insights/day-plan", {
    params: { query: { planId, date } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * Plan my day / Plan again (FS-02JBZ R13, R15, R17, R65, R68): draws or
 * redraws the caller's day plan for a plan and date and returns it. The first
 * successful draw of a date is free; when nothing changed since the stored
 * plan was drawn, the stored plan comes back (same `generatedAt`) and nothing
 * is spent.
 *
 * THROWS an `ApiError`. Callers switch on its `code` (ADR-0004):
 * `BRIEF_REFRESH_LIMIT_REACHED` (429) means no redraws are left and the
 * stored plan is unchanged; anything else, 503 included, is worth retrying.
 */
export const planMyDay = async (
  planId: string,
  date: string,
): Promise<DayPlan> => {
  const { data, error, response } = await api.POST("/api/insights/day-plan", {
    body: { planId, date },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * The day tick, "done for today" (FS-02JBZ R24–R28, R75): ticks an entry by
 * id and returns it ticked. Never touches the item's `done` (R28).
 *
 * `itemId` is only for a `new_step` entry: the item Add created from it (R25,
 * S5). The server checks it and binds the entry, so the entry comes back as
 * `existing_item` for that item. Omit it for an `existing_item` entry.
 * Idempotent: ticking a ticked entry returns it unchanged.
 *
 * THROWS an `ApiError`. Callers switch on its `code` (ADR-0004):
 * `DAY_PLAN_ENTRY_NOT_TICKABLE` (400) means the entry can't be ticked as it
 * stands (its item archived or deleted, or no valid `itemId`) and the day
 * plan on screen is stale.
 */
export const tickDayPlanEntry = async (
  entryId: string,
  itemId?: string,
): Promise<DayPlanEntry> => {
  const { data, error, response } = await api.PUT(
    "/api/insights/day-plan/entries/{id}/tick",
    {
      params: { path: { id: entryId } },
      body: itemId ? { itemId } : {},
    },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * Removes an entry's day tick, and the revision that tick scheduled
 * (FS-02JBZ R29, R46, R75). Unticking an unticked entry succeeds.
 *
 * THROWS an `ApiError` on failure.
 */
export const untickDayPlanEntry = async (entryId: string): Promise<void> => {
  const { error, response } = await api.DELETE(
    "/api/insights/day-plan/entries/{id}/tick",
    { params: { path: { id: entryId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
};

/**
 * Switches the revision ladder on or off for one item of a plan, for the
 * caller (FS-02JBZ R47, R48, R80). Off cancels the item's open revision; on
 * schedules nothing by itself, the item's next day tick does. Accepted, with
 * no effect, on a project plan.
 *
 * THROWS an `ApiError`: 404 when the item is not in the plan or the plan is
 * not the caller's to see.
 */
export const setItemRevisions = async (
  itemId: string,
  planId: string,
  enabled: boolean,
): Promise<void> => {
  const { error, response } = await api.PUT(
    "/api/insights/day-plan/items/{itemId}/revisions",
    {
      params: { path: { itemId } },
      body: { planId, enabled },
    },
  );
  if (error) throw apiErrorFrom(error, response.status);
};

export const getSuggestedVideos = async (
  planId: string,
): Promise<VideoSuggestion[]> => {
  const { data, error, response } = await api.GET(
    "/api/insights/suggest-videos",
    { params: { query: { plan_id: planId } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data ?? [];
};
