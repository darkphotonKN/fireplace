import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { config } from "@/config/environment";
import { authFetch } from "@/services/api";
import { api } from "./client";
import { sessionFetch } from "./session";

const USER = { id: "user-1", name: "Ada" };
const REFRESH_URL = `${config.apiBaseUrl}/api/users/refresh`;
const THING_URL = `${config.apiBaseUrl}/api/things`;

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function authResponse(
  accessToken = "new-access",
  refreshToken = "new-refresh",
  userInfo: unknown = USER,
) {
  return json(200, {
    accessToken,
    refreshToken,
    accessExpiresIn: 1,
    refreshExpiresIn: 1,
    userInfo,
  });
}

function signedIn() {
  localStorage.setItem("accessToken", "old-access");
  localStorage.setItem("refreshToken", "old-refresh");
  localStorage.setItem("user", JSON.stringify(USER));
}

/** A stand-in gateway: refresh and everything else answer from their own queue. */
function gateway(routes: {
  refresh?: (request: Request) => Response | Promise<Response>;
  other: (bearer: string | null) => Response | Promise<Response>;
}) {
  const refreshCalls: { body: string; authorization: string | null }[] = [];
  const otherCalls: { method: string; bearer: string | null; body: string }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: Request) => {
      // Consumes the body as a real fetch does, so resending an uncloned request fails here too.
      const body = await input.text();
      const authorization = input.headers.get("Authorization");
      if (input.url === REFRESH_URL) {
        refreshCalls.push({ body, authorization });
        if (!routes.refresh) throw new Error("unexpected refresh call");
        return routes.refresh(input);
      }
      const bearer = authorization?.replace("Bearer ", "") ?? null;
      otherCalls.push({ method: input.method, bearer, body });
      return routes.other(bearer);
    }),
  );
  return { refreshCalls, otherCalls };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

/** Lets every request already answered reach its 401 handling. */
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const unauthorized = () => json(401, { code: "UNAUTHENTICATED" });
const okIfNewToken = (bearer: string | null) =>
  bearer === "new-access" ? json(200, { ok: true }) : unauthorized();

