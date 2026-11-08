// Test helper: an Api whose every method fails loudly unless the test overrides it.
// Add each new Api method here once, instead of in every test's fake.
import type { Api } from "../api";

function unexpected(name: string) {
  return () => Promise.reject(new Error(`fakeApi: unexpected call to ${name}`));
}

export function fakeApi(overrides: Partial<Api> = {}): Api {
  return {
    health: unexpected("health"),
    me: unexpected("me"),
    users: unexpected("users"),
    setRole: unexpected("setRole"),
    createEvent: unexpected("createEvent"),
    updateEvent: unexpected("updateEvent"),
    setEventCancelled: unexpected("setEventCancelled"),
    deleteEvent: unexpected("deleteEvent"),
    setRsvp: unexpected("setRsvp"),
    addScout: unexpected("addScout"),
    ...overrides,
  };
}
