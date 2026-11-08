// Test helper: a Live whose collections are writable stores the test pushes
// snapshots into. Add each new Live method here once.
import { writable, type Writable } from "svelte/store";
import { Timestamp } from "firebase/firestore";
import type { Event, RSVP, Scout } from "../contract.gen";
import type { Live, LiveDocs, WithId } from "../live";

export interface FakeLive extends Live {
  scoutsStore: Writable<LiveDocs<Scout>>;
  eventsStore: Writable<LiveDocs<Event>>;
  rsvpsStore: Writable<LiveDocs<RSVP>>;
}

export function fakeLive(): FakeLive {
  const scoutsStore = writable<LiveDocs<Scout>>({ state: "loading" });
  const eventsStore = writable<LiveDocs<Event>>({ state: "loading" });
  const rsvpsStore = writable<LiveDocs<RSVP>>({ state: "loading" });
  return {
    scoutsStore,
    eventsStore,
    rsvpsStore,
    scouts: () => scoutsStore,
    events: () => eventsStore,
    rsvps: () => rsvpsStore,
  };
}

/** A loaded snapshot of docs, for store.set(ready([...])). */
export function ready<T>(docs: WithId<T>[]): LiveDocs<T> {
  return { state: "ready", docs };
}

/** A Scout doc for tests. */
export function scout(
  id: string,
  firstName: string,
  lastName: string,
  active = true,
): WithId<Scout> {
  return {
    id,
    firstName,
    lastName,
    active,
    createdAt: Timestamp.fromDate(new Date("2026-11-01T00:00:00Z")),
  };
}
