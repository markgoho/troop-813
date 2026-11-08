// Firestore security rules against the emulator. Run via `bun run test` from the
// repo root, which boots the emulators and exports FIRESTORE_EMULATOR_HOST.
import { readFileSync } from "node:fs";
import { randomUUID } from "node:crypto";
import {
  assertFails,
  assertSucceeds,
  initializeTestEnvironment,
  type RulesTestEnvironment,
} from "@firebase/rules-unit-testing";
import { collection, doc, getDoc, getDocs, setDoc } from "firebase/firestore";
import { afterAll, beforeAll, describe, expect, test } from "vitest";

const rules = readFileSync(
  new URL("../../../firestore.rules", import.meta.url),
  "utf8",
);

let env: RulesTestEnvironment;

/** Asserts the rules reject op with permission-denied. */
async function expectDenied(op: Promise<unknown>) {
  await expect(assertFails(op)).resolves.toMatchObject({
    code: "permission-denied",
  });
}

/** Asserts the rules allow op. */
async function expectAllowed(op: Promise<unknown>) {
  await expect(assertSucceeds(op)).resolves.toBeDefined();
}

beforeAll(async () => {
  const host = process.env.FIRESTORE_EMULATOR_HOST;
  if (!host) {
    throw new Error(
      "FIRESTORE_EMULATOR_HOST is unset: run rules tests with `bun run test` from the repo root",
    );
  }
  const base = process.env.GCLOUD_PROJECT ?? "demo-troop-813";
  env = await initializeTestEnvironment({
    // A project per test file keeps rules and docs isolated from other suites.
    projectId: `${base}-rules-${randomUUID().slice(0, 8)}`,
    firestore: { rules },
  });
  await env.withSecurityRulesDisabled(async (ctx) => {
    const db = ctx.firestore();
    for (const [uid, role] of [
      ["u-parent", "parent"],
      ["u-leader", "leader"],
      ["u-admin", "admin"],
    ]) {
      await setDoc(doc(db, `users/${uid}`), {
        email: `${uid}@example.com`,
        displayName: uid,
        role,
      });
    }
    await setDoc(doc(db, "scouts/s1"), { firstName: "Avery", lastName: "Q" });
    await setDoc(doc(db, "meetings/2026-11-09"), { startedBy: "u-admin" });
  });
});

afterAll(async () => {
  await env?.cleanup();
});

describe("users docs", () => {
  test("a user can read their own users doc", async () => {
    const db = env.authenticatedContext("u-parent").firestore();
    await expectAllowed(getDoc(doc(db, "users/u-parent")));
  });

  test("a user can't read another user's doc", async () => {
    const db = env.authenticatedContext("u-leader").firestore();
    await expectDenied(getDoc(doc(db, "users/u-parent")));
  });

  test("anonymous users can't read users docs", async () => {
    const db = env.unauthenticatedContext().firestore();
    await expectDenied(getDoc(doc(db, "users/u-parent")));
  });
});

describe("Leaders read the Roster", () => {
  for (const uid of ["u-leader", "u-admin"]) {
    test(`${uid} can get a Scout and list scouts (onSnapshot)`, async () => {
      const db = env.authenticatedContext(uid).firestore();
      await expectAllowed(getDoc(doc(db, "scouts/s1")));
      await expectAllowed(getDocs(collection(db, "scouts")));
    });
  }
});

describe("Parents and anonymous users read nothing else", () => {
  const identities = {
    anonymous: () => env.unauthenticatedContext(),
    parent: () => env.authenticatedContext("u-parent"),
    // Signed in, but no users doc yet (Go creates it on the first API call).
    "first sign-in": () => env.authenticatedContext("u-new"),
  };
  for (const [name, context] of Object.entries(identities)) {
    test(`${name} can't read scouts or meetings`, async () => {
      const db = context().firestore();
      await expectDenied(getDoc(doc(db, "scouts/s1")));
      await expectDenied(getDocs(collection(db, "scouts")));
      await expectDenied(getDoc(doc(db, "meetings/2026-11-09")));
    });
  }
});

describe("writes are denied for every identity", () => {
  const identities = {
    anonymous: () => env.unauthenticatedContext(),
    parent: () => env.authenticatedContext("u-parent"),
    leader: () => env.authenticatedContext("u-leader"),
    kiosk: () =>
      env.authenticatedContext("kiosk-test", {
        role: "kiosk",
        kioskSessionId: "session-1",
        meetingId: "2026-11-09",
      }),
  };

  for (const [name, context] of Object.entries(identities)) {
    test(`${name} cannot write`, async () => {
      const db = context().firestore();
      await expectDenied(
        setDoc(doc(db, "meetings/2026-11-09"), { startedBy: "x" }),
      );
      await expectDenied(setDoc(doc(db, "scouts/s1"), { firstName: "Avery" }));
      // Not even their own users doc: only Go sets roles.
      await expectDenied(
        setDoc(doc(db, "users/u-parent"), { role: "admin" }, { merge: true }),
      );
      await expectDenied(
        setDoc(doc(db, "users/u-leader"), { role: "admin" }, { merge: true }),
      );
    });
  }
});