beforeEach(() => {
  vi.stubGlobal("location", { href: "http://localhost/plans" });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("sessionFetch", () => {
  it("should retry with the new token and store the new pair when the access token expired", async () => {
    signedIn();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: okIfNewToken,
    });

    const response = await sessionFetch(THING_URL);

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ ok: true });
    expect(localStorage.getItem("accessToken")).toBe("new-access");
    expect(localStorage.getItem("refreshToken")).toBe("new-refresh");
    expect(refreshCalls).toEqual([
      { body: JSON.stringify({ refreshToken: "old-refresh" }), authorization: null },
    ]);
    expect(otherCalls.map((c) => c.bearer)).toEqual(["old-access", "new-access"]);
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it.each(["POST", "PATCH", "DELETE"])(
    "should resend the identical body when a %s is retried",
    async (method) => {
      signedIn();
      const { otherCalls } = gateway({
        refresh: () => authResponse(),
        other: okIfNewToken,
      });
      const body = JSON.stringify({ description: "keep what I typed" });

      const response = await sessionFetch(THING_URL, {
        method,
        body,
        headers: { "Content-Type": "application/json", "X-Custom": "kept" },
      });

      expect(response.status).toBe(200);
      expect(otherCalls).toEqual([
        { method, bearer: "old-access", body },
        { method, bearer: "new-access", body },
      ]);
    },
  );

  describe.each([
    ["a network error", () => Promise.reject(new TypeError("Failed to fetch"))],
    ["a 500", () => json(500, { code: "INTERNAL_ERROR" })],
    ["a 503", () => json(503, { code: "SERVICE_UNAVAILABLE" })],
    ["a 404", () => new Response("404 page not found", { status: 404 })],
    ["a 422", () => json(422, { code: "VALIDATION_FAILED" })],
    ["a 200 that is not JSON", () => new Response("<html>", { status: 200 })],
  ])("when refresh answers with %s", (_name, refresh) => {
    it("should keep all three keys, not redirect, and hand back the original 401", async () => {
      signedIn();
      const { otherCalls } = gateway({ refresh, other: unauthorized });

      const response = await sessionFetch(THING_URL);

      expect(response.status).toBe(401);
      expect(await response.json()).toEqual({ code: "UNAUTHENTICATED" });
      expect(localStorage.getItem("accessToken")).toBe("old-access");
      expect(localStorage.getItem("refreshToken")).toBe("old-refresh");
      expect(localStorage.getItem("user")).toBe(JSON.stringify(USER));
      expect(window.location.href).toBe("http://localhost/plans");
      expect(otherCalls).toHaveLength(1);
    });
  });

  function expectSignedOut() {
    expect(localStorage.getItem("accessToken")).toBeNull();
    expect(localStorage.getItem("refreshToken")).toBeNull();
    expect(localStorage.getItem("user")).toBeNull();
    expect(window.location.href).toBe("/auth");
  }

  it("should pass a non-401 from the retry to the caller and keep the new tokens", async () => {
    signedIn();
    gateway({
      refresh: () => authResponse(),
      other: (bearer) =>
        bearer === "new-access" ? json(500, { code: "INTERNAL_ERROR" }) : unauthorized(),
    });

    const response = await sessionFetch(THING_URL);

    expect(response.status).toBe(500);
    expect(localStorage.getItem("accessToken")).toBe("new-access");
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should reject with the fetch error and keep the new tokens when the retry hits a network error", async () => {
    signedIn();
    gateway({
      refresh: () => authResponse(),
      other: (bearer) =>
        bearer === "new-access"
          ? Promise.reject(new TypeError("Failed to fetch"))
          : unauthorized(),
    });

    await expect(sessionFetch(THING_URL)).rejects.toThrow("Failed to fetch");
    expect(localStorage.getItem("accessToken")).toBe("new-access");
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should leave the stored user byte-identical after a refresh", async () => {
    signedIn();
    const stored = '{ "id": "user-1",  "name": "Ada", "extra": true }';
    localStorage.setItem("user", stored);
    gateway({
      refresh: () => authResponse("new-access", "new-refresh", { id: "user-1", name: "Renamed" }),
      other: okIfNewToken,
    });

    await sessionFetch(THING_URL);

    expect(localStorage.getItem("user")).toBe(stored);
  });

  describe("concurrency", () => {
    it("should start a new exchange for a later 401 once the last one settled and failed", async () => {
      signedIn();
      let answer = () => json(503, { code: "SERVICE_UNAVAILABLE" });
      const { refreshCalls } = gateway({
        refresh: () => answer(),
        other: okIfNewToken,
      });

      expect((await sessionFetch(THING_URL)).status).toBe(401);
      answer = () => authResponse();
      expect((await sessionFetch(THING_URL)).status).toBe(200);

      expect(refreshCalls).toHaveLength(2);
    });

    it("should sign out with no refresh when the retry with a replaced token is still 401", async () => {
      signedIn();
      const { refreshCalls, otherCalls } = gateway({
        refresh: () => authResponse(),
        other: (bearer) => {
          if (bearer === "old-access") localStorage.setItem("accessToken", "other-tab");
          return unauthorized();
        },
      });

      const response = await sessionFetch(THING_URL);

      expect(response.status).toBe(401);
      expect(refreshCalls).toHaveLength(0);
      expect(otherCalls).toHaveLength(2);
      expectSignedOut();
    });
  });

  describe("when the user changes during the exchange", () => {
    function exchangeInFlight() {
      signedIn();
      const pending = deferred<Response>();
      const { refreshCalls } = gateway({
        refresh: () => pending.promise,
        other: unauthorized,
      });
      const call = sessionFetch(THING_URL);
      return { pending, call, started: vi.waitFor(() => expect(refreshCalls).toHaveLength(1)) };
    }

    function signInAsSomeoneElse() {
      localStorage.setItem("accessToken", "b-access");
      localStorage.setItem("refreshToken", "b-refresh");
      localStorage.setItem("user", JSON.stringify({ id: "user-2", name: "Bea" }));
    }

    const answers: [string, () => Response][] = [
      ["succeeds", () => authResponse()],
      ["answers 401", unauthorized],
    ];

    it.each(answers)("should leave storage empty and not redirect after sign-out when refresh %s", async (_name, answer) => {
      const { pending, call, started } = exchangeInFlight();
      await started;
      localStorage.clear();
      pending.resolve(answer());

      const response = await call;

      expect(response.status).toBe(401);
      expect(await response.json()).toEqual({ code: "UNAUTHENTICATED" });
      expect(localStorage.length).toBe(0);
      expect(window.location.href).toBe("http://localhost/plans");
    });

    it.each(answers)("should leave the other user's tokens in place when refresh %s", async (_name, answer) => {
      const { pending, call, started } = exchangeInFlight();
      await started;
      signInAsSomeoneElse();
      pending.resolve(answer());

      const response = await call;

      expect(response.status).toBe(401);
      expect(localStorage.getItem("accessToken")).toBe("b-access");
      expect(localStorage.getItem("refreshToken")).toBe("b-refresh");
      expect(JSON.parse(localStorage.getItem("user")!).id).toBe("user-2");
      expect(window.location.href).toBe("http://localhost/plans");
    });

    it("should not redirect when a second 401 lands after sign-out during the exchange", async () => {
      signedIn();
      const pending = deferred<Response>();
      const late = deferred<Response>();
      let sends = 0;
      const { refreshCalls } = gateway({
        refresh: () => pending.promise,
        other: () => (++sends === 1 ? unauthorized() : late.promise),
      });

      const first = sessionFetch(THING_URL);
      const second = sessionFetch(THING_URL);
      await vi.waitFor(() => expect(refreshCalls).toHaveLength(1));
      localStorage.clear();
      late.resolve(unauthorized());
      await flush();
      pending.resolve(authResponse());

      expect((await first).status).toBe(401);
      expect((await second).status).toBe(401);
      expect(refreshCalls).toHaveLength(1);
      expect(localStorage.length).toBe(0);
      expect(window.location.href).toBe("http://localhost/plans");
    });

    it("should discard a result with nothing written when no user is stored", async () => {
      signedIn();
      localStorage.removeItem("user");
      gateway({ refresh: () => authResponse(), other: unauthorized });

      const response = await sessionFetch(THING_URL);

      expect(response.status).toBe(401);
      expect(localStorage.getItem("accessToken")).toBe("old-access");
      expect(localStorage.getItem("refreshToken")).toBe("old-refresh");
      expect(window.location.href).toBe("http://localhost/plans");
    });
  });

  describe("when refresh never answers", () => {
    /** Answers as a stalled connection does: only the request's own signal ends it. */
    const stalled = (request: Request) =>
      new Promise<Response>((_resolve, reject) => {
        request.signal.addEventListener("abort", () => reject(request.signal.reason));
      });

    /** The real signal on a 1ms wait, so the deadline fires without slowing the suite. */
    function shortenDeadline() {
      const realTimeout = AbortSignal.timeout.bind(AbortSignal);
      return vi.spyOn(AbortSignal, "timeout").mockImplementation(() => realTimeout(1));
    }

    it("should give up at the deadline, keep the tokens and hand back the original 401", async () => {
      signedIn();
      const deadline = shortenDeadline();
      gateway({ refresh: stalled, other: unauthorized });

      const response = await sessionFetch(THING_URL);

      expect(deadline).toHaveBeenCalledWith(10_000);
      expect(response.status).toBe(401);
      expect(await response.json()).toEqual({ code: "UNAUTHENTICATED" });
      expect(localStorage.getItem("accessToken")).toBe("old-access");
      expect(localStorage.getItem("refreshToken")).toBe("old-refresh");
      expect(localStorage.getItem("user")).toBe(JSON.stringify(USER));
      expect(window.location.href).toBe("http://localhost/plans");
    });

    it("should let a later 401 start a fresh exchange after the deadline", async () => {
      signedIn();
      shortenDeadline();
      let answer = stalled;
      const { refreshCalls } = gateway({
        refresh: (request) => answer(request),
        other: okIfNewToken,
      });

      expect((await sessionFetch(THING_URL)).status).toBe(401);
      answer = async () => authResponse();
      const response = await sessionFetch(THING_URL);

      expect(response.status).toBe(200);
      expect(refreshCalls).toHaveLength(2);
      expect(localStorage.getItem("accessToken")).toBe("new-access");
    });
  });

  describe("aborting", () => {
    it("should carry the abort signal into the retry", async () => {
      signedIn();
      const controller = new AbortController();
      const signals: AbortSignal[] = [];
      vi.stubGlobal(
        "fetch",
        vi.fn(async (input: Request) => {
          if (input.url === REFRESH_URL) return authResponse();
          signals.push(input.signal);
          return okIfNewToken(input.headers.get("Authorization")?.replace("Bearer ", "") ?? null);
        }),
      );

      await sessionFetch(THING_URL, { signal: controller.signal });
      controller.abort();

      expect(signals).toHaveLength(2);
      expect(signals.every((s) => s.aborted)).toBe(true);
    });
  });
});

