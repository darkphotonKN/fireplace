import { api, apiErrorFrom } from "./client";
import type { components } from "./generated/schema";

/**
 * The users surface through the generated client (FS-0004, I-0015).
 *
 * Nothing here restates a request or response shape — every type below comes
 * from openapi.yaml via openapi-typescript. Change the Go transport type and
 * this file stops compiling, which is the whole point of the contract layer.
 *
 * SHAPE CHANGE from the legacy `services/api.ts` versions: responses are bare
 * resources. `signIn` used to return `{statusCode, message, result: {...}}` and
 * now returns the token pair directly. Callers read `data.accessToken`, not
 * `data.result.accessToken`.
 */
export type AuthResponse = components["schemas"]["AuthResponse"];
export type UserResponse = components["schemas"]["UserResponse"];
export type SignupRequest = components["schemas"]["SignupRequest"];
export type SigninRequest = components["schemas"]["SigninRequest"];

export const signIn = async (
  email: string,
  password: string,
): Promise<AuthResponse> => {
  const { data, error, response } = await api.POST("/api/users/signin", {
    body: { email, password },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * Registers an account. Resolves with nothing.
 *
 * The endpoint returns 201 with an EMPTY body: auth-service issues a token pair
 * on signup, but the gateway has always discarded it. Callers sign in
 * afterwards — which is exactly what AuthContext does. Returning those tokens
 * would be auto-login on signup: a real improvement, a real behaviour change,
 * and its own feature spec rather than something a retrofit smuggles in.
 */
export const signUp = async (
  name: string,
  email: string,
  password: string,
): Promise<void> => {
  const { error, response } = await api.POST("/api/users/signup", {
    body: { name, email, password },
  });
  if (error) throw apiErrorFrom(error, response.status);
};

export const getUser = async (id: string): Promise<UserResponse> => {
  const { data, error, response } = await api.GET("/api/users/{id}", {
    params: { path: { id } },
  });
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};

/**
 * Lists every user. UNBOUNDED — the endpoint has no pagination, and that was
 * transcribed rather than fixed (ADR-0006 §2). Treat the result as potentially
 * large.
 */
export const listUsers = async (): Promise<UserResponse[]> => {
  const { data, error, response } = await api.GET("/api/users");
  if (error) throw apiErrorFrom(error, response.status);
  return data!;
};
