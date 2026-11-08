// Typed client for the Go API. Components take an Api so tests can pass a fake.
// Body types come from contract.gen.ts (generated from Go, ADR 0002): never hand-write them.
import type { ErrorBody, Health } from "./contract.gen";

export interface Api {
  health(): Promise<Health>;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: ErrorBody | null,
  ) {
    super(`API ${status}${body ? `: ${body.code}` : ""}`);
  }
}

async function getJSON<T>(fetchFn: typeof fetch, path: string): Promise<T> {
  const res = await fetchFn(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorBody | null;
    throw new ApiError(res.status, body);
  }
  return (await res.json()) as T;
}

/** The real client: same-origin `/api` (Vite proxies it in dev, Hosting rewrites it in prod). */
export function httpApi(fetchFn: typeof fetch = fetch): Api {
  return {
    health: () => getJSON<Health>(fetchFn, "/api/health"),
  };
}