/**
 * The same cases through both real paths (R13, R25): the point of one seam is
 * that they cannot drift, so this exercises the wiring rather than the module.
 */
const PROFILE_URL = `${config.apiBaseUrl}/api/users/profile`;
const PROFILE_BODY = { name: "Ada" };

type Outcome = { status: number; body: unknown };
type Path = {
  throwsOnSignOut: boolean;
  call: (signal?: AbortSignal) => Promise<Outcome>;
};

const clientPaths: [string, Path][] = [
  [
    "the generated client",
    {
      throwsOnSignOut: false,
      call: async (signal) => {
        const { data, error, response } = await api.PATCH("/api/users/profile", {
          body: PROFILE_BODY,
          signal,
        });
        return { status: response.status, body: data ?? error };
      },
    },
  ],
  [
    "authFetch",
    {
      throwsOnSignOut: true,
      call: async (signal) => {
        const response = await authFetch(PROFILE_URL, {
          method: "PATCH",
          body: JSON.stringify(PROFILE_BODY),
          headers: { "Content-Type": "application/json" },
          signal,
        });
        return { status: response.status, body: await response.json() };
      },
    },
  ],
];

describe.each(clientPaths)("through %s", (_name, path) => {
  const OK = { status: 200, body: { ok: true } };
  const ORIGINAL_401 = { status: 401, body: { code: "UNAUTHENTICATED" } };

  async function expectSignOut(call: Promise<Outcome>) {
    if (path.throwsOnSignOut) await expect(call).rejects.toThrow("Unauthorized");
    else expect(await call).toEqual(ORIGINAL_401);
    expect(localStorage.getItem("accessToken")).toBeNull();
    expect(localStorage.getItem("refreshToken")).toBeNull();
    expect(localStorage.getItem("user")).toBeNull();
    expect(window.location.href).toBe("/auth");
  }

  function expectKept() {
    expect(localStorage.getItem("accessToken")).toBe("old-access");
    expect(localStorage.getItem("refreshToken")).toBe("old-refresh");
    expect(localStorage.getItem("user")).toBe(JSON.stringify(USER));
    expect(window.location.href).toBe("http://localhost/plans");
  }

  it("should attach the stored bearer and leave a success alone", async () => {
    signedIn();
    const { otherCalls, refreshCalls } = gateway({ other: () => json(200, { ok: true }) });

    expect(await path.call()).toEqual(OK);
    expect(otherCalls.map((c) => c.bearer)).toEqual(["old-access"]);
    expect(refreshCalls).toHaveLength(0);
  });

  it("should renew, resend the identical body, and leave the user byte-identical", async () => {
    signedIn();
    const { otherCalls } = gateway({ refresh: () => authResponse(), other: okIfNewToken });

    expect(await path.call()).toEqual(OK);
    expect(otherCalls).toEqual([
      { method: "PATCH", bearer: "old-access", body: JSON.stringify(PROFILE_BODY) },
      { method: "PATCH", bearer: "new-access", body: JSON.stringify(PROFILE_BODY) },
    ]);
    expect(localStorage.getItem("accessToken")).toBe("new-access");
    expect(localStorage.getItem("refreshToken")).toBe("new-refresh");
    expect(localStorage.getItem("user")).toBe(JSON.stringify(USER));
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should share one refresh across concurrent 401s", async () => {
    signedIn();
    const pending = deferred<Response>();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => pending.promise,
      other: okIfNewToken,
    });

    const all = Promise.all([path.call(), path.call(), path.call()]);
    await vi.waitFor(() => expect(refreshCalls).toHaveLength(1));
    await flush();
    pending.resolve(authResponse());

    expect(await all).toEqual([OK, OK, OK]);
    expect(refreshCalls).toHaveLength(1);
    expect(otherCalls).toHaveLength(6);
  });

  it("should retry with the stored token and no refresh when it was replaced meanwhile", async () => {
    signedIn();
    const { refreshCalls } = gateway({
      refresh: () => authResponse(),
      other: (bearer) => {
        if (bearer === "old-access") localStorage.setItem("accessToken", "new-access");
        return okIfNewToken(bearer);
      },
    });

    expect(await path.call()).toEqual(OK);
    expect(refreshCalls).toHaveLength(0);
  });

  it("should not resend under another user when they signed in while the request was out", async () => {
    signedIn();
    const USER_B = JSON.stringify({ id: "user-2", name: "Bea" });
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: (bearer) => {
        localStorage.setItem("accessToken", "b-access");
        localStorage.setItem("refreshToken", "b-refresh");
        localStorage.setItem("user", USER_B);
        return bearer === "b-access" ? json(200, { ok: true }) : unauthorized();
      },
    });

    expect(await path.call()).toEqual(ORIGINAL_401);
    expect(otherCalls.map((c) => c.bearer)).toEqual(["old-access"]);
    expect(refreshCalls).toHaveLength(0);
    expect(localStorage.getItem("accessToken")).toBe("b-access");
    expect(localStorage.getItem("refreshToken")).toBe("b-refresh");
    expect(localStorage.getItem("user")).toBe(USER_B);
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should return the 401 untouched when the user signed out while the request was out", async () => {
    signedIn();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: () => {
        localStorage.clear();
        return unauthorized();
      },
    });

    expect(await path.call()).toEqual(ORIGINAL_401);
    expect(otherCalls).toHaveLength(1);
    expect(refreshCalls).toHaveLength(0);
    expect(localStorage.length).toBe(0);
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should still retry when only the tokens changed for the same user", async () => {
    signedIn();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: (bearer) => {
        if (bearer === "old-access") {
          localStorage.setItem("accessToken", "new-access");
          localStorage.setItem("refreshToken", "new-refresh");
        }
        return okIfNewToken(bearer);
      },
    });

    expect(await path.call()).toEqual(OK);
    expect(otherCalls.map((c) => c.bearer)).toEqual(["old-access", "new-access"]);
    expect(refreshCalls).toHaveLength(0);
    expect(localStorage.getItem("user")).toBe(JSON.stringify(USER));
  });

  it("should keep the no-bearer sign-out when someone signed in while the request was out", async () => {
    const { refreshCalls, otherCalls } = gateway({
      other: () => {
        localStorage.setItem("accessToken", "b-access");
        localStorage.setItem("refreshToken", "b-refresh");
        localStorage.setItem("user", JSON.stringify({ id: "user-2" }));
        return unauthorized();
      },
    });

    await expectSignOut(path.call());
    expect(otherCalls.map((c) => c.bearer)).toEqual([null]);
    expect(refreshCalls).toHaveLength(0);
  });

  it.each([
    ["a network error", () => Promise.reject(new TypeError("Failed to fetch"))],
    ["a 500", () => json(500, { code: "INTERNAL_ERROR" })],
    ["a 503", () => json(503, { code: "SERVICE_UNAVAILABLE" })],
    ["a 404", () => new Response("404 page not found", { status: 404 })],
  ])("should keep the tokens and return the 401 when refresh fails with %s", async (_n, refresh) => {
    signedIn();
    gateway({ refresh, other: unauthorized });

    expect(await path.call()).toEqual(ORIGINAL_401);
    expectKept();
  });

  it("should sign out when refresh answers 401", async () => {
    signedIn();
    gateway({ refresh: unauthorized, other: unauthorized });

    await expectSignOut(path.call());
  });

  it("should sign out when no refresh token is stored", async () => {
    signedIn();
    localStorage.removeItem("refreshToken");
    const { refreshCalls } = gateway({ other: unauthorized });

    await expectSignOut(path.call());
    expect(refreshCalls).toHaveLength(0);
  });

  it("should sign out with no second refresh or retry when the retry is still 401", async () => {
    signedIn();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: unauthorized,
    });

    await expectSignOut(path.call());
    expect(refreshCalls).toHaveLength(1);
    expect(otherCalls).toHaveLength(2);
  });

  it("should sign out with no refresh call when a 401 had no bearer attached", async () => {
    const { refreshCalls, otherCalls } = gateway({ other: unauthorized });

    await expectSignOut(path.call());
    expect(refreshCalls).toHaveLength(0);
    expect(otherCalls.map((c) => c.bearer)).toEqual([null]);
  });

  it("should discard the result and not redirect when the user signs out during the exchange", async () => {
    signedIn();
    const pending = deferred<Response>();
    const { refreshCalls } = gateway({ refresh: () => pending.promise, other: unauthorized });

    const call = path.call();
    await vi.waitFor(() => expect(refreshCalls).toHaveLength(1));
    localStorage.clear();
    pending.resolve(authResponse());

    expect(await call).toEqual(ORIGINAL_401);
    expect(localStorage.length).toBe(0);
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should leave the other user's tokens when they sign in during the exchange", async () => {
    signedIn();
    const pending = deferred<Response>();
    const { refreshCalls } = gateway({ refresh: () => pending.promise, other: unauthorized });

    const call = path.call();
    await vi.waitFor(() => expect(refreshCalls).toHaveLength(1));
    localStorage.setItem("accessToken", "b-access");
    localStorage.setItem("refreshToken", "b-refresh");
    localStorage.setItem("user", JSON.stringify({ id: "user-2" }));
    pending.resolve(authResponse());

    expect(await call).toEqual(ORIGINAL_401);
    expect(localStorage.getItem("accessToken")).toBe("b-access");
    expect(localStorage.getItem("refreshToken")).toBe("b-refresh");
    expect(window.location.href).toBe("http://localhost/plans");
  });

  it("should not fail other requests when the one that started the exchange aborts", async () => {
    signedIn();
    const pending = deferred<Response>();
    const { refreshCalls } = gateway({ refresh: () => pending.promise, other: okIfNewToken });
    const controller = new AbortController();

    const starter = path.call(controller.signal);
    const sharer = path.call();
    await vi.waitFor(() => expect(refreshCalls).toHaveLength(1));
    await flush();
    controller.abort();
    await expect(starter).rejects.toMatchObject({ name: "AbortError" });
    pending.resolve(authResponse());

    expect(await sharer).toEqual(OK);
    expect(refreshCalls).toHaveLength(1);
    expect(localStorage.getItem("accessToken")).toBe("new-access");
  });
});

