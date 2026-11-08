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
  onAuthStateChanged,
  signInWithRedirect,
  signOut,
} from "firebase/auth";
import {
  type Firestore,
  connectFirestoreEmulator,
  getFirestore,
} from "firebase/firestore";
import type { Session } from "./session";

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
    const url = import.meta.env.VITE_FIREBASE_AUTH_EMULATOR_URL;
    if (url) connectAuthEmulator(auth, url, { disableWarnings: true });
  }
  return auth;
}

/** Firestore for onSnapshot reads (rules allow reads only; Go does every write). */
export function firestore(): Firestore {
  if (!db) {
    db = getFirestore(firebaseApp());
    const host = import.meta.env.VITE_FIRESTORE_EMULATOR_HOST;
    if (host) {
      const [hostname = "127.0.0.1", port = "8080"] = host.split(":");
      connectFirestoreEmulator(db, hostname, Number(port));
    }
  }
  return db;
}

/** The real Session: Google sign-in by redirect. */
export function firebaseSession(): Session {
  const a = firebaseAuth();
  return {
    onChange: (cb) =>
      onAuthStateChanged(a, (u) => cb(u ? { uid: u.uid } : null)),
    signIn: () => signInWithRedirect(a, new GoogleAuthProvider()),
    signOut: () => signOut(a),
    idToken: async () => (a.currentUser ? a.currentUser.getIdToken() : null),
  };
}
