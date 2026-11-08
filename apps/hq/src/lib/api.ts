// Typed client for the Go API. Components take an Api so tests can pass a fake.
// Body types come from contract.gen.ts (generated from Go, ADR 0002): never hand-write them.
import type {
  AddScoutRequest,
  AddScoutResponse,
  ErrorBody,
  Health,
  Me,
  SetRole,
  UserList,
} from "./contract.gen";

export interface Api {
  health(): Promise<Health>;
  /** Who the caller is. The first call after a first sign-in creates their users doc. */
  me(): Promise<Me>;
  /** Admin only: everyone who has signed in, newest first. */
  users(): Promise<UserList>;
  /** Admin only: Parent↔Leader. Applies on that person's next request. */
  setRole(uid: string, role: "parent" | "leader"): Promise<void>;
  /** Leader only: adds an active Scout. ApiError 409 duplicate_scout for an active exact match. */
  addScout(scout: AddScoutRequest): Promise<AddScoutResponse>;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: ErrorBody | null,
  ) {
    super(`API ${status}${body ? `: ${body.code}` : ""}`);
  }
}

/** Returns the current ID token, or null when signed out (then requests go without one). */
export type TokenSource = () => Promise<string | null>;

async function send(
  fetchFn: typeof fetch,
  token: TokenSource,
  method: string,
  path: string,
  body?: unknown,
): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const t = await token();
  if (t) headers.Authorization = `Bearer ${t}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const res = await fetchFn(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => null)) as ErrorBody | null;
    throw new ApiError(res.status, err);
  }
  return res;
}

async function getJSON<T>(
  fetchFn: typeof fetch,
  token: TokenSource,
  path: string,
): Promise<T> {
  const res = await send(fetchFn, token, "GET", path);
  return (await res.json()) as T;
}

/** The real client: same-origin `/api` (Vite proxies it in dev, Hosting rewrites it in prod). */
export function httpApi(
  fetchFn: typeof fetch = fetch,
  token: TokenSource = async () => null,
): Api {
  return {
    health: () => getJSON<Health>(fetchFn, token, "/api/health"),
    me: () => getJSON<Me>(fetchFn, token, "/api/me"),
    users: () => getJSON<UserList>(fetchFn, token, "/api/users"),
    setRole: async (uid, role) => {
      const body: SetRole = { role };
      await send(
        fetchFn,
        token,
        "PUT",
        `/api/users/${encodeURIComponent(uid)}/role`,
        body,
      );
    },
    addScout: async (scout) => {
      const res = await send(fetchFn, token, "POST", "/api/scouts", scout);
      return (await res.json()) as AddScoutResponse;
    },
  };
}
