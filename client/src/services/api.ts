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

/**
 * Search plans by term with pagination.
 * Backend requires `term` to be non-empty; limit/offset are passed as strings
 * per the SearchParam struct's `form:"limit"` / `form:"offset"` tags.
 * Response has no total count — caller detects "last page" by checking if
 * the returned array has fewer than `limit` items.
 */
export const searchPlans = async (
  term: string,
  limit: number,
  offset: number,
): Promise<SearchPlansResponse> => {
  const params = new URLSearchParams({
    term,
    limit: String(limit),
    offset: String(offset),
  });
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/search?${params.toString()}`,
  );
  if (!response.ok) {
    throw new Error(`Failed to search plans: ${response.statusText}`);
  }
  return await response.json();
};

/**
 * Fetch Plan Information
 */
export const fetchPlan = async (id: string): Promise<PlanDetailResponse> => {
  const response = await authFetch(`${API_BASE_URL}/api/plans/${id}`);

  if (!response.ok) {
    throw new Error(`Failed to fetch checklist: ${response.statusText}`);
  }

  return await response.json();
};

/**
 * Fetch all checklist items for the specified plan
 */
export const fetchChecklist = async (
  planId: string,
  scope: "daily" | "longterm" = "daily",
  archived: boolean = false,
  type?: ChecklistItemType,
): Promise<ChecklistResponse> => {
  const params = new URLSearchParams({ scope, archived: String(archived) });
  if (type) params.set("type", type);
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}/checklists?${params.toString()}`,
  );

  if (!response.ok) {
    throw new Error(`Failed to fetch checklist: ${response.statusText}`);
  }

  return await response.json();
};

/**
 * Create a new checklist item
 */
export const createChecklistItem = async (
  description: string,
  planId: string,
  scope: "daily" | "longterm" = "daily",
  opts?: { type?: ChecklistItemType; parentId?: string | null },
): Promise<ChecklistItem> => {
  const body: Record<string, unknown> = { description, scope };
  if (opts?.type) body.type = opts.type;
  if (opts?.parentId !== undefined) body.parentId = opts.parentId;
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}/checklists`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    },
  );

  if (!response.ok) {
    throw new Error(`Failed to create checklist item: ${response.statusText}`);
  }

  const jsonRes = await response.json();

  return await jsonRes.result;
};

/**
 * Update a checklist item (description and/or done status)
 */
export const updateChecklistItem = async (
  id: string,
  updates: UpdateChecklistItemRequest,
  planId: string,
  scope: "daily" | "longterm" = "daily",
): Promise<UpdateChecklistItemResponse> => {
  try {
    const response = await authFetch(
      `${API_BASE_URL}/api/plans/${planId}/checklists/${id}?scope=${scope}`,
      {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updates),
      },
    );

    if (!response.ok) {
      throw new Error(
        `Failed to update checklist item: ${response.statusText}`,
      );
    }

    const data = await response.json();
    return { result: "success", item: data };
  } catch (err) {
    console.error("Failed to update checklist item:", err);
    return { result: "failure" };
  }
};

/**
 * Delete a checklist item
 */
export const deleteChecklistItem = async (
  id: string,
  planId: string,
  scope: "daily" | "longterm" = "daily",
): Promise<DeleteChecklistItemResponse> => {
  try {
    const response = await authFetch(
      `${API_BASE_URL}/api/plans/${planId}/checklists/${id}?scope=${scope}`,
      {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
      },
    );

    return await response.json();
  } catch (err) {
    console.log("Failed to delete checklist item, error:", err);
    return { result: "failure" };
  }
};

/**
 * Get an AI-generated checklist item suggestion
 */
export const getChecklistSuggestion = async (
  planId: string,
  scope: "daily" | "longterm" = "daily",
): Promise<ChecklistSuggestionResponse> => {
  try {
    const response = await authFetch(
      `${API_BASE_URL}/api/insights/checklist-suggestion?plan_id=${planId}&scope=${scope}`,
    );

    if (!response.ok) {
      throw new Error(
        `Failed to get checklist suggestion: ${response.statusText}`,
      );
    }

    return await response.json();
  } catch (err) {
    console.error("Failed to get checklist suggestion:", err);
    // Return a fallback suggestion if the API fails
    return {
      message: "Failed to generate suggestion",
      result: "Review your current project priorities",
      statusCode: 200,
    };
  }
};

/**
 * Update startDate / dueDate for a checklist item.
 * Body fields are optional with three-state semantics:
 *   - absent key      → leave column unchanged
 *   - explicit null   → clear column
 *   - "YYYY-MM-DD"    → set column
 * Backend validates start_date <= due_date post-merge with the current row.
 */
export const updateChecklistDates = async (
  planId: string,
  checklistId: string,
  body: UpdateChecklistDatesRequest,
): Promise<UpdateChecklistItemResponse> => {
  try {
    const response = await authFetch(
      `${API_BASE_URL}/api/plans/${planId}/checklists/${checklistId}/dates`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );

    if (!response.ok) {
      throw new Error(
        `Failed to update checklist dates: ${response.statusText}`,
      );
    }

    const data = await response.json();
    return { result: "success", item: data.result };
  } catch (err) {
    console.error("Failed to update checklist dates:", err);
    return { result: "failure" };
  }
};

/**
 * Legacy schedule call — kept for backwards-compat with Todo's schedule UI.
 * The backend /schedule endpoint was removed; this now PATCHes /dates with
 * a date-only startDate (time-of-day is dropped — known regression).
 */
export const scheduleChecklistItem = async (
  id: string,
  planId: string,
  scheduleTime: Date,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _scope: "daily" | "longterm" = "daily",
): Promise<UpdateChecklistItemResponse> => {
  const startDate = scheduleTime.toISOString().slice(0, 10);
  return updateChecklistDates(planId, id, { startDate });
};

/**
 * Fetch the per-plan calendar items for a window.
 * @param view "week" or "month"
 * @param date "YYYY-MM-DD" for week view, "YYYY-MM" for month view
 */
export const getPlanCalendar = async (
  planId: string,
  view: CalendarView,
  date: string,
): Promise<CalendarResponse> => {
  const params = new URLSearchParams({ view, date });
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}/calendar?${params.toString()}`,
  );
  if (!response.ok) {
    throw new Error(`Failed to fetch calendar: ${response.statusText}`);
  }
  return await response.json();
};

