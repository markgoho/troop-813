// The HQ live source (spec #17 Modules): Firestore onSnapshot subscriptions
// behind store interfaces, so screens re-render on every change and tests
// pass a writable store instead. The real one is firebaseLive() in firebase.ts.
//
// Add one method per live view (Events, today's Meeting, RSVPs…), each
// returning a Readable of the generic shapes below.
import type { Readable } from "svelte/store";
import type { Scout } from "./contract.gen";

/** A Firestore doc's data with its ID. */
export type WithId<T> = T & { id: string };

/** A live collection: loading until the first snapshot, then its docs, or an error (e.g. rules denied). */
export type LiveDocs<T> =
  | { state: "loading" }
  | { state: "ready"; docs: WithId<T>[] }
  | { state: "error"; error: Error };

export interface Live {
  /** The whole Roster, active and inactive, unordered. */
  scouts(): Readable<LiveDocs<Scout>>;
}
