// The Firebase client SDK for HQ: the real Session and (for live views) Firestore.
// Only this file imports firebase/*; components get a Session or an Api.
//
// Config is build-time env (Vite exposes VITE_* from the environment or .env):
//   VITE_FIREBASE_API_KEY, VITE_FIREBASE_APP_ID, VITE_FIREBASE_PROJECT_ID
//     the Firebase web app config. `bun run dev` sets demo values; the web
//     deploy sets troop-813's.
//   VITE_FIREBASE_AUTH_EMULATOR_URL, VITE_FIRESTORE_EMULATOR_HOST
//     when set (only by `bun run dev`), point the SDK at the emulators.
// authDomain is always the app's own origin, so the redirect sign-in goes
// through Hosting's /__/auth/handler on the same site (no third-party cookies).
import { type FirebaseApp, initializeApp } from "firebase/app";
import {
  type Auth,
  GoogleAuthProvider,
  connectAuthEmulator,
  getAuth,
  type User,
  onAuthStateChanged,
  signInWithCustomToken,
  signInWithRedirect,
  signOut,
} from "firebase/auth";
import {
  type Firestore,
  collection,
  connectFirestoreEmulator,
  doc,
  getFirestore,
  onSnapshot,
  query,
  where,
} from "firebase/firestore";
import { type Readable, readable } from "svelte/store";
import type {
  Attendance,
  Event,
  KioskSession,
  Meeting,
  RSVP,
  Scout,
} from "./contract.gen";
import type { Live, LiveDoc, LiveDocs } from "./live";
import type { Session, SignedInUser } from "./session";

let app: FirebaseApp | undefined;
let auth: Auth | undefined;
let db: Firestore | undefined;

function firebaseApp(): FirebaseApp {
  const env = import.meta.env;
  app ??= initializeApp({
    apiKey: env.VITE_FIREBASE_API_KEY,
    appId: env.VITE_FIREBASE_APP_ID,
    projectId: env.VITE_FIREBASE_PROJECT_ID,
    authDomain: location.host,
  });
  return app;
}

function firebaseAuth(): Auth {
  if (!auth) {
    auth = getAuth(firebaseApp());
    // "same-origin" (`bun run dev:lan`): the dev server proxies the emulator.
    const env = import.meta.env.VITE_FIREBASE_AUTH_EMULATOR_URL;
    const url = env === "same-origin" ? location.origin : env;
    if (url) connectAuthEmulator(auth, url, { disableWarnings: true });
  }
  return auth;
}

/** Firestore for onSnapshot reads (rules allow reads only; Go does every write). */
export function firestore(): Firestore {
  if (!db) {
    db = getFirestore(firebaseApp());
    const env = import.meta.env.VITE_FIRESTORE_EMULATOR_HOST;
    const host = env === "same-origin" ? location.host : env;
    if (host) {
      const [hostname = "127.0.0.1", port = "8080"] = host.split(":");
      connectFirestoreEmulator(db, hostname, Number(port));
    }
  }
  return db;
}

/**
 * A collection as a store, through onSnapshot. The listener starts with the
 * first subscriber and stops with the last, so a screen only listens while shown.
 * T is the doc's contract type; rules decide who may read (Go writes).
 */
export function liveCollection<T>(path: string): Readable<LiveDocs<T>> {
  return readable<LiveDocs<T>>({ state: "loading" }, (set) =>
    onSnapshot(
      collection(firestore(), path),
      (snap) =>
        set({
          state: "ready",
          docs: snap.docs.map((d) => ({ ...(d.data() as T), id: d.id })),
        }),
      (error) => set({ state: "error", error }),
    ),
  );
}

/** Docs whose `field` equals `value`, as a store (same lifetime as liveCollection). */
export function liveWhere<T>(
  path: string,
  field: string,
  value: string,
): Readable<LiveDocs<T>> {
  return readable<LiveDocs<T>>({ state: "loading" }, (set) =>
    onSnapshot(
      query(collection(firestore(), path), where(field, "==", value)),
      (snap) =>
        set({
          state: "ready",
          docs: snap.docs.map((d) => ({ ...(d.data() as T), id: d.id })),
        }),
      (error) => set({ state: "error", error }),
    ),
  );
}

