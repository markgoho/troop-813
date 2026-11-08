// Test helper: a Session the test drives. signInWithToken(token) signs in as
// the Kiosk user kioskUsers[token]; set(user) simulates any other change.
import { vi } from "vitest";
import type { Session, SignedInUser } from "../session";

export interface FakeSession extends Session {
  set(user: SignedInUser | null): void;
  current(): SignedInUser | null;
}

export function fakeSession(
  initial: SignedInUser | null,
  kioskUsers: Record<string, SignedInUser> = {},
): FakeSession {
  let user = initial;
  const listeners = new Set<(u: SignedInUser | null) => void>();
  const set = (u: SignedInUser | null) => {
    user = u;
    for (const cb of listeners) cb(u);
  };
  return {
    set,
    current: () => user,
    onChange(cb) {
      listeners.add(cb);
      cb(user);
      return () => listeners.delete(cb);
    },
    signIn: vi.fn(async () => {}),
    signInWithToken: vi.fn(async (token: string) => {
      const u = kioskUsers[token];
      if (!u) throw new Error(`fakeSession: unknown token ${token}`);
      set(u);
    }),
    signOut: vi.fn(async () => set(null)),
    idToken: async () => (user ? "token" : null),
  };
}
