// Firestore security rules against the emulator. Run via `bun run test` from the
// repo root, which boots the emulators and exports FIRESTORE_EMULATOR_HOST.
import { readFileSync } from "node:fs";
import { randomUUID } from "node:crypto";
import {
  assertFails,
  initializeTestEnvironment,
  type RulesTestEnvironment,
} from "@firebase/rules-unit-testing";
import { doc, setDoc } from "firebase/firestore";
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
});

afterAll(async () => {
  await env?.cleanup();
});

describe("writes are denied for every identity", () => {
  const identities = {
    anonymous: () => env.unauthenticatedContext(),
    parent: () => env.authenticatedContext("parent-test"),
    leader: () => env.authenticatedContext("leader-test"),
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
    });
  }
});
