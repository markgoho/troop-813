// `bun run bootstrap-admin <email> [--prod]`: make one signed-in user the Admin
// (ADR 0003: the only way `admin` is ever set). The user must have signed in to
// HQ once.
//
// Default target: this worktree's emulators (start them with `bun run dev`).
// --prod: the real troop-813 project, using gcloud application-default credentials.
import { emulatorEnv, loadLocalStack, repoRoot } from "./ports";

const PROD_PROJECT_ID = "troop-813";

const args = process.argv.slice(2);
const prod = args.includes("--prod");
const [email, ...rest] = args.filter((a) => a !== "--prod");
if (!email || rest.length > 0) {
  console.error("usage: bun run bootstrap-admin <email> [--prod]");
  process.exit(2);
}

const env: Record<string, string | undefined> = { ...process.env };
if (prod) {
  delete env.FIRESTORE_EMULATOR_HOST;
  delete env.FIREBASE_AUTH_EMULATOR_HOST;
  env.GOOGLE_CLOUD_PROJECT = PROD_PROJECT_ID;
  delete env.GCLOUD_PROJECT;
} else {
  Object.assign(env, emulatorEnv(loadLocalStack()));
  delete env.GOOGLE_CLOUD_PROJECT;
}

const proc = Bun.spawn(["go", "run", "./cmd/bootstrap-admin", email], {
  cwd: `${repoRoot}api`,
  env,
  stdio: ["inherit", "inherit", "inherit"],
});
process.exit(await proc.exited);
