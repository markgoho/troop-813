# Stack and hosting: static SvelteKit, Go on Cloud Run, Firestore, Identity Platform

Status: accepted (2026-11-07). Decided in [Stack and hosting decision](https://github.com/markgoho/troop-813/issues/5), informed by [Free-tier GCP hosting and data options](https://github.com/markgoho/troop-813/issues/2).

## Context

The Meeting Kiosk needs a front end, a server-enforced Kiosk session, a datastore and Leader sign-in. The constraints are roughly $0/mo, real GCP services where a free path exists, Svelte + Go, and a tiny load: about 40 Scouts, one Meeting a week, and more than one Kiosk device per Meeting.

## Decision

- **Front end**: SvelteKit built static (adapter-static), served from **Firebase Hosting**. `/api/**` rewrites to Cloud Run in `us-east1`, so the browser sees one origin.
- **API**: one **Go service on Cloud Run** in `us-east1`. It scales to zero and uses request-based billing. It is stateless, and nothing about a Meeting lives in instance memory. It is the **only writer** to the database. Contended writes, such as two Kiosks claiming the same Scout, go through Firestore transactions.
- **Database**: **Firestore** (Native mode, `us-east1`). Browsers subscribe with `onSnapshot` to read-only live Meeting state, which keeps multiple Kiosks in sync. Security rules allow only those reads and deny every write.
- **Auth**: **Identity Platform**, used through the Firebase Auth JS SDK, for Leader Google sign-in. Go verifies ID tokens and checks the Leader allowlist. On unlock, Go mints a custom token that gives the Kiosk its Firestore read identity. The session model itself is decided in [Leader auth and Kiosk session model](https://github.com/markgoho/troop-813/issues/6).
- **Rewrite constraints**: Hosting strips every cookie except `__session` and times out after 60 s. Leader API calls send `Authorization: Bearer <ID token>`.
- **Billing**: a dedicated billing account, separate from other projects so the free quotas aren't shared. It is upgraded to Paid immediately (not left on the 90-day trial) and has a $1 budget alert. There is no billing kill switch.
- **Backups**: none for the MVP. We accept the risk.

## Considered and rejected

- **Cloud SQL**: about $9/mo minimum, which breaks the $0 target.
- **Postgres on an e2-micro VM**: $0, but we'd own OS and Postgres patching and backups. SQL isn't worth that at this data size.
- **Neon**: $0 Postgres, but it runs on AWS, so there's no GCP experience on the data side, and it adds a second cold start.
- **SQLite/Litestream on Cloud Run**: single-writer and replication-lag risks could silently lose a Meeting's Attendance.
- **SvelteKit server routes (adapter-node)**: that would be a second server alongside Go, or dropping Go entirely.
- **Front end embedded in the Go binary**: the first page load would wait on a cold start, and there'd be no CDN.
- **WebSockets for Kiosk sync**: the Hosting rewrite times out at 60 s, sockets don't fan out across Cloud Run instances, and an open connection bills the whole Meeting.
- **Polling for Kiosk sync**: it works, but listeners are native, instant and use fewer requests.

## Consequences

- Leader reports, like per-Scout Attendance history, are document queries. Any denormalization happens in Go.
- The rules file isn't deny-all. It must be kept narrow to the live Meeting state.
- There is no recovery from bad deletes or bad writes. The data model should prefer deactivating over deleting.
- The first request each Monday pays a Cloud Run cold start.
