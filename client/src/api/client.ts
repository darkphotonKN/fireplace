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
  PROFILE_NAME_EMPTY: "Name can't be empty.",
};

export function messageFor(problem: Problem | undefined): string {
  const code = problem?.code as ErrorCode | undefined;
  return (code && MESSAGES[code]) || MESSAGES.INTERNAL_ERROR;
}

/** Which form field a failure belongs to, when the code identifies one. */
export function fieldFor(problem: Problem | undefined): string | undefined {
  return problem?.code === "PROFILE_NAME_EMPTY" ? "name" : undefined;
}
