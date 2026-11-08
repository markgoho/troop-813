// Test helper: a Live whose collections are writable stores the test pushes
// snapshots into. Add each new Live method here once.
import { derived, writable, type Writable } from "svelte/store";
import { Timestamp } from "firebase/firestore";
import type {
  Attendance,
  Event,
  KioskSession,
  Meeting,
  RSVP,
  Scout,
} from "../contract.gen";
import type { Live, LiveDoc, LiveDocs, WithId } from "../live";

export interface FakeLive extends Live {
  /** One store per Meeting ID, created on first use by the test or the screen. */
  meetingStore(id: string): Writable<LiveDoc<Meeting>>;
  /** One store per Meeting ID, like meetingStore. */
  attendanceStore(meetingId: string): Writable<LiveDocs<Attendance>>;
  meetingsStore: Writable<LiveDocs<Meeting>>;
  /** One store per key; kioskSession(id) and currentKioskSession(meetingId) both read it, keyed by the id they are given. */
  kioskSessionStore(key: string): Writable<LiveDoc<KioskSession>>;
  scoutsStore: Writable<LiveDocs<Scout>>;
  eventsStore: Writable<LiveDocs<Event>>;
  rsvpsStore: Writable<LiveDocs<RSVP>>;
}

export function fakeLive(): FakeLive {
  const meetings = new Map<string, Writable<LiveDoc<Meeting>>>();
  const meetingStore = (id: string) => {
    let s = meetings.get(id);
    if (!s) {
      s = writable<LiveDoc<Meeting>>({ state: "loading" });
      meetings.set(id, s);
    }
    return s;
  };
  const attendances = new Map<string, Writable<LiveDocs<Attendance>>>();
  const attendanceStore = (id: string) => {
    let s = attendances.get(id);
    if (!s) {
      s = writable<LiveDocs<Attendance>>({ state: "loading" });
      attendances.set(id, s);
    }
    return s;
  };
  const meetingsStore = writable<LiveDocs<Meeting>>({ state: "loading" });
  const sessions = new Map<string, Writable<LiveDoc<KioskSession>>>();
  const kioskSessionStore = (key: string) => {
    let s = sessions.get(key);
    if (!s) {
      s = writable<LiveDoc<KioskSession>>({ state: "loading" });
      sessions.set(key, s);
    }
    return s;
  };
  const scoutsStore = writable<LiveDocs<Scout>>({ state: "loading" });
  const eventsStore = writable<LiveDocs<Event>>({ state: "loading" });
  const rsvpsStore = writable<LiveDocs<RSVP>>({ state: "loading" });
  return {
    meetingStore,
    attendanceStore,
    attendance: attendanceStore,
    meetingsStore,
    kioskSessionStore,
    kioskSession: kioskSessionStore,
    currentKioskSession: kioskSessionStore,
    scoutsStore,
    eventsStore,
    rsvpsStore,
    meetings: () => meetingsStore,
    meeting: meetingStore,
    scouts: () => scoutsStore,
    events: () => eventsStore,
    rsvps: () => rsvpsStore,
    // The same docs, filtered, like the real query.
    scoutRsvps: (scoutId) =>
      derived(rsvpsStore, ($r) =>
        $r.state === "ready"
          ? { ...$r, docs: $r.docs.filter((d) => d.scoutId === scoutId) }
          : $r,
      ),
  };
}

/** A loaded snapshot of docs, for store.set(ready([...])). */
export function ready<T>(docs: WithId<T>[]): LiveDocs<T> {
  return { state: "ready", docs };
}

/** A Meeting doc snapshot, for meetingStore(id).set(meeting(id, {...})). */
export function meeting(
  id: string,
  fields: Partial<Meeting> = {},
): LiveDoc<Meeting> {
  return {
    state: "ready",
    doc: {
      id,
      startedAt: Timestamp.fromDate(new Date(`${id}T23:00:00Z`)),
      startedBy: "u-admin",
      handoffOrder: [],
      notFound: [],
      checkedIn: {},
      ...fields,
    },
  };
}

/** An Attendance doc for tests; `at` is an ISO instant. */
export function attendanceDoc(
  id: string,
  scoutId: string,
  method: Attendance["method"],
  at: string,
  by = "kiosk-sess-1",
  undoneAt?: string,
): WithId<Attendance> {
  return {
    id,
    scoutId,
    method,
    at: Timestamp.fromDate(new Date(at)),
    by,
    ...(undoneAt && {
      undoneAt: Timestamp.fromDate(new Date(undoneAt)),
      undoneBy: "u-leader",
    }),
  };
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

/**
 * A Kiosk session snapshot, for kioskSessionStore(key).set(kioskSession(...)).
 * `lastActivityAt` is an ISO instant.
 */
export function kioskSession(
  id: string,
  meetingId: string,
  lastActivityAt: string,
): LiveDoc<KioskSession> {
  return {
    state: "ready",
    doc: {
      id,
      meetingId,
      createdAt: Timestamp.fromDate(new Date(lastActivityAt)),
      createdBy: "u-admin",
      lastActivityAt: Timestamp.fromDate(new Date(lastActivityAt)),
      replaced: false,
    },
  };
}
