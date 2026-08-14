import { api, apiErrorFrom, ApiError } from "./client";
import type { components } from "./generated/schema";

// ApiError moved to ./client when users.ts also needed it. Re-exported so the
// callers already importing it from here keep working.
export { ApiError };

/**
 * Profile read/write through the generated client (FS-0002, I-0006).
 *
 * Nothing here restates a request or response shape: `ProfileResponse` and
 * `UpdateProfileRequest` come from openapi.yaml via openapi-typescript. Change
 * the Go transport type and this file stops compiling — which is the point.
 */
export type UserProfile = components["schemas"]["ProfileResponse"];
export type UpdateProfileRequest = components["schemas"]["UpdateProfileRequest"];

export const getProfile = async (): Promise<UserProfile> => {
  const { data, error, response } = await api.GET("/api/users/profile");
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

export const updateProfile = async (
  updates: UpdateProfileRequest,
): Promise<UserProfile> => {
  const { data, error, response } = await api.PATCH("/api/users/profile", {
    body: updates,
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};
