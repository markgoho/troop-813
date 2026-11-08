// `bun run seed`: load fictional Scouts into this worktree's emulators (start
// them with `bun run dev`). There is no production mode: the Go command
// refuses anything but an emulator demo- project.
import { emulatorEnv, loadLocalStack, repoRoot } from "./ports";

if (process.argv.length > 2) {
  console.error("usage: bun run seed (emulators only, takes no arguments)");
  process.exit(2);
}

const env: Record<string, string | undefined> = {
  ...process.env,
  ...emulatorEnv(loadLocalStack()),
};
// Never let an ambient real project win over the emulator project.
delete env.GOOGLE_CLOUD_PROJECT;

const proc = Bun.spawn(["go", "run", "./cmd/seed"], {
  cwd: `${repoRoot}api`,
  env,
  stdio: ["inherit", "inherit", "inherit"],
});
process.exit(await proc.exited);