/**
 * Get daily insights suggestions based on long-term items
 */
export const getDailyInsights = async (
  planId: string,
): Promise<DailyInsightsResponse> => {
  try {
    const response = await authFetch(
      `${config.apiBaseUrl}/api/insights/checklist-suggestion-daily?plan_id=${planId}`,
    );

    if (!response.ok) {
      throw new Error(`Failed to get daily insights: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error("Failed to get daily insights:", error);
    // Return a fallback suggestion if the API fails
    return {
      message: "Failed to generate insights",
      result: [],
      statusCode: 200,
    };
  }
};

/**
 * Archive a checklist item
 */
export const archiveChecklistItem = async (
  id: string,
  planId: string,
  scope: "daily" | "longterm" = "daily",
): Promise<UpdateChecklistItemResponse> => {
  try {
    console.log("planId:", planId, " checklist id:", id);
    const response = await authFetch(
      `${API_BASE_URL}/api/plans/${planId}/checklists/${id}/archive?scope=${scope}`,
      {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
      },
    );

    if (!response.ok) {
      throw new Error(
        `Failed to archive checklist item: ${response.statusText}`,
      );
    }

    const data = await response.json();
    return { result: "success", item: data };
  } catch (err) {
    console.error("Failed to archive checklist item:", err);
    return { result: "failure" };
  }
};

/**
 * Fetch archived checklist items for the specified plan
 */
export const fetchArchivedChecklist = async (
  planId: string,
  scope: "daily" | "longterm" = "daily",
): Promise<ChecklistResponse> => {
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}/checklists/archived?scope=${scope}`,
  );

  if (!response.ok) {
    throw new Error(
      `Failed to fetch archived checklist: ${response.statusText}`,
    );
  }

  return await response.json();
};

export const toggleDailyReset = async (
  planId: string,
): Promise<ApiResponse> => {
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}/toggle-daily-reset`,
    {
      method: "PATCH",
    },
  );

  if (!response.ok) {
    throw new Error("Failed to toggle daily reset");
  }

  return response.json();
};

// --- Auth ---

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface SignInResponse {
  statusCode: number;
  message: string;
  result: {
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
    userInfo: AuthUser;
  };
}

export interface SignUpResponse {
  statusCode: number;
  message: string;
}

export const signIn = async (
  email: string,
  password: string,
): Promise<SignInResponse> => {
  const response = await fetch(`${API_BASE_URL}/api/users/signin`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.message || "Sign in failed");
  }

  return data;
};

export const signUp = async (
  name: string,
  email: string,
  password: string,
): Promise<SignUpResponse> => {
  const response = await fetch(`${API_BASE_URL}/api/users/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, email, password }),
  });

  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.message || "Sign up failed");
  }

  return data;
};

// --- User Profile ---

// Profile types come from the generated schema now — not restated by hand.
export type { UserProfile, UpdateProfileRequest } from "@/api/profile";

export { getProfile, updateProfile } from "@/api/profile";

export const fetchPlanDetails = async (
  planId: string,
): Promise<PlanResponse> => {
  const response = await authFetch(
    `${API_BASE_URL}/api/plans/${planId}`,
  );

  if (!response.ok) {
    throw new Error("Failed to fetch plan details");
  }

  return response.json();
};
