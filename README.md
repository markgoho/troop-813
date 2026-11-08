# troop-813

Monorepo for Troop 813 apps and websites.

- `apps/hq`: Troop 813 HQ, the SvelteKit sign-in app (Meeting Kiosk + Leader screens)
- `api`: the Go API, the only writer to Firestore
- `firebase.json`, `firestore.rules`, `firestore.indexes.json`: Firebase config
- `scripts`: repo-root Bun scripts (local stack, emulators)
- `terraform`: prod infra, applied only by CI (ADR 0005); `bootstrap.sh` is the one-time setup
- `.github/workflows`: `api.yml`, `web.yml`, `terraform.yml`, each path-filtered: checks, then deploy on push to `trunk`

## Prod

`https://troop-813-hq.web.app` (Hosting, `/api/**` rewritten to Cloud Run `api` in us-east1), project `troop-813`. A push to `trunk` deploys whatever it touched. The one-time manual steps (billing, $1 budget, `terraform/bootstrap.sh`, Google sign-in provider, OAuth redirect URI) are a wizard:

```sh
scripts/prod-setup-wizard.sh
```

## Prerequisites

Bun, Go (version in `api/go.mod`), Java 21+ (Firestore emulator), `golangci-lint` v2. Then `bun install`.

## Commands (run from the root)

| Command           | What it does                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------ |
| `bun run dev`     | Firestore + Auth emulators, the Go API and HQ (Vite proxies `/api` to Go). Prints the URLs.      |
| `bun run test`    | Root script tests, then boots the emulators once and runs Go tests, rules tests and HQ tests.    |
| `bun run test:go` | Just the Go tests, against the emulators.                                                        |
| `bun run check`   | `tsc` on root scripts, `svelte-check` on each app, `go vet`.                                     |
| `bun run lint`    | ESLint on each app, `golangci-lint` on the API.                                                  |
| `bun run fmt`     | Bun's formatter (everything but Go) and `gofmt`. `--check` to only check (CI). See note below.   |
| `bun run gen`     | Regenerates `apps/hq/src/lib/contract.gen.ts` from `api/internal/contract` with tygo. Commit it. |

Note: because the root has a `fmt` script, a bare `bun fmt …` at the root runs that script (`scripts/fmt.ts`), not Bun's formatter directly. The script honours `--check` and paths, so `bun fmt --check .` still works.

## Ports, port offset and project ID

All local ports sit in one block of 10 starting at 21000: Firestore, Firestore websocket, Auth, hub, logging, Emulator UI, API, HQ (in that order, so offset 0 is 21000–21007). To run several checkouts at once (e.g. git worktrees), give each its own offset:

```sh
echo 3 > .port-offset     # this checkout uses 21030–21037 (gitignored file)
PORT_OFFSET=3 bun run test   # or per command; the env var wins over the file
```

The emulators run as project `demo-troop-813` (the `demo-` prefix keeps them offline and away from the real project). Override with `DEMO_PROJECT_ID=demo-something`. Tests go further: each Go test and each rules test file gets its own `demo-…-<random>` project, so they never see each other's data.

`scripts/emulators.ts` writes the shifted config to the gitignored `.firebase.local-<offset>.json` and runs `firebase emulators:exec`/`start` with it. Use it to run anything against the emulators: `bun scripts/emulators.ts exec "<command>"`.