/**
 * A stub cannot prove what a real fetch does to a cloned body, so this one sends
 * through the real thing to a real socket. Only refresh is answered in-process:
 * its URL is fixed to the configured gateway, which a test must not reach.
 */
describe("through a real fetch", () => {
  it.each(["POST", "PATCH", "DELETE"])("should put the identical %s body on the wire twice", async (method) => {
    const seen: { method?: string; authorization?: string; contentType?: string; body: string }[] = [];
    const server = createServer((req, res) => {
      const chunks: Buffer[] = [];
      req.on("data", (chunk) => chunks.push(chunk));
      req.on("end", () => {
        seen.push({
          method: req.method,
          authorization: req.headers.authorization,
          contentType: req.headers["content-type"],
          body: Buffer.concat(chunks).toString(),
        });
        res.statusCode = req.headers.authorization === "Bearer new-access" ? 200 : 401;
        res.end(JSON.stringify({ ok: res.statusCode === 200 }));
      });
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const url = `http://127.0.0.1:${(server.address() as AddressInfo).port}/things`;

    try {
      signedIn();
      const realFetch = globalThis.fetch;
      vi.stubGlobal("fetch", (request: Request) =>
        request.url === REFRESH_URL ? Promise.resolve(authResponse()) : realFetch(request),
      );
      const body = JSON.stringify({ description: "keep what I typed", n: "é".repeat(2048) });

      const response = await sessionFetch(url, {
        method,
        body,
        headers: { "Content-Type": "application/json" },
      });

      expect(response.status).toBe(200);
      const common = { method, contentType: "application/json", body };
      expect(seen).toEqual([
        { ...common, authorization: "Bearer old-access" },
        { ...common, authorization: "Bearer new-access" },
      ]);
    } finally {
      // `close` alone waits on fetch's keep-alive socket until the test times out.
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    }
  });
});

/** FS-ZESEF §Edge States, Unauthorized: signin is not special-cased, only the bearer decides. */
describe("a wrong password at sign-in", () => {
  const signIn = () =>
    api.POST("/api/users/signin", { body: { email: "a@example.test", password: "wrong" } });

  it("should make no refresh call and redirect as today when no tokens are stored", async () => {
    const { refreshCalls, otherCalls } = gateway({ other: unauthorized });

    const { response } = await signIn();

    expect(response.status).toBe(401);
    expect(refreshCalls).toHaveLength(0);
    expect(otherCalls.map((c) => c.bearer)).toEqual([null]);
    expect(window.location.href).toBe("/auth");
  });

  it("should refresh once, retry once and land on /auth when stale tokens are stored", async () => {
    signedIn();
    const { refreshCalls, otherCalls } = gateway({
      refresh: () => authResponse(),
      other: unauthorized,
    });

    const { response } = await signIn();

    expect(response.status).toBe(401);
    expect(refreshCalls).toHaveLength(1);
    expect(otherCalls).toHaveLength(2);
    expect(localStorage.getItem("accessToken")).toBeNull();
    expect(window.location.href).toBe("/auth");
  });
});
