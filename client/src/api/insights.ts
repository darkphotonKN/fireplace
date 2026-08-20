import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The insights surface through the generated client (FS-0004, I-0019).
 *
 * SHAPE CHANGE from the hand-written versions in services/api.ts: responses are
 * bare. `getChecklistSuggestion` resolves to the suggestion string itself and
 * `getDailyInsights` to the array — no `{statusCode, message, result}` wrapper.
 *
 * `plan_id` stays snake_case here while notes uses camelCase. Both are
 * transcribed as they ship; normalizing is a later feature's job.
 *
 * NOTE: these three endpoints are served in-process by the gateway today and
 * are about to be repointed at insights-service over gRPC. This module is the
 * consumer half of the contract that makes that move observable.
 */
export type VideoSuggestion = components["schemas"]["VideoSuggestionResponse"];

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

export const getDailyInsights = async (planId: string): Promise<string[]> => {
  try {
    const { data, error, response } = await api.GET(
      "/api/insights/checklist-suggestion-daily",
      { params: { query: { plan_id: planId } } },
    );
    if (error) throw apiErrorFrom(error, response.status);
    return data ?? [];
  } catch (err) {
    console.error("Failed to get daily insights:", err);
    return [];
  }
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
