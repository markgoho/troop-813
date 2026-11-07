# Go structs are the single source of Go↔TS contract types

Status: accepted (2026-11-07). Decided in [Monorepo layout and tooling](https://github.com/markgoho/troop-813/issues/8).

The front end and Go share two contracts: HTTP request and response bodies, and the **Firestore live-state documents** that Go writes and browsers read raw through `onSnapshot` ([ADR 0001](0001-stack-and-hosting.md)). No API layer validates those documents, so if the TS types drift from what Go writes, the bug is silent. Both contracts are defined as Go structs in `api/internal/contract`. `tygo` generates TS types into the SvelteKit app, the output is committed, and CI fails if `go generate` leaves a diff.

## Considered options

- **OpenAPI spec-first** (oapi-codegen + openapi-typescript): too heavy for about 15 endpoints, and it doesn't describe Firestore documents at all.
- **Hand-written types on both sides**: drift is guaranteed, and on Firestore documents nothing catches it.

## Consequences

- Types only, with no runtime validation in the browser. Go is the only writer, so it is trusted.
- Contract structs must use types tygo can map (no unexported or exotic types in that package).
