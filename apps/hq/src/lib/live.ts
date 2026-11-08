// The HQ live source (spec #17 Modules): Firestore onSnapshot subscriptions
// behind store interfaces, so screens re-render on every change and tests
// pass a writable store instead. The real one is firebaseLive() in firebase.ts.
//
// Add one method per live view (Events, today's Meeting, RSVPs…), each
// returning a Readable of the generic shapes below.
import type { Readable } from "svelte/store";
import type {
  Attendance,
  Event,
  KioskSession,
  Meeting,
  RSVP,
  Scout,
} from "./contract.gen";

/** A Firestore doc's data with its ID. */
export type WithId<T> = T & { id: string };

/** A live collection: loading until the first snapshot, then its docs, or an error (e.g. rules denied). */
export type LiveDocs<T> =
  | { state: "loading" }
  | { state: "ready"; docs: WithId<T>[] }
  | { state: "error"; error: Error };

/** A live single doc: loading, then the doc (null while it doesn't exist), or an error. */
export type LiveDoc<T> =
  | { state: "loading" }
  | { state: "ready"; doc: WithId<T> | null }
  | { state: "error"; error: Error };

export interface Live {
  /** One Meeting doc (meetings/{YYYY-MM-DD}): the Kiosk's live state, and Tonight's. */
  meeting(id: string): Readable<LiveDoc<Meeting>>;
  /** Every Attendance record of a Meeting, undone ones included (Leaders only). */
  attendance(meetingId: string): Readable<LiveDocs<Attendance>>;
  /** One Kiosk session doc: the tablet watches its own to sleep and wake. */
  kioskSession(id: string): Readable<LiveDoc<KioskSession>>;
  /** The live (not replaced) Kiosk session of a Meeting, null while there is none. Leaders only. */
  currentKioskSession(meetingId: string): Readable<LiveDoc<KioskSession>>;
  /** The whole Roster, active and inactive, unordered. */
  scouts(): Readable<LiveDocs<Scout>>;
  /** Every Event, past and upcoming, cancelled or not, unordered. */
  events(): Readable<LiveDocs<Event>>;
  /** Every RSVP doc (a few hundred at troop scale); screens filter by eventId. */
  rsvps(): Readable<LiveDocs<RSVP>>;
  /** One Scout's RSVP docs: the only RSVPs the Kiosk may read, and only for a Scout checked in. */
  scoutRsvps(scoutId: string): Readable<LiveDocs<RSVP>>;
}
