// Local stack ports and project ID, shared by the emulator wrapper, `bun run dev`
// and apps/hq/vite.config.ts. Plain Node APIs only: vite.config loads this under Node.
//
// Every port sits in one block of PORT_STEP starting at BASE_PORT. A worktree picks
// its block with PORT_OFFSET (env) or a `.port-offset` file at the repo root, so
// concurrent checkouts never collide. Offset 0 matches the ports in firebase.json.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

export const BASE_PORT = 21000;
export const PORT_STEP = 10;
export const DEFAULT_PROJECT_ID = "demo-troop-813";

const SLOTS = [
  "firestore",
  "firestoreWebsocket",
  "auth",
  "hub",
  "logging",
  "ui",
  "api",
  "hq",
] as const;

export type Ports = Record<(typeof SLOTS)[number], number>;
export type Env = Record<string, string | undefined>;

export interface LocalStack {
  offset: number;
  ports: Ports;
  projectId: string;
}

export const repoRoot = fileURLToPath(new URL("..", import.meta.url));

function parseOffset(raw: string, source: string): number {
  const text = raw.trim();
  if (!/^\d+$/.test(text)) {
    throw new Error(`${source} must be a non-negative integer, got "${raw}"`);
  }
  return Number(text);
}

export function portOffset(env: Env, fileContents: string | null): number {
  if (env.PORT_OFFSET !== undefined && env.PORT_OFFSET !== "") {
    return parseOffset(env.PORT_OFFSET, "PORT_OFFSET");
  }
  if (fileContents !== null) return parseOffset(fileContents, ".port-offset");
  return 0;
}

export function localStack(env: Env, fileContents: string | null): LocalStack {
  const offset = portOffset(env, fileContents);
  const base = BASE_PORT + offset * PORT_STEP;
  const ports = Object.fromEntries(
    SLOTS.map((slot, i) => [slot, base + i]),
  ) as Ports;
  const projectId = env.DEMO_PROJECT_ID || DEFAULT_PROJECT_ID;
  if (!projectId.startsWith("demo-")) {
    throw new Error(
      `DEMO_PROJECT_ID must start with "demo-" so the emulators never reach a real project, got "${projectId}"`,
    );
  }
  return { offset, ports, projectId };
}

/** Reads PORT_OFFSET, DEMO_PROJECT_ID and `<repo root>/.port-offset`. */
export function loadLocalStack(env: Env = process.env): LocalStack {
  let fileContents: string | null = null;
  try {
    fileContents = readFileSync(`${repoRoot}.port-offset`, "utf8");
  } catch {
    // No file: offset 0 (main checkout, CI).
  }
  return localStack(env, fileContents);
}

interface EmulatorBlock {
  host?: string;
  port?: number;
  websocketPort?: number;
  enabled?: boolean;
}

export interface FirebaseConfig {
  emulators: Record<string, EmulatorBlock | boolean> & {
    firestore: EmulatorBlock;
    auth: EmulatorBlock;
    hub: EmulatorBlock;
    logging: EmulatorBlock;
    ui: EmulatorBlock;
  };
  [key: string]: unknown;
}

/** firebase.json with every emulator port moved into this stack's block. */
export function emulatorConfig<T extends FirebaseConfig>(
  config: T,
  ports: Ports,
): T {
  const e = config.emulators;
  return {
    ...config,
    emulators: {
      ...e,
      firestore: {
        ...e.firestore,
        port: ports.firestore,
        websocketPort: ports.firestoreWebsocket,
      },
      auth: { ...e.auth, port: ports.auth },
      hub: { ...e.hub, port: ports.hub },
      logging: { ...e.logging, port: ports.logging },
      ui: { ...e.ui, port: ports.ui },
    },
  };
}

/** The env the Admin SDKs, Go and @firebase/rules-unit-testing read to find the emulators. */
export function emulatorEnv(stack: LocalStack): Record<string, string> {
  return {
    FIRESTORE_EMULATOR_HOST: `127.0.0.1:${stack.ports.firestore}`,
    FIREBASE_AUTH_EMULATOR_HOST: `127.0.0.1:${stack.ports.auth}`,
    GCLOUD_PROJECT: stack.projectId,
  };
}
