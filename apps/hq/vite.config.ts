import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import { svelteTesting } from "@testing-library/svelte/vite";
import { defineConfig } from "vitest/config";
import { loadLocalStack } from "../../scripts/ports.ts";

const { ports } = loadLocalStack();

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
    proxy: { "/api": `http://127.0.0.1:${ports.api}` },
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
