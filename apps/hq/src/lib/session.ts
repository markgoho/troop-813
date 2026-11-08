// Who is signed in to HQ, behind an interface so components and tests never
// touch the Firebase SDK. The real one is firebaseSession() in firebase.ts.

/** The Kiosk claims Go put in the custom token at Start (ADR 0003). */
export interface KioskClaims {
  sessionId: string;
  meetingId: string;
}

/** The signed-in Firebase user, as far as HQ needs to know. */
export interface SignedInUser {
  uid: string;
  /** Set when signed in with a Kiosk token (role: kiosk); absent for Google sign-ins. */
  kiosk?: KioskClaims;
}

export interface Session {
  /** Calls cb with the current user (null when signed out) once known, then on every change. Returns an unsubscribe. */
  onChange(cb: (user: SignedInUser | null) => void): () => void;
  /** Starts Google sign-in (a full-page redirect in the real session). */
  signIn(): Promise<void>;
  /**
   * Signs in with a custom token (the Kiosk token Start returns). It replaces
   * whoever was signed in, so the Admin's Google session is gone from HQ.
   */
  signInWithToken(token: string): Promise<void>;
  signOut(): Promise<void>;
  /** A fresh ID token for the API, or null when signed out. */
  idToken(): Promise<string | null>;
}