/**
 * One doc as a store, through onSnapshot (same lifetime as liveCollection).
 * A doc that doesn't exist (yet) is ready with doc: null.
 */
export function liveDoc<T>(path: string): Readable<LiveDoc<T>> {
  return readable<LiveDoc<T>>({ state: "loading" }, (set) =>
    onSnapshot(
      doc(firestore(), path),
      (snap) =>
        set({
          state: "ready",
          doc: snap.exists() ? { ...(snap.data() as T), id: snap.id } : null,
        }),
      (error) => set({ state: "error", error }),
    ),
  );
}

/**
 * The live (not replaced) Kiosk session of a Meeting, as a doc store: a query
 * with two equality filters (no composite index), at most one match.
 */
function liveCurrentKioskSession(
  meetingId: string,
): Readable<LiveDoc<KioskSession>> {
  return readable<LiveDoc<KioskSession>>({ state: "loading" }, (set) =>
    onSnapshot(
      query(
        collection(firestore(), "kioskSessions"),
        where("meetingId", "==", meetingId),
        where("replaced", "==", false),
      ),
      (snap) => {
        const d = snap.docs[0];
        set({
          state: "ready",
          doc: d ? { ...(d.data() as KioskSession), id: d.id } : null,
        });
      },
      (error) => set({ state: "error", error }),
    ),
  );
}

/** The real live source. */
export function firebaseLive(): Live {
  return {
    meeting: (id) => liveDoc<Meeting>(`meetings/${id}`),
    meetings: () => liveCollection<Meeting>("meetings"),
    attendance: (id) => liveCollection<Attendance>(`meetings/${id}/attendance`),
    kioskSession: (id) => liveDoc<KioskSession>(`kioskSessions/${id}`),
    currentKioskSession: liveCurrentKioskSession,
    scouts: () => liveCollection<Scout>("scouts"),
    events: () => liveCollection<Event>("events"),
    rsvps: () => liveCollection<RSVP>("rsvps"),
    scoutRsvps: (scoutId) => liveWhere<RSVP>("rsvps", "scoutId", scoutId),
  };
}

/** Reads the Kiosk claims (role: kiosk) off the user's ID token, if any. */
async function signedInUser(u: User): Promise<SignedInUser> {
  const { claims } = await u.getIdTokenResult();
  if (
    claims.role === "kiosk" &&
    typeof claims.kioskSessionId === "string" &&
    typeof claims.meetingId === "string"
  ) {
    return {
      uid: u.uid,
      kiosk: { sessionId: claims.kioskSessionId, meetingId: claims.meetingId },
    };
  }
  return { uid: u.uid };
}

/** The real Session: Google sign-in by redirect, or a Kiosk custom token. */
export function firebaseSession(): Session {
  const a = firebaseAuth();
  return {
    onChange: (cb) => {
      // Claims resolve asynchronously; only the latest auth change may report.
      let latest = 0;
      return onAuthStateChanged(a, (u) => {
        const seq = ++latest;
        if (!u) {
          cb(null);
          return;
        }
        signedInUser(u).then(
          (user) => seq === latest && cb(user),
          // Claims unreadable (e.g. offline token refresh): report the bare
          // uid. Screens then ask the API, which fails visibly; never treat
          // it as signed out, which would offer Google sign-in on the Kiosk.
          () => seq === latest && cb({ uid: u.uid }),
        );
      });
    },
    signIn: () => signInWithRedirect(a, new GoogleAuthProvider()),
    signInWithToken: async (token) => {
      await signInWithCustomToken(a, token);
    },
    signOut: () => signOut(a),
    idToken: async () => (a.currentUser ? a.currentUser.getIdToken() : null),
  };
}
