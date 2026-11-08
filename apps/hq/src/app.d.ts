// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
  namespace App {}

  // Build-time config read by src/lib/firebase.ts.
  interface ImportMetaEnv {
    readonly VITE_FIREBASE_API_KEY?: string;
    readonly VITE_FIREBASE_APP_ID?: string;
    readonly VITE_FIREBASE_PROJECT_ID?: string;
    readonly VITE_FIREBASE_AUTH_EMULATOR_URL?: string;
    readonly VITE_FIRESTORE_EMULATOR_HOST?: string;
  }
}

export {};
