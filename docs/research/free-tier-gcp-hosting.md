# Free-tier GCP hosting and data options

Research for [#2](https://github.com/markgoho/troop-813/issues/2), under map [#1](https://github.com/markgoho/troop-813/issues/1). Sources were read 2026-11-07. Prices are USD, Tier 1 / `us-central1` unless noted. **This doc doesn't make the decision. That belongs to [#5](https://github.com/markgoho/troop-813/issues/5).**

## TL;DR

- **$0/mo is realistic** with Firebase Hosting (static SvelteKit), Go on Cloud Run (scale to zero) and **Firestore**, with Identity Platform for Leader Google sign-in. All usage sits 1–2 orders of magnitude below the free tiers.
- **Cloud SQL is not free.** The cheapest floor is about **$8.5–9.5/mo** (db-f1-micro plus 10 GB of storage plus IP).
- **Postgres on the always-free e2-micro** is $0 for compute and disk. The external IPv4 may cost about **$3.65/mo**, and you run the OS, patching and backups yourself.
- **SQLite on Cloud Run** costs $0 but is the riskiest for data. GCS FUSE has no file locking, and Litestream needs CPU between requests and exactly one writer.
- **Neon free** is $0 and is real Postgres, but it runs on AWS only, so you get no GCP experience on the data side.
- **A billing account is required no matter what**, because Cloud Run needs one. That puts the Firebase project on Blaze. Free quotas still apply on Blaze. Nothing hard-caps spend unless you script it.

## 1. Framing facts that collapse several sub-questions

| Fact | Source |
|---|---|
| Cloud Run needs a Cloud Billing account: you "still must have a Cloud Billing account associated with your Firebase project to use or try out Cloud Run." | [Firebase: Hosting + Cloud Run](https://firebase.google.com/docs/hosting/cloud-run) |
| Linking a billing account (including a $300-trial account) auto-upgrades the Firebase project to **Blaze**. Blaze "keeps the no-cost quotas" and adds pay-as-you-go for overage. | [Firebase pricing plans](https://firebase.google.com/docs/projects/billing/firebase-pricing-plans) |
| The GCP Free Tier needs an active billing account "in good standing" (Paid or Free Trial). | [GCP free features](https://docs.cloud.google.com/free/docs/free-cloud-features) |
| The Cloud Run free tier "is aggregated across projects by billing account". If this project shares a billing account with doula-cloud or anything else, they share the 180k vCPU-s and 2M requests. | [Cloud Run pricing](https://cloud.google.com/run/pricing) |
| $300 trial: 90 days. If you don't upgrade, "billing is disabled and resources are stopped". Data is marked for deletion, followed by a 30-day grace period. **Upgrade to a Paid account before day 90 or the Kiosk dies around week 13.** | [GCP free features](https://docs.cloud.google.com/free/docs/free-cloud-features) |
| Budget alerts "do not cap your usage or charges". Spend-cap budgets exist only for Firebase AI Logic, App Hosting, Cloud Functions for Firebase and Extensions, not for Cloud Run, Firestore or Hosting. The only hard stop is to "programmatically disable Cloud Billing" from a budget Pub/Sub notification. | [Avoid surprise bills](https://firebase.google.com/docs/projects/billing/avoid-surprise-bills) |

So "Spark vs Blaze" and "does X need billing" are moot. You will be on Blaze with a card on file. The question is only whether usage stays under the free quotas.

**Region:** `us-east1` (South Carolina) is the closest region to Spencerport, NY that has every free tier. It is a Cloud Run Tier 1 region, a free e2-micro region, a GCS free-tier region, and on Firebase's recommended list for Hosting→Cloud Run rewrites ([Cloud Run pricing](https://cloud.google.com/run/pricing), [free features](https://docs.cloud.google.com/free/docs/free-cloud-features), [Hosting + Cloud Run](https://firebase.google.com/docs/hosting/cloud-run)).

## 2. Cloud Run (Go API)

| Item | Value | Source |
|---|---|---|
| Free, request-based billing (default) | 180,000 vCPU-s, 360,000 GiB-s, 2M requests per month | [pricing](https://cloud.google.com/run/pricing) |
| Free, instance-based billing / Jobs | 240,000 vCPU-s, 450,000 GiB-s per month (no request allowance listed) | [pricing](https://cloud.google.com/run/pricing) |
| Free egress | 1 GiB/month within North America | [pricing](https://cloud.google.com/run/pricing) |
| Request-based, active | $0.000024/vCPU-s, $0.0000025/GiB-s, $0.40 per 1M requests | [pricing](https://cloud.google.com/run/pricing) |
| Request-based, idle min-instance | $0.0000025/vCPU-s, $0.0000025/GiB-s | [pricing](https://cloud.google.com/run/pricing) |
| Instance-based | $0.000018/vCPU-s, $0.000002/GiB-s, billed for the whole instance lifetime, minimum 1 min | [pricing](https://cloud.google.com/run/pricing) |
| How the free tier is applied | "as a spending based discount using Tier 1 pricing" | [pricing](https://cloud.google.com/run/pricing) |

**Fit for us.** A Monday is about 40 check-ins times a handful of calls each, plus Leader screens. Call it at most a few thousand requests a month against a 2M allowance. Request-based billing counts only time when a request is in flight. Even a pessimistic 90 min × 1 vCPU every Monday is about 23k vCPU-s/month, roughly 13% of 180k. **$0.**

**Cold starts and scaling gotchas**
- With request-based billing, "CPU is only allocated during request processing." An instance "will never stay idle for more than 15 minutes after processing a request", and SIGTERM gives a 10 s grace period ([billing settings](https://docs.cloud.google.com/run/docs/configuring/billing-settings)). So the first request each Monday is a cold start. After that the instance stays warm through the Meeting.
- Google publishes no cold-start latency figure. Startup CPU boost exists. Requests queue for up to max(3.5× average startup, 10 s) ([general tips](https://docs.cloud.google.com/run/docs/tips/general)). A small static Go binary is about the best case. This is not quantified here, so measure it.
- `min-instances=1` removes cold starts but **is billed** ([min instances](https://docs.cloud.google.com/run/docs/configuring/min-instances)). At 1 vCPU / 512 MiB idle all month: 2.628M s × ($0.0000025 + 0.5 × $0.0000025) ≈ $9.86 gross. The free-tier credit is worth at most $4.32 + $0.90 = $5.22, so the net is **≈ $4–5/mo** (my arithmetic). Not $0.
- Max instances "can be exceeded for a brief period due to circumstances such as traffic spikes". Excess requests get 429 ([max instances](https://docs.cloud.google.com/run/docs/configuring/max-instances)). This matters for any single-writer design (SQLite).

## 3. Static assets: Firebase Hosting vs Cloud Run

| | Firebase Hosting | Serve from Cloud Run (Go `embed.FS`) |
|---|---|---|
| Cost | Free: 10 GB storage. Transfer is listed as **360 MB/day** on the [pricing page](https://firebase.google.com/pricing) and **10 GB/month** on the [quotas page](https://firebase.google.com/docs/hosting/usage-quotas-pricing) (about the same). Blaze overage is $0.026/GB storage and $0.15/GB transfer. | Counts against Cloud Run's 1 GiB/month North America egress, then network pricing |
| CDN / TLS / custom domain | Global CDN, managed TLS, custom domain | No CDN. A custom domain needs Cloud Run domain mapping or a load balancer (not verified here) |
| Cold start | None for HTML/JS | First page load waits on the cold start |
| `/api/**` | Rewrite to Cloud Run gives one origin and no CORS (same as doula-cloud) | Natively the same origin |
| Deploys | Two deploy targets | One artifact |

**Hosting→Cloud Run rewrite gotchas**
- 60 s request timeout, then 504 ([Hosting + Cloud Run](https://firebase.google.com/docs/hosting/cloud-run)).
- Cookies are stripped. "Only the specially-named `__session` cookie is permitted to pass through". Hosting adds `Cookie` and `Authorization` to `Vary` for dynamic content ([manage cache](https://firebase.google.com/docs/hosting/manage-cache)). In practice, send the Firebase ID token as `Authorization: Bearer`, or use a `__session` cookie for the Kiosk unlock.
- Spark-only caveat (we won't be on Spark): when transfer is exceeded, "your sites will be disabled" until next month ([quotas](https://firebase.google.com/docs/hosting/usage-quotas-pricing)).
- Default rewrite region is `us-central1`. Set `region` explicitly ([Hosting + Cloud Run](https://firebase.google.com/docs/hosting/cloud-run)).

The pure-GCP option for static files is GCS plus an external HTTPS load balancer. Forwarding rules are billed hourly, so that is not $0 (rate not fetched here). Bundle size is a few hundred KB, cached on one tablet, so egress is negligible on any option.

## 4. Database options

### 4a. Firestore (Native mode)

| Item | Value | Source |
|---|---|---|
| Free quota | 1 GiB stored, 50,000 reads, 20,000 writes, 20,000 deletes **per day**, 10 GiB egress/month | [Firestore pricing](https://cloud.google.com/firestore/pricing), [free features](https://docs.cloud.google.com/free/docs/free-cloud-features) |
| Scope | "exactly one free database per project". Quotas reset around midnight Pacific | [Firestore pricing](https://cloud.google.com/firestore/pricing) |
| Overage | $0.03 per 100k reads, $0.09 per 100k writes, $0.01 per 100k deletes | [Firestore pricing](https://cloud.google.com/firestore/pricing) |
| Not in free tier | PITR, backups, restore, TTL deletes | [Firestore pricing](https://cloud.google.com/firestore/pricing) |

**Fit.** ~40 Scouts, ~50 Meetings/yr, ~12 Events/yr means a few thousand Attendance docs a year. A Monday is probably under 5k reads and under 200 writes, at least 10× under the daily caps. No cold start, no server, GCP-native, official Go client. **$0.**
**Cost of fit:** it is a document store, so there are no joins or SQL. The Leader reports (per-Scout Attendance history, per-Event RSVP list) become denormalized queries and may need composite indexes. Backups aren't free, so you'd have to roll your own export. Choosing Firestore here also shapes the data model ([#5](https://github.com/markgoho/troop-813/issues/5) and the data-model ticket).

### 4b. Cloud SQL for PostgreSQL: true floor

| Item | Value | Source |
|---|---|---|
| db-f1-micro (shared, 0.6 GB) | $0.0105/hr ≈ **$7.67/mo** | [Cloud SQL pricing](https://cloud.google.com/sql/pricing) |
| db-g1-small (shared, 1.7 GB) | $0.035/hr ≈ $25.55/mo | same |
| SSD / HDD storage | $0.000232877 / $0.000123288 per GiB-hr ≈ $0.17 / $0.09 per GiB-mo | same |
| Minimum storage | 10 GB (from the auto-increase setting text) → ≈ $1.70 SSD / $0.90 HDD | [instance settings](https://docs.cloud.google.com/sql/docs/postgres/instance-settings) |
| Idle public IPv4 | $0.01/hr | [Cloud SQL pricing](https://cloud.google.com/sql/pricing) |
| Backups | $0.000109589 per GiB-hr used | same |
| Stopped instance | "charges for storage and IP addresses continue to apply" | [start/stop](https://docs.cloud.google.com/sql/docs/postgres/start-stop-restart-instance) |
| Free trial instance | 30 days, Enterprise Plus 8 vCPU / 64 GB, one per new project. Stops serving after 30 days | [free trial](https://docs.cloud.google.com/sql/docs/postgres/free-trial-instance) |

**Floor ≈ $8.5–9.5/mo** before backups. There is no Always Free tier. Shared-core tiers are "not covered by the Cloud SQL SLA" and are meant for test/dev.
**Gotcha:** PostgreSQL 16+ defaults to **Enterprise Plus**, which has no shared-core machines. Pass `--edition=ENTERPRISE` explicitly ([instance settings](https://docs.cloud.google.com/sql/docs/postgres/instance-settings)). Private IP avoids the IPv4 charge but needs VPC plumbing.

### 4c. Postgres on an always-free e2-micro VM

| Item | Value | Source |
|---|---|---|
| Free compute | 1 non-preemptible e2-micro per month (limit is by hours), `us-west1` / `us-central1` / `us-east1` | [free features](https://docs.cloud.google.com/free/docs/free-cloud-features) |
| Free disk | 30 GB-months **standard** persistent disk | same |
| Free egress | 1 GB/month North America (excluding China and Australia) | same |
| External IPv4 | $0.005/hr in use (≈ $3.65/mo). The free-tier text reads "This free usage is limited to one hour per month per account", and the free-features page lists no IP under Compute Engine. **Treat the IP as billed.** | [network pricing](https://cloud.google.com/vpc/network-pricing) |
| Static IP | Counts as "in use" while attached, even when the VM is stopped | same |

**The $0 variant:** run the VM with no external IP. Cloud Run reaches its internal IP over Direct VPC egress, which is billed as VM data transfer at [VPC rates](https://cloud.google.com/run/pricing) (tiny here). Use IAP TCP forwarding for SSH. The catch: the VM then has no outbound internet for `apt` or Postgres updates without Cloud NAT, which isn't free ($0.005/hr IP per [network pricing](https://cloud.google.com/vpc/network-pricing), plus NAT charges). So you attach a temporary IP when patching.
**Gotchas:** the default boot disk type is not "standard", so pick `pd-standard` explicitly or the 30 GB isn't free (unverified default, so check in the console). The VM has 1 GB RAM. You own OS patching, Postgres upgrades, backups (pg_dump to GCS fits the 5 GB US-region free tier), and monitoring. It is always on, so there are no DB cold starts.

### 4d. SQLite on Cloud Run (GCS volume mount / Litestream)

- **SQLite directly on a GCS FUSE mount is unsafe.** Per the docs: "Cloud Storage FUSE does not provide concurrency control for multiple writes (file locking) to the same file", "the last write wins and all previous writes are lost", and it "is not a fully POSIX-compliant file system" ([GCS volume mounts](https://docs.cloud.google.com/run/docs/configuring/services/cloud-storage-volume-mounts)). SQLite relies on POSIX locks.
- **Litestream pattern** (restore from GCS at startup, write to a local or in-memory file, replicate continuously):
  - Replication is async, "every second" by default, and a crash loses that window. Synchronous replication "has not yet been implemented". Running multiple replicators to the same path can leave you "unable to restore", and "It is your responsibility" to prevent it ([Litestream tips](https://litestream.io/tips/)).
  - On Cloud Run, request-based billing gives CPU only during requests, so background replication stalls between requests. You need **instance-based billing**: 240k vCPU-s free, the whole lifetime billed, and the instance lingers up to 15 min idle. That probably still fits $0 at weekly usage.
  - `max-instances=1` is required but can be "exceeded for a brief period". Revision rollouts can also overlap old and new instances. Either way you get two writers to one replica.
  - The cold start includes the DB restore. An in-memory volume counts against container memory.
  - Each upload is a GCS Class A op (5,000/month free in US regions, [free features](https://docs.cloud.google.com/free/docs/free-cloud-features)). A busy 90-min Meeting syncing every second could exceed that, though the overage is pennies.
- **Verdict:** $0 and SQL, but the failure mode is silently losing Monday's Attendance. That's a poor trade for a Kiosk unless you also accept and test those edge cases. Litestream GCS specifics weren't verified here.

### 4e. Neon (non-GCP fallback)

| Item | Value | Source |
|---|---|---|
| Free plan | 100 projects. 1 GB storage per project (20 GB total). 100 CU-hours per project per month. Autoscale to 2 CU. 10 branches. No card. Permanent | [Neon pricing](https://neon.com/pricing) |
| Scale to zero | Suspends after 5 min idle, and you can't disable this on Free. Resume is "typically within a few hundred milliseconds" | [Neon pricing](https://neon.com/pricing) |
| Regions | AWS only. Nearest are `aws-us-east-1` (N. Virginia) and `aws-us-east-2` (Ohio). No GCP regions | [Neon regions](https://neon.com/docs/introduction/regions) |

**Fit:** real Postgres for $0. Cross-cloud traffic from Cloud Run uses its 1 GiB/month egress (tiny here). The first query after idle pays Neon's resume on top of Cloud Run's cold start. **No GCP learning on the data side**, which cuts against the stated goal.

## 5. Auth: Identity Platform vs Firebase Auth (Leader Google sign-in only)

Only Leaders sign in (a short allowlist), so MAU will be in the single digits. Both options are $0 here.

| | Firebase Auth (legacy, not upgraded) | Firebase Auth with Identity Platform / Identity Platform |
|---|---|---|
| Google sign-in tier | Tier 1 "social" | Tier 1 "social" |
| Free | Spark: unlimited for email/social/anonymous/custom | 0–50,000 MAU free, then $0.0055 down to $0.0025 per MAU. Upgraded Spark projects get 3,000 DAU |
| SAML/OIDC | not available | 50 MAU free, then $0.015/MAU (Spark: 2 DAU) |
| Extras | none | MFA, blocking functions, multi-tenancy, IAP integration, 99.95% SLA |
| Upgrade | "does not require any migration". Same client and Admin SDKs | |

Sources: [Identity Platform pricing](https://cloud.google.com/identity-platform/pricing), [Firebase Auth overview](https://firebase.google.com/docs/auth#identity-platform), [Auth limits](https://firebase.google.com/docs/auth/limits), [product comparison](https://docs.cloud.google.com/identity-platform/docs/product-comparison).

The difference is surface area, not cost. Identity Platform is the GCP-console product (matches doula-cloud and the "real GCP" goal). Legacy Firebase Auth lives in the Firebase console. Either way, the Go API verifies the ID token with the Firebase Admin SDK and checks the Leader allowlist. Spark email limits (password-reset 150/day, email-link 5/day) don't matter for Google-only sign-in. Phone auth is Blaze-only and billed per SMS, so avoid it.

## 6. Other free-tier line items that bite

| Item | Free | Gotcha | Source |
|---|---|---|---|
| Artifact Registry | 0.5 GB/month | Every deploy adds an image. Set a cleanup policy (keep the last N) | [free features](https://docs.cloud.google.com/free/docs/free-cloud-features) |
| Cloud Build | 2,500 build-min/month (e2-standard-2) | Or build in GitHub Actions and push the image | same |
| Secret Manager | 6 active secret versions, 10k accesses/month | Old versions count until destroyed | same |
| Cloud Storage | 5 GB-months, 5k Class A, 50k Class B, 100 GB egress (NA), only in `us-east1`/`us-west1`/`us-central1` | Bucket region matters | same |

## 7. Ranked recommendation (input to #5, not the decision)

The ranking depends on two axes. #5 can flip it by changing either one:
1. **Hard $0, or is ~$4–10/mo tolerable?**
2. **Is a document model acceptable, or do you want SQL/relational?**

| Rank | Stack | $/mo | Why / when |
|---|---|---|---|
| 1 | Firebase Hosting + Cloud Run (Go, scale to zero) + **Firestore** + Identity Platform, all `us-east1` | **$0** | Every piece GCP-native, nothing to operate, ≥10× headroom on every quota. Choose if a document model is OK |
| 2 | Same + **Postgres on e2-micro** (no external IP, Direct VPC egress) | **$0–3.65** | SQL plus real Compute Engine/VPC experience. You run the VM. Choose if SQL matters and you'll do the ops |
| 3 | Same + **Neon free** | **$0** | SQL with zero ops, but not GCP and with double cold starts. The fallback if VM ops are unwanted |
| 4 | Same + **Cloud SQL db-f1-micro** (Enterprise, HDD) | **≈ $8.5–9.5** | The "proper" managed-Postgres GCP experience, like doula-cloud. Choose only if the $0 target relaxes |
| 5 | Cloud Run + **SQLite/Litestream** | **$0** | Cheapest SQL on paper, but single-writer and replication-lag risks with Kiosk data. Not recommended |

Applies to every option: upgrade off the $300 trial right away, use a billing account separate from doula-cloud (or accept shared free quotas), set a budget alert, and optionally wire a budget→Pub/Sub billing kill-switch.

**The decision itself belongs to [#5](https://github.com/markgoho/troop-813/issues/5).**

## Unverified / not fetched

- Concrete Go cold-start latency on Cloud Run (no official figure).
- Cloud Run domain-mapping availability in `us-east1`, and load balancer forwarding-rule rates.
- The default Compute Engine boot-disk type, and whether an e2-micro free-tier external IP waiver still exists. The current network-pricing wording reads as 1 hour/month.
- Litestream GCS replica specifics, and current Litestream version behavior.
- Cloud Logging and Cloud Scheduler free allotments (not needed for the core stack).
