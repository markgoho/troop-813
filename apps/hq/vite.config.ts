import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import { svelteTesting } from "@testing-library/svelte/vite";
import { defineConfig } from "vitest/config";
import { loadLocalStack } from "../../scripts/ports.ts";

const { ports } = loadLocalStack();
const lan = process.env.HQ_LAN === "1";

export default defineConfig({
  plugins: [
    sveltekit({
      compilerOptions: { runes: true },
      // Static SPA (ADR 0001): unknown paths serve 200.html; Hosting rewrites ** to it.
      adapter: adapter({ fallback: "200.html" }),
    }),
  ],
  server: {
    port: ports.hq,
    strictPort: true,
    // `bun run dev:lan`: listen on the network and proxy the Auth and
    // Firestore emulators through this port (the app connects same-origin).
    host: lan ? true : undefined,
    proxy: {
      "/api": `http://127.0.0.1:${ports.api}`,
      ...(lan
        ? Object.fromEntries([
            ...[
              "/identitytoolkit.googleapis.com",
              "/securetoken.googleapis.com",
              "/emulator",
            ].map((p) => [p, `http://127.0.0.1:${ports.auth}`]),
            [
              "/google.firestore.v1.Firestore",
              `http://127.0.0.1:${ports.firestore}`,
            ],
          ])
        : {}),
    },
  },
  test: {
    expect: { requireAssertions: true },
    projects: [
      {
        extends: true,
        plugins: [svelteTesting()],
        test: {
          name: "hq",
          environment: "jsdom",
          include: ["src/**/*.test.ts"],
          setupFiles: ["./vitest-setup.ts"],
        },
      },
      {
        extends: true,
        test: {
          name: "rules",
          environment: "node",
          include: ["rules/**/*.test.ts"],
        },
      },
    ],
  },
});
