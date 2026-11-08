// Root `fmt` script: Bun's formatter for everything but Go, gofmt for Go.
//
//   bun run fmt                 write
//   bun run fmt --check         check only (CI); exits 1 if anything needs formatting
//   bun run fmt <paths...>      only those paths (Go is formatted only when no paths are given)
//
// Because the root package.json has a "fmt" script, a bare `bun fmt --check .`
// at the root runs this script with those args instead of Bun's formatter, so
// this script honours --check and paths itself.
import { $ } from "bun";
import { repoRoot } from "./ports";

const args = process.argv.slice(2);
const check = args.some((a) => a === "--check" || a === "-c");
const paths = args.filter((a) => !a.startsWith("-"));
const wholeRepo = paths.length === 0 || paths.includes(".");
$.cwd(repoRoot);
$.throws(false);

// With npm_lifecycle_event=fmt, `bun fmt` is the builtin formatter rather than
// a recursive call into this script (verified on Bun 1.4.4).
const env = { ...process.env, npm_lifecycle_event: "fmt" };
const flags = check ? ["--check"] : [];
const targets = wholeRepo ? ["."] : paths;
let failed =
  (await $`${process.execPath} fmt ${flags} ${targets}`.env(env)).exitCode !==
  0;

if (wholeRepo) {
  if (check) {
    const unformatted = (await $`gofmt -l api`.text()).trim();
    if (unformatted) {
      console.error(`gofmt: needs formatting:\n${unformatted}`);
      failed = true;
    }
  } else {
    failed ||= (await $`gofmt -w api`).exitCode !== 0;
  }
}
process.exit(failed ? 1 : 0);
