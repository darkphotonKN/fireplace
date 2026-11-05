import createClient from "openapi-fetch";
import { config } from "@/config/environment";
import type { paths } from "./generated/schema";

/**
 * The one place that attaches the bearer, renews an expired access token and
 * signs out (FS-ZESEF R13). Both client HTTP paths go through it: `client.ts`
 * passes `sessionFetch` to `createClient`, and `authFetch` in `services/api.ts`
 * delegates to `sessionFetchOrThrow`.
 *
 * `client.ts` imports this module, so this module must never import it.
 */

/**
 * The stated exception to `client.ts`'s "only import path" rule (R18): the
 * refresh call must not pass through `sessionFetch`, or a 401 from refresh would
 * try to refresh again. Still the generated client, so a contract change breaks
 * at `tsc`.
 *
 * openapi-fetch captures its default `fetch` when `createClient` runs, so the
 * lookup is deferred to call time.
 */
const refreshClient = createClient<paths>({
  baseUrl: config.apiBaseUrl,
  fetch: (request) => globalThis.fetch(request),
});

type Recovery =
  /** A usable access token: retry with it. */
  | { kind: "renewed"; accessToken: string }
  /** The session is over: sign out. */
  | { kind: "rejected" }
  /** No verdict on the session: keep the tokens, give the caller its 401. */
  | { kind: "kept" };

function storedUserId(): string | undefined {
  try {
    return JSON.parse(localStorage.getItem("user") ?? "null")?.id;
  } catch {
    return undefined;
  }
}

/** Past this, a stalled refresh counts as a failed one, so it cannot hold `inFlight` forever. */
const REFRESH_DEADLINE_MS = 10_000;

async function exchange(refreshToken: string): Promise<Recovery> {
  try {
    // Its own deadline and no caller's signal: the exchange belongs to everyone
    // sharing it (R21).
    const { data, response } = await refreshClient.POST("/api/users/refresh", {
      body: { refreshToken },
      signal: AbortSignal.timeout(REFRESH_DEADLINE_MS),
    });
    if (data) {
      // The user signed out or switched account while this was in flight (R28).
      if (!data.userInfo?.id || data.userInfo.id !== storedUserId()) return { kind: "kept" };
      localStorage.setItem("accessToken", data.accessToken);
      localStorage.setItem("refreshToken", data.refreshToken);
      return { kind: "renewed", accessToken: data.accessToken };
    }
    // Same race as R28, from the other side: this verdict is about the token we
    // sent, so it must not wipe a session that has replaced it.
    const stillCurrent = localStorage.getItem("refreshToken") === refreshToken;
    return response.status === 401 && stillCurrent ? { kind: "rejected" } : { kind: "kept" };
  } catch {
    // A network error, the deadline, or a 200 whose body is not JSON (R24).
    return { kind: "kept" };
  }
}

let inFlight: Promise<Recovery> | null = null;

function recover(failedBearer: string): Promise<Recovery> {
  const stored = localStorage.getItem("accessToken");
  // Already replaced, in this tab or another: nothing to exchange (R20).
  if (stored && stored !== failedBearer) {
    return Promise.resolve({ kind: "renewed", accessToken: stored });
  }
  if (inFlight) return inFlight;

  const refreshToken = localStorage.getItem("refreshToken");
  if (!refreshToken) return Promise.resolve({ kind: "rejected" });

  inFlight = exchange(refreshToken).finally(() => {
    inFlight = null;
  });
  return inFlight;
}

/** Rejects with the caller's abort while the shared work carries on without it. */
function unlessAborted<T>(work: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) return reject(signal.reason);
    const onAbort = () => reject(signal.reason);
    signal.addEventListener("abort", onAbort, { once: true });
    work.then(resolve, reject).finally(() => signal.removeEventListener("abort", onAbort));
  });
}

function signOut() {
  localStorage.removeItem("accessToken");
  localStorage.removeItem("refreshToken");
  localStorage.removeItem("user");
  window.location.href = "/auth";
}

function send(request: Request, bearer: string | null): Promise<Response> {
  if (bearer) request.headers.set("Authorization", `Bearer ${bearer}`);
  // Looked up per call so a stub installed after this module loads still applies.
  return globalThis.fetch(request);
}

async function run(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<{ response: Response; signedOut: boolean }> {
  // One shape for both callers: openapi-fetch's (request, init), authFetch's (url, options).
  const request = new Request(input, init);
  // Taken before the first send consumes the body, so the retry can resend it (R17).
  const retry = request.clone();
  const bearer = localStorage.getItem("accessToken");
  const sentFor = storedUserId();

  const response = await send(request, bearer);
  if (response.status !== 401) return { response, signedOut: false };

  if (!bearer) {
    signOut();
    return { response, signedOut: true };
  }

  // This request belongs to whoever was signed in when it left. Resending it now
  // could land their write in another account, and its 401 says nothing about
  // the session that has replaced theirs.
  if (storedUserId() !== sentFor) return { response, signedOut: false };

  const recovery = await unlessAborted(recover(bearer), request.signal);
  if (recovery.kind === "kept") return { response, signedOut: false };
  if (recovery.kind === "rejected") {
    signOut();
    return { response, signedOut: true };
  }

  const retried = await send(retry, recovery.accessToken);
  if (retried.status !== 401) return { response: retried, signedOut: false };
  signOut();
  return { response: retried, signedOut: true };
}

/** For `createClient`. A sign-out still resolves with the 401, which the typed client maps to `ApiError`. */
export async function sessionFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  return (await run(input, init)).response;
}

/** As `sessionFetch`, but a sign-out throws: the contract `authFetch` keeps (R25). */
export async function sessionFetchOrThrow(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const { response, signedOut } = await run(input, init);
  if (signedOut) throw new Error("Unauthorized");
  return response;
}
