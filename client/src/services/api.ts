import {
  listChecklists,
  listArchivedChecklists,
  createChecklistItem as createItem,
  updateChecklistItem as updateItem,
  deleteChecklistItem as deleteItem,
  archiveChecklistItem as archiveItem,
  updateChecklistDates as updateDates,
  type ChecklistItem as ApiChecklistItem,
} from "@/api/checklists";
import { config } from "@/config/environment";

// API base URL from environment config
const API_BASE_URL = config.apiBaseUrl;

/**
 * Authenticated fetch wrapper. Attaches Bearer token from localStorage.
 * On 401: clears tokens and redirects to /auth.
 */
export async function authFetch(
  url: string,
  options: RequestInit = {},
): Promise<Response> {
  const token = localStorage.getItem("accessToken");

  const headers = new Headers(options.headers || {});
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(url, { ...options, headers });

  if (response.status === 401) {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    localStorage.removeItem("user");
    window.location.href = "/auth";
    throw new Error("Unauthorized");
  }

  return response;
}

export const scope = {
  longterm: "longterm",
  daily: "daily",
} as const;
export type ScopeEnum = (typeof scope)[keyof typeof scope];

export type ChecklistItemType = "task" | "note";

export interface ChecklistItem {
  id: string;
  description: string;
  done: boolean;
  /** @deprecated Backend dropped scheduled_time. UI still reads it where present
   *  for legacy schedule UX in Todo; the value now comes from start_date and is
   *  date-only (midnight UTC). New code should use startDate / dueDate. */
  scheduledTime?: string;
  /** ISO date string (YYYY-MM-DD) — Plan Calendar Gantt range start. */
  startDate?: string;
  /** ISO date string (YYYY-MM-DD) — Plan Calendar Gantt range end. */
  dueDate?: string;
  scope?: ScopeEnum;
  /** "task" (default) renders a checkbox; "note" is plain text, no checkbox. */
  type?: ChecklistItemType;
  /** Parent item id when nested; null/undefined for top-level rows. */
  parentId?: string | null;
}

export interface CalendarItem {
  id: string;
  description: string;
  scope: string;
  done: boolean;
  /** "" when null on the backend. */
  startDate: string;
  /** "" when null on the backend. */
  dueDate: string;
}

export interface CalendarResponse {
  statusCode: number;
  message: string;
  result: {
    planId: string;
    view: "week" | "month";
    windowStart: string;
    windowEnd: string;
    items: CalendarItem[];
  };
}

export type CalendarView = "week" | "month";

export interface UpdateChecklistDatesRequest {
  /** Omit to leave unchanged; null to clear; "YYYY-MM-DD" to set. */
  startDate?: string | null;
  dueDate?: string | null;
}

export interface ChecklistResponse {
  statusCode: number;
  message: string;
  result: ChecklistItem[];
}

export interface PlanResponse {
  statusCode: number;
  message: string;
  result: {
    items: ChecklistItem[];
    dailyReset: boolean;
  };
}

export interface ChecklistCreateRequest {
  description: string;
}

export interface UpdateChecklistItemRequest {
  description?: string;
  done?: boolean;
  scope?: ScopeEnum;
  type?: ChecklistItemType;
  /** Omit to leave parent_id alone; null to outdent; UUID to indent. */
  parentId?: string | null;
}

export interface DeleteChecklistItemResponse {
  result: "success" | "failure";
}

export interface UpdateChecklistItemResponse {
  result: "success" | "failure";
  item?: ChecklistItem;
}

export interface ChecklistSuggestionResponse {
  message: string;
  result: string;
  statusCode: number;
}

export interface DailyInsightsResponse {
  result: string[];
  message: string;
  statusCode: number;
}

export interface PlanDetailResponse {
  statusCode: number;
  message: string;
  result: PlanDetailData;
}

export interface PlanDetailData {
  id: string;
  name: string;
  description: string;
  focus: string;
}

export interface ApiResponse {
  statusCode: number;
  message: string;
  result: "success" | "failure";
}

