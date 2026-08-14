import createClient from "openapi-fetch";
import { config } from "@/config/environment";
import type { paths, components } from "./generated/schema";

/**
 * The typed client for the SERIALIZED gateway surface (ADR-0002).
 *
 * This is the ONLY import path allowed for endpoints that appear in
 * openapi.yaml. Hand-written fetch against a serialized endpoint is a HIGH
 * code-review finding. Endpoints NOT yet serialized keep using
 * `services/api.ts` and the legacy `{statusCode, message, result}` envelope —
 * that is the grandfather clause, not an inconsistency to clean up.
 *
 * `paths` is generated from openapi.yaml. Nothing in this file restates a
 * request or response shape by hand; if the backend contract moves, this fails
 * at `tsc` instead of at runtime.
 *
 * The baseUrl carries NO `/api`, and that changed. The document used to declare
 * `servers: [{url: /api}]` with paths written relative to it (`/users/profile`),
 * and since openapi-fetch never reads `servers` — it joins `baseUrl` + the
 * literal path and nothing else — the prefix had to live here.
 *
 * Mounting huma on the engine removed that indirection: operations now declare
 * their full path (`/api/users/profile`), so the document describes the real
 * URL. Keeping `/api` here would send every call to `/api/api/...`.
 *
 * That refactor left this file stale and nothing caught it, because the two
 * errors cancelled — a relative path in a stale `schema.d.ts` plus this prefix
 * produced the correct URL by accident. `make gates` has no client-staleness
 * check; adding one is I-0020's job. Until then, a contract change not followed
 * by `make client` is caught by nothing.
 */
export const api = createClient<paths>({
  baseUrl: config.apiBaseUrl,
});

// Attach the bearer token and mirror the legacy 401 handling so serialized and
// legacy calls behave identically from the user's point of view.
api.use({
  onRequest({ request }) {
    const token = localStorage.getItem("accessToken");
    if (token) request.headers.set("Authorization", `Bearer ${token}`);
    return request;
  },
  onResponse({ response }) {
    if (response.status === 401) {
      localStorage.removeItem("accessToken");
      localStorage.removeItem("refreshToken");
      localStorage.removeItem("user");
      window.location.href = "/auth";
    }
    return response;
  },
});

/** RFC 9457 problem, exactly as the contract publishes it. */
export type Problem = components["schemas"]["Problem"];

/**
 * Domain error codes (ADR-0004). Branch on THESE, never on `detail` — detail is
 * prose and is explicitly allowed to change between releases.
 */
export type ErrorCode =
  | "UNAUTHENTICATED"
  | "VALIDATION_FAILED"
  | "NOT_FOUND"
  | "ALREADY_EXISTS"
  | "FORBIDDEN"
  | "INTERNAL_ERROR"
  | "SERVICE_UNAVAILABLE"
  | "PROFILE_NAME_EMPTY";

/**
 * User-facing message per domain code.
 *
 * NOTE: `errors[]` is empty for downstream failures — gRPC statuses carry no
 * structured field detail — which is exactly why PROFILE_NAME_EMPTY exists as
 * its own code rather than a generic VALIDATION_FAILED with a field pointer.
 * The code has to carry the precision, so this map is the whole UX contract.
 */
const MESSAGES: Record<ErrorCode, string> = {
  UNAUTHENTICATED: "Your session has expired. Please sign in again.",
  VALIDATION_FAILED: "Some of the details you entered aren't valid.",
  NOT_FOUND: "We couldn't find that.",
  ALREADY_EXISTS: "That already exists.",
  FORBIDDEN: "You don't have access to that.",
  INTERNAL_ERROR: "Something went wrong on our end. Please try again.",
  // Distinct from INTERNAL_ERROR on purpose: this one is worth retrying, and
  // saying so is the entire reason FS-0004 R13 stopped mapping a downstream
  // outage to 500.
  SERVICE_UNAVAILABLE: "That service is temporarily unavailable. Please try again in a moment.",
  PROFILE_NAME_EMPTY: "Name can't be empty.",
};

export function messageFor(problem: Problem | undefined): string {
  const code = problem?.code as ErrorCode | undefined;
  return (code && MESSAGES[code]) || MESSAGES.INTERNAL_ERROR;
}

/**
 * Thrown with the domain code intact so callers can branch on it.
 *
 * Lives here rather than in profile.ts because every serialized module throws
 * it — profile.ts owned it when profile was the only serialized surface, and
 * it re-exports it now for the callers that already import it from there.
 */
export class ApiError extends Error {
  constructor(
    readonly problem: Problem | undefined,
    readonly status: number,
  ) {
    super(messageFor(problem));
    this.name = "ApiError";
  }
  get code() {
    return this.problem?.code;
  }
}

/** Builds an ApiError from openapi-fetch's untyped `error` channel. */
export function apiErrorFrom(error: unknown, status: number): ApiError {
  return new ApiError(error as Problem, status);
}

/** Which form field a failure belongs to, when the code identifies one. */
export function fieldFor(problem: Problem | undefined): string | undefined {
  return problem?.code === "PROFILE_NAME_EMPTY" ? "name" : undefined;
}
