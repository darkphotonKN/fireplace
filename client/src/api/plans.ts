import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The plans surface through the generated client (FS-0004, I-0016).
 *
 * SHAPE CHANGE from the hand-written versions in services/api.ts: responses are
 * bare resources. `getPlan` returns the plan itself, not
 * `{statusCode, message, result}`. Callers read `plan.name`, not
 * `response.result.name`.
 *
 * Date keys are camelCase (`createdAt`, `updatedAt`). PlanResp published them
 * snake_case; nothing in this client read them, so the rename is invisible here.
 */
export type Plan = components["schemas"]["PlanResp"];
export type CreatePlanRequest = components["schemas"]["CreatePlanReq"];
export type UpdatePlanRequest = components["schemas"]["UpdatePlanReq"];
export type SearchResult = components["schemas"]["SearchResult"];

export const listPlans = async (): Promise<Plan[]> => {
  const { data, error, response } = await api.GET("/api/plans");
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const getPlan = async (id: string): Promise<Plan> => {
  const { data, error, response } = await api.GET("/api/plans/{id}", {
    params: { path: { id } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const createPlan = async (plan: CreatePlanRequest): Promise<Plan> => {
  const { data, error, response } = await api.POST("/api/plans", { body: plan });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const updatePlan = async (
  id: string,
  updates: UpdatePlanRequest,
): Promise<Plan> => {
  const { data, error, response } = await api.PATCH("/api/plans/{id}", {
    params: { path: { id } },
    body: updates,
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const deletePlan = async (id: string): Promise<void> => {
  const { error, response } = await api.DELETE("/api/plans/{id}", {
    params: { path: { id } },
  });
  if (error) throw apiErrorFrom(error, response.status);
};

export const toggleDailyReset = async (id: string): Promise<Plan> => {
  const { data, error, response } = await api.PATCH(
    "/api/plans/{id}/toggle-daily-reset",
    { params: { path: { id } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const listSharedPlans = async (
  limit = 20,
  offset = 0,
): Promise<Plan[]> => {
  const { data, error, response } = await api.GET("/api/plans/shared", {
    params: { query: { limit, offset } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * Searches the caller's plans.
 *
 * The response carries no total count, so "last page" is detected by the
 * returned array being shorter than `limit` — unchanged from the legacy
 * behaviour, and the reason limit/offset are still exposed rather than hidden
 * behind a page abstraction.
 */
export const searchPlans = async (
  term: string,
  limit = 20,
  offset = 0,
): Promise<SearchResult[]> => {
  const { data, error, response } = await api.GET("/api/plans/search", {
    params: { query: { term, limit, offset } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};