export interface SearchPlan {
  id: string;
  name: string;
  focus: string;
  description: string;
  planType: string;
  dailyReset: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface SearchPlansResponse {
  statusCode: number;
  message: string;
  result: SearchPlan[];
}





















// --- Checklists ---
//
// SERIALIZED (FS-0004, I-0017). These are thin adapters over @/api/checklists,
// kept at their original names and ARGUMENT ORDER so the ~22 existing call
// sites keep compiling. What they no longer do is invent a response shape:
// each returns the bare resource, because the {statusCode, message, result}
// envelope is gone.
//
// Two parameters the endpoints never honoured were dropped rather than
// forwarded:
//   - `archived` on fetchChecklist — the gateway only read scope and type, so
//     passing true returned NON-archived items. Use fetchArchivedChecklist.
//   - `scope` on update/delete/archive — never read by those handlers.
// They stay in these signatures only where a caller still passes them, and are
// ignored here rather than sent, which is what already happened server-side.

export const fetchChecklist = async (
  planId: string,
  scope: "daily" | "longterm" = "daily",
  _archived: boolean = false,
  type?: ChecklistItemType,
): Promise<ApiChecklistItem[]> => listChecklists(planId, scope, type);

export const fetchArchivedChecklist = async (
  planId: string,
  _scope: "daily" | "longterm" = "daily",
): Promise<ApiChecklistItem[]> => listArchivedChecklists(planId);

export const createChecklistItem = async (
  description: string,
  planId: string,
  scope: "daily" | "longterm" = "daily",
  opts?: { type?: ChecklistItemType; parentId?: string | null },
): Promise<ApiChecklistItem> =>
  createItem(planId, {
    description,
    scope,
    ...(opts?.type ? { type: opts.type } : {}),
    // parentId is a plain optional string on CREATE, not three-state: there is
    // nothing to clear on an item that does not exist yet. null and undefined
    // both mean "top level", so only a real id is sent.
    ...(typeof opts?.parentId === "string" ? { parentId: opts.parentId } : {}),
  });

export const updateChecklistItem = async (
  id: string,
  updates: UpdateChecklistItemRequest,
  planId: string,
  _scope: "daily" | "longterm" = "daily",
): Promise<ApiChecklistItem> => updateItem(planId, id, updates);

export const deleteChecklistItem = async (
  id: string,
  planId: string,
  _scope: "daily" | "longterm" = "daily",
): Promise<void> => deleteItem(planId, id);

export const archiveChecklistItem = async (
  id: string,
  planId: string,
  _scope: "daily" | "longterm" = "daily",
): Promise<ApiChecklistItem> => archiveItem(planId, id, true);

export const updateChecklistDates = async (
  planId: string,
  checklistId: string,
  body: UpdateChecklistDatesRequest,
): Promise<ApiChecklistItem> => updateDates(planId, checklistId, body);

export const scheduleChecklistItem = async (
  id: string,
  planId: string,
  scheduleTime: Date,
  _scope: "daily" | "longterm" = "daily",
): Promise<ApiChecklistItem> =>
  updateDates(planId, id, { startDate: scheduleTime.toISOString().slice(0, 10) });

// --- Auth ---
//
// SERIALIZED (FS-0004, I-0015). The hand-written fetch versions of signIn and
// signUp are gone; these re-export the generated-client versions from
// @/api/users so existing import sites keep working.
//
// The response SHAPE changed with them: the {statusCode, message, result}
// envelope is removed, so signIn resolves with the token pair directly and
// signUp resolves with nothing. AuthUser is no longer restated here either —
// it comes from the contract as UserResponse, which also fixes its date fields,
// which were declared snake_case by hand and were never what the API returned.

export type { AuthResponse, UserResponse } from "@/api/users";
export { signIn, signUp, getUser, listUsers } from "@/api/users";

/** @deprecated Use UserResponse from the generated contract. */
export type AuthUser = import("@/api/users").UserResponse;

// --- User Profile ---

// Profile types come from the generated schema now — not restated by hand.
export type { UserProfile, UpdateProfileRequest } from "@/api/profile";

export { getProfile, updateProfile } from "@/api/profile";


