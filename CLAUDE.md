## Toolchain

Monorepo. Layout and tooling decided in [Monorepo layout and tooling](https://github.com/markgoho/troop-813/issues/8).

- `apps/*`: SvelteKit apps (Bun workspaces). `api/`: the Go API (its own Go module). Firebase config at the root.
- Run every check from the root through `package.json` scripts (`bun run check`, `test`, `lint`, `fmt`, `gen`, `dev`).
- Go↔TS types come from Go structs via `tygo` ([ADR 0002](docs/adr/0002-go-generated-contract-types.md)). Don't hand-write TS types for API bodies or Firestore docs. Run `bun run gen` and commit the output.

### SvelteKit apps (`apps/*`)

- SvelteKit is built on Vite. Use Vite and Vitest there, run through Bun (`bun run dev`, `bun run test`).
- Tests: Vitest + `@testing-library/svelte` + jsdom, for components and pure functions alike.
- Typecheck with `svelte-check` (`bun check`/`tsc` can't read `.svelte`). Lint with ESLint (until `bun lint` ships).
- The Svelte MCP server (`.mcp.json`) has Svelte 5 docs and an autofixer. Use it when writing Svelte.

### Go (`api/`)

- `gofmt`, `go vet`, `go test ./...`, `golangci-lint` (v2).

### Formatting

- `bun fmt` formats everything except Go (TS, Svelte, JSON, YAML, Markdown). `gofmt` formats Go. They don't overlap.

### Everywhere else (repo-root scripts)

Default to Bun instead of Node.js.

- `bun <file>` not `node`/`ts-node`. `bun test` for root scripts. `bun install`, `bun run`, `bunx`.
- Bun loads `.env` automatically, so don't use dotenv.
- Prefer `Bun.file` over `node:fs` readFile/writeFile, and Bun.$`ls` over execa.
- Bun API docs: `node_modules/bun-types/docs/**.mdx`.

## Agent skills

### Issue tracker

GitHub Issues on `markgoho/troop-813` via `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five canonical labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: root `GLOSSARY.md` + `docs/adr/`. See `docs/agents/domain.md`.

## Git workflow

Commit and push directly to `trunk`. No feature branches or PRs in this repo.
