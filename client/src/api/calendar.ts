import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The calendar surface through the generated client (FS-0004, I-0019).
 *
 * SHAPE CHANGE: the response is the calendar itself, not
 * `{statusCode, message, result}`. `view` is forwarded as given — the gateway
 * does not validate it, so an unrecognised value reaches calendar-service
 * exactly as it did before.
 */
export type Calendar = components["schemas"]["CalendarResponse"];
export type CalendarItem = components["schemas"]["CalendarItemResponse"];

export const getPlanCalendar = async (
  planId: string,
  view: string,
  date: string,
): Promise<Calendar> => {
  const { data, error, response } = await api.GET(
    "/api/plans/{id}/calendar",
    { params: { path: { id: planId }, query: { view, date } } },
  );
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};
