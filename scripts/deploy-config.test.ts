// Prod names are written in several files that nothing else checks against
// each other: firebase.json, .firebaserc, terraform/*.tf, terraform/bootstrap.sh
// and .github/workflows/*.yml. A mismatch only shows up as a failed deploy on
// trunk, so check here that they agree.
import { describe, expect, test } from "bun:test";
import { repoRoot } from "./ports";

const text = (path: string) => Bun.file(repoRoot + path).text();
const json = async (path: string) => JSON.parse(await text(path));
const yaml = async (path: string) =>
  Bun.YAML.parse(await text(path)) as Workflow;

interface Step {
  uses?: string;
  run?: string;
  with?: Record<string, string>;
}
interface Workflow {
  on: { push: { branches: string[]; paths: string[] } };
  concurrency: { group: string; "cancel-in-progress": boolean };
  env: Record<string, string>;
  jobs: Record<string, { steps: Step[]; permissions: Record<string, string> }>;
}

const SERVICE = "api";
const REGION = "us-east1";
const SITE = "troop-813-hq";
const WIF_PROVIDER =
  "projects/1003933813667/locations/global/workloadIdentityPools/github/providers/github";

const tf = async () => {
  const glob = new Bun.Glob("terraform/*.tf");
  let all = "";
  for await (const f of glob.scan(repoRoot)) all += await text(f);
  return all;
};

describe("Hosting", () => {
  test("serves the HQ build on the troop-813-hq site, /api/** going to Cloud Run first", async () => {
    const config = await json("firebase.json");
    const [hq] = config.hosting;
    expect(hq.target).toBe("hq");
    expect(hq.public).toBe("apps/hq/build");
    expect(hq.rewrites[0]).toEqual({
      source: "/api/**",
      run: { serviceId: SERVICE, region: REGION },
    });
    expect(hq.rewrites.at(-1)).toEqual({
      source: "**",
      destination: "/200.html",
    });

    const rc = await json(".firebaserc");
    expect(rc.projects.default).toBe("troop-813");
    expect(rc.targets["troop-813"].hosting.hq).toEqual([SITE]);
  });

  test("Terraform owns that site and lists it as an authorized domain", async () => {
    const all = await tf();
    expect(all).toContain(`site_id  = "${SITE}"`);
    expect(all).toContain(
      `"${"$"}{google_firebase_hosting_site.hq.site_id}.web.app"`,
    );
  });

  test("web.yml builds HQ with the Firebase web config of Terraform's HQ app", async () => {
    const all = await tf();
    expect(all).toMatch(/display_name\s+= "HQ"/);
    const source = await text(".github/workflows/web.yml");
    expect(source).toContain(`select(.displayName == "HQ")`);
    // Every VITE_FIREBASE_* the client reads (apps/hq/src/lib/firebase.ts),
    // except the emulator-only ones.
    const client = await text("apps/hq/src/lib/firebase.ts");
    const vars = new Set(
      [...client.matchAll(/env\.(VITE_FIREBASE_\w+)/g)].map((m) => m[1]),
    );
    vars.delete("VITE_FIREBASE_AUTH_EMULATOR_URL");
    expect(vars.size).toBeGreaterThan(0);
    for (const v of vars) expect(source).toContain(`${v}=`);
  });
});

describe("Cloud Run", () => {
  test("Terraform's service is the one api.yml deploys to", async () => {
    const all = await tf();
    expect(all).toMatch(new RegExp(`name\\s+= "${SERVICE}"`));
    expect(all).toMatch(new RegExp(`region\\s+= "${REGION}"`));
    const api = await yaml(".github/workflows/api.yml");
    expect(api.env.SERVICE).toBe(SERVICE);
    expect(api.env.REGION).toBe(REGION);
  });
});

describe("workflows", () => {
  const files = ["api.yml", "web.yml", "terraform.yml"];

  test.each(files)(
    "%s runs on trunk only, queued and never cancelled, one job",
    async (file) => {
      const wf = await yaml(`.github/workflows/${file}`);
      expect(wf.on.push.branches).toEqual(["trunk"]);
      expect(wf.on.push.paths).toContain(`.github/workflows/${file}`);
      expect(wf.concurrency["cancel-in-progress"]).toBe(false);
      expect(Object.keys(wf.jobs)).toHaveLength(1);
    },
  );

  test.each(files)(
    "%s authenticates with WIF from bootstrap.sh, never a key",
    async (file) => {
      const source = await text(`.github/workflows/${file}`);
      const wf = await yaml(`.github/workflows/${file}`);
      const job = Object.values(wf.jobs)[0]!;
      expect(job.permissions["id-token"]).toBe("write");
      const auth = job.steps.find((s) =>
        s.uses?.startsWith("google-github-actions/auth@"),
      );
      expect(auth?.with?.workload_identity_provider).toBe(WIF_PROVIDER);
      expect(source).not.toMatch(/credentials_json|secrets\./);

      const bootstrap = await text("terraform/bootstrap.sh");
      expect(bootstrap).toContain("POOL=github");
      expect(bootstrap).toContain("PROVIDER=github");
    },
  );

  test("api.yml runs when the tygo output changes", async () => {
    const wf = await yaml(".github/workflows/api.yml");
    expect(wf.on.push.paths).toContain("apps/hq/src/lib/contract.gen.ts");
  });

  test("the deploy SA the workflows use is the one Terraform creates", async () => {
    const all = await tf();
    expect(all).toContain(`account_id   = "deploy"`);
    for (const file of ["api.yml", "web.yml"]) {
      const wf = await yaml(`.github/workflows/${file}`);
      const auth = Object.values(wf.jobs)[0]!.steps.find((s) =>
        s.uses?.startsWith("google-github-actions/auth@"),
      );
      expect(auth?.with?.service_account).toBe(
        "deploy@troop-813.iam.gserviceaccount.com",
      );
    }
  });
});
