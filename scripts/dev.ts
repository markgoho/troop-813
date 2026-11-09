// `bun run dev`: Firestore + Auth emulators, the Go API and HQ (Vite, proxying
// /api to Go), all on this worktree's ports. Ctrl-C stops everything; if any
// one process exits, the rest are stopped too.
import type { Subprocess } from "bun";
import { firebaseArgs, writeLocalConfig } from "./emulators";
import { emulatorEnv, loadLocalStack, repoRoot } from "./ports";

const stack = loadLocalStack();
const { ports } = stack;
const baseEnv = {
  ...process.env,
  ...emulatorEnv(stack),
  PORT_OFFSET: String(stack.offset),
  DEMO_PROJECT_ID: stack.projectId,
};

const children: Subprocess[] = [];
let stopping = false;

async function stop(code: number): Promise<never> {
  if (!stopping) {
    stopping = true;
    for (const child of children) child.kill("SIGINT");
    // The emulators need a moment to stop their Java process; don't wait forever.
    await Promise.race([
      Promise.all(children.map((c) => c.exited)),
      Bun.sleep(10_000),
    ]);
  }
  process.exit(code);
}

function start(
  name: string,
  cmd: string[],
  cwd: string,
  env: Record<string, string | undefined> = baseEnv,
) {
  const child = Bun.spawn(cmd, {
    cwd,
    env,
    stdio: ["inherit", "inherit", "inherit"],
  });
  children.push(child);
  void child.exited.then((code) => {
    if (!stopping) {
      console.error(`dev: ${name} exited (${code}); stopping everything`);
      void stop(code || 1);
    }
  });
}

const lan = process.argv.includes("--lan");

async function waitForPort(
  name: string,
  port: number,
  timeoutMs = 120_000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline && !stopping) {
    try {
      const socket = await Bun.connect({
        hostname: "127.0.0.1",
        port,
        socket: { data() {} },
      });
      socket.end();
      return;
    } catch {
      await Bun.sleep(250);
    }
  }
  if (!stopping) {
    console.error(
      `dev: ${name} not listening on :${port} after ${timeoutMs}ms`,
    );
    await stop(1);
  }
}

process.on("SIGINT", () => void stop(0));
process.on("SIGTERM", () => void stop(0));

const configPath = await writeLocalConfig(stack);
start("emulators", firebaseArgs(stack, configPath, "start"), repoRoot);
await waitForPort("firestore emulator", ports.firestore);
await waitForPort("auth emulator", ports.auth);

// Build then run the binary, rather than `go run`, so the API gets our signals directly.
const apiBinary = `${repoRoot}.dev/api`;
const build = Bun.spawnSync(["go", "build", "-o", apiBinary, "."], {
  cwd: `${repoRoot}api`,
  stdio: ["inherit", "inherit", "inherit"],
});
if (build.exitCode !== 0) await stop(build.exitCode ?? 1);
start("api", [apiBinary], `${repoRoot}api`, {
  ...baseEnv,
  PORT: String(ports.api),
});
await waitForPort("api", ports.api);

// HQ's Firebase client config (see apps/hq/src/lib/firebase.ts): demo values
// plus the emulator addresses, so sign-in uses the Auth emulator's fake Google.
start(
  "hq",
  [`${repoRoot}apps/hq/node_modules/.bin/vite`, "dev"],
  `${repoRoot}apps/hq`,
  {
    ...baseEnv,
    VITE_FIREBASE_API_KEY: "demo-api-key",
    VITE_FIREBASE_APP_ID: "demo-app-id",
    VITE_FIREBASE_PROJECT_ID: stack.projectId,
    // --lan (`bun run dev:lan`): HQ listens on every interface and proxies
    // the emulators through its own port, so a tablet on the home network
    // needs only http://<this machine>:<hq port>.
    ...(lan
      ? {
          HQ_LAN: "1",
          VITE_FIREBASE_AUTH_EMULATOR_URL: "same-origin",
          VITE_FIRESTORE_EMULATOR_HOST: "same-origin",
        }
      : {
          VITE_FIREBASE_AUTH_EMULATOR_URL: `http://127.0.0.1:${ports.auth}`,
          VITE_FIRESTORE_EMULATOR_HOST: `127.0.0.1:${ports.firestore}`,
        }),
  },
);
await waitForPort("hq", ports.hq);

console.error(`
dev: ready (port offset ${stack.offset}, project ${stack.projectId})
  HQ            http://127.0.0.1:${ports.hq}${lan ? " (and on this machine's network address)" : ""}
  API           http://127.0.0.1:${ports.api}/api/health
  Emulator UI   http://127.0.0.1:${ports.ui}
`);
