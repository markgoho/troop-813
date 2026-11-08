// terraform/bootstrap.sh, driven through a fake `gcloud` on PATH. The fake
// logs every call and answers `describe` from a list of resources that
// "already exist", so the tests see exactly what the script would do to the
// real project.
import { afterEach, beforeEach, describe, expect, test } from "bun:test";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { repoRoot } from "./ports";

const script = `${repoRoot}terraform/bootstrap.sh`;

// Resource keys the fake recognises in `describe` calls.
type Existing = "project" | "billing" | "bucket" | "pool" | "provider" | "sa";

const fakeGcloud = `#!/usr/bin/env bash
printf '%s\\n' "$*" >> "$FAKE_LOG"
has() { [[ " $FAKE_EXISTING " == *" $1 "* ]]; }
case "$*" in
  "projects describe"*"value(projectNumber)"*) has project && { echo 1003933813667; exit 0; } ; exit 1 ;;
  "billing projects describe"*) has billing && echo True || echo False ;;
  "storage buckets describe"*) has bucket || exit 1 ;;
  "iam workload-identity-pools providers describe"*) has provider || exit 1 ;;
  "iam workload-identity-pools describe"*) has pool || exit 1 ;;
  "iam service-accounts describe"*) has sa || exit 1 ;;
esac
exit 0
`;

let dir: string;
beforeEach(async () => {
  dir = await mkdtemp(join(tmpdir(), "bootstrap-test-"));
  await Bun.write(join(dir, "gcloud"), fakeGcloud);
  await Bun.$`chmod +x ${join(dir, "gcloud")}`;
});
afterEach(async () => {
  await rm(dir, { recursive: true, force: true });
});

async function bootstrap(existing: Existing[]) {
  const log = join(dir, "calls.log");
  const proc = Bun.spawn(["bash", script], {
    env: {
      PATH: `${dir}:${process.env.PATH}`,
      FAKE_LOG: log,
      FAKE_EXISTING: existing.join(" "),
    },
    stdout: "pipe",
    stderr: "pipe",
  });
  const code = await proc.exited;
  const file = Bun.file(log);
  const calls = (await file.exists())
    ? (await file.text()).trim().split("\n")
    : [];
  const output =
    (await new Response(proc.stdout).text()) +
    (await new Response(proc.stderr).text());
  return { code, calls, output };
}

const mutating = (calls: string[]) =>
  calls.filter((c) => !/ (describe|list)( |$)/.test(` ${c}`));

describe("bootstrap.sh", () => {
  test("refuses, changing nothing, when the troop-813 project does not exist", async () => {
    const { code, calls, output } = await bootstrap([]);
    expect(code).not.toBe(0);
    expect(output).toContain("troop-813");
    expect(mutating(calls)).toEqual([]);
  });

  test("refuses, changing nothing, until billing is linked", async () => {
    const { code, calls, output } = await bootstrap(["project"]);
    expect(code).not.toBe(0);
    expect(output.toLowerCase()).toContain("billing");
    expect(mutating(calls)).toEqual([]);
  });

  test("on a fresh project creates the versioned us-east1 state bucket", async () => {
    const { code, calls } = await bootstrap(["project", "billing"]);
    expect(code).toBe(0);
    const create = calls.find((c) => c.startsWith("storage buckets create"));
    expect(create).toContain("gs://troop-813-tfstate");
    expect(create).toContain("--location=us-east1");
    expect(create).toContain("--uniform-bucket-level-access");
    expect(calls).toContainEqual(
      expect.stringMatching(
        /^storage buckets update gs:\/\/troop-813-tfstate .*--versioning/,
      ),
    );
  });

  test("creates a WIF provider that only trusts trunk on markgoho/troop-813", async () => {
    const { calls } = await bootstrap(["project", "billing"]);
    expect(calls).toContainEqual(
      expect.stringMatching(/^iam workload-identity-pools create github /),
    );
    const provider = calls.find((c) =>
      c.startsWith("iam workload-identity-pools providers create-oidc"),
    );
    expect(provider).toContain(
      "--issuer-uri=https://token.actions.githubusercontent.com",
    );
    expect(provider).toContain("assertion.repository=='markgoho/troop-813'");
    expect(provider).toContain("assertion.repository_id=='1448981854'");
    expect(provider).toContain("assertion.ref=='refs/heads/trunk'");
  });

  test("creates the Terraform SA and lets only this repo's WIF principals impersonate it", async () => {
    const { calls } = await bootstrap(["project", "billing"]);
    expect(calls).toContainEqual(
      expect.stringMatching(/^iam service-accounts create terraform /),
    );
    const wif = calls.find(
      (c) =>
        c.startsWith(
          "iam service-accounts add-iam-policy-binding terraform@troop-813.iam.gserviceaccount.com",
        ) && c.includes("roles/iam.workloadIdentityUser"),
    );
    expect(wif).toContain(
      "principalSet://iam.googleapis.com/projects/1003933813667/locations/global/workloadIdentityPools/github/attribute.repository/markgoho/troop-813",
    );
    expect(calls).toContainEqual(
      expect.stringMatching(
        /^projects add-iam-policy-binding troop-813 .*serviceAccount:terraform@troop-813\.iam\.gserviceaccount\.com.*roles\/owner/,
      ),
    );
  });

  test("never creates or deletes a project", async () => {
    const { calls } = await bootstrap(["project", "billing"]);
    expect(calls.filter((c) => /^projects (create|delete)/.test(c))).toEqual(
      [],
    );
  });

  test("a re-run against a bootstrapped project creates nothing", async () => {
    const { code, calls } = await bootstrap([
      "project",
      "billing",
      "bucket",
      "pool",
      "provider",
      "sa",
    ]);
    expect(code).toBe(0);
    expect(calls.filter((c) => / create(-oidc)? /.test(` ${c} `))).toEqual([]);
    // ...but still converges the provider's trust condition.
    expect(calls).toContainEqual(
      expect.stringMatching(
        /^iam workload-identity-pools providers update-oidc github .*refs\/heads\/trunk/,
      ),
    );
  });
});
