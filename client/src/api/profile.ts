import { api, messageFor, type Problem } from "./client";
import type { components } from "./generated/schema";

/**
 * Profile read/write through the generated client (FS-0002, I-0006).
 *
 * Nothing here restates a request or response shape: `ProfileResponse` and
 * `UpdateProfileRequest` come from openapi.yaml via openapi-typescript. Change
 * the Go transport type and this file stops compiling — which is the point.
 */
export type UserProfile = components["schemas"]["ProfileResponse"];
export type UpdateProfileRequest = components["schemas"]["UpdateProfileRequest"];

/** Thrown with the domain code intact so callers can branch on it. */
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

export const getProfile = async (): Promise<UserProfile> => {
  const { data, error, response } = await api.GET("/users/profile");
  if (error) throw new ApiError(error as Problem, response.status);
  return data!;
};

export const updateProfile = async (
  updates: UpdateProfileRequest,
): Promise<UserProfile> => {
  const { data, error, response } = await api.PATCH("/users/profile", {
    body: updates,
  });
  if (error) throw new ApiError(error as Problem, response.status);
  return data!;
};
