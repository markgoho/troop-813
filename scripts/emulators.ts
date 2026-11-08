// Runs the Firestore + Auth emulators on this worktree's ports (see scripts/ports.ts).
//
//   bun scripts/emulators.ts exec "<command>"   boot, run <command> with the emulator env, shut down
//   bun scripts/emulators.ts start              boot and keep running (used by `bun run dev`)
//
// firebase-tools has no CLI flag for emulator ports, so this writes a gitignored
// `.firebase.local-<offset>.json` (firebase.json with the ports shifted; one per
// offset so concurrent runs in one checkout do not clobber each other) and passes --config.
import {
  emulatorConfig,
  type FirebaseConfig,
  type LocalStack,
  loadLocalStack,
  repoRoot,
} from "./ports";

export async function writeLocalConfig(stack: LocalStack): Promise<string> {
  const committed = (await Bun.file(
    `${repoRoot}firebase.json`,
  ).json()) as FirebaseConfig;
  const path = `${repoRoot}.firebase.local-${stack.offset}.json`;
  await Bun.write(
    path,
    `${JSON.stringify(emulatorConfig(committed, stack.ports), null, 2)}\n`,
  );
  return path;
}

export function firebaseArgs(
  stack: LocalStack,
  configPath: string,
  mode: "exec" | "start",
  command?: string,
): string[] {
  const args = [
    `${repoRoot}node_modules/.bin/firebase`,
    mode === "exec" ? "emulators:exec" : "emulators:start",
    "--config",
    configPath,
    "--only",
    "firestore,auth",
    "--project",
    stack.projectId,
  ];
  if (command !== undefined) args.push(command);
  return args;
}

if (import.meta.main) {
  const [mode, command] = process.argv.slice(2);
  if (mode !== "exec" && mode !== "start") {
    console.error('usage: bun scripts/emulators.ts exec "<command>" | start');
    process.exit(2);
  }
  if (mode === "exec" && !command) {
    console.error('usage: bun scripts/emulators.ts exec "<command>"');
    process.exit(2);
  }
  const stack = loadLocalStack();
  const configPath = await writeLocalConfig(stack);
  console.error(
    `emulators: offset ${stack.offset}, project ${stack.projectId}, firestore :${stack.ports.firestore}, auth :${stack.ports.auth}, ui :${stack.ports.ui}`,
  );
  const proc = Bun.spawn(firebaseArgs(stack, configPath, mode, command), {
    cwd: repoRoot,
    stdio: ["inherit", "inherit", "inherit"],
    env: {
      ...process.env,
      PORT_OFFSET: String(stack.offset),
      DEMO_PROJECT_ID: stack.projectId,
    },
  });
  process.exit(await proc.exited);
}
