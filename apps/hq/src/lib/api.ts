// Typed client for the Go API. Components take an Api so tests can pass a fake.
// Body types come from contract.gen.ts (generated from Go, ADR 0002): never hand-write them.
import type { ErrorBody, Health, Me } from "./contract.gen";

export interface Api {
  health(): Promise<Health>;
  /** Who the caller is. The first call after a first sign-in creates their users doc. */
  me(): Promise<Me>;
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

async function getJSON<T>(
  fetchFn: typeof fetch,
  token: TokenSource,
  path: string,
): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const t = await token();
  if (t) headers.Authorization = `Bearer ${t}`;
  const res = await fetchFn(path, { headers });
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorBody | null;
    throw new ApiError(res.status, body);
  }
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
  };
}
