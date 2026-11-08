# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- **Scouts** (Scouts BSA, ages 11–17, ~30 on the Roster) at the Monday Meeting, using the **Meeting Kiosk** on one shared troop Pixel Tablet. Each Scout confirms who they are (records Attendance), answers RSVPs for upcoming Events, then hands the tablet to the Scout the Kiosk names (Handoff). Short, one-at-a-time sessions.
- **Leaders** on their own phones (Leader screens: Tonight / Roster / Events / Meetings). They watch and fix tonight's Meeting live, manage the Roster and Events, backfill past Meetings, set RSVPs and check per-Scout history.
- **Admin**: the one Leader who unlocks the Kiosk on the tablet (Google sign-in, then Start today's Meeting) and grants Leader access (Access tab).
- **Parents** who sign in see only a waiting page. Parent screens are planned for later, not built.

## Product Purpose

Troop 813 HQ records who comes to Monday Meetings and who's going to upcoming Events (mostly monthly Fri–Sun Campouts), captured at the Meeting itself with little Leader effort. Success: Leaders can answer "how often has this Scout been coming?" and "who's going to the Campout?" from their phones, and fix mistakes afterwards. Must cost about $0/mo.

## Positioning

Capture happens in the room, by the Scouts themselves: the tablet passes Scout to Scout via a server-held, randomized Handoff chain, so Attendance and RSVPs are collected without a Leader with a clipboard. Leaders supervise and correct from their phones, live.

## Operating Context

- Meetings: Mondays 7:00–8:30pm at the Troop 92 lodge in Spencerport, NY.
- Kiosk: installed PWA, fullscreen, app-pinned with an Android PIN (`docs/kiosk-tablet-setup.md`). Docked or plugged in before the Meeting; during it the tablet may be handed around or left on its stand, so **orientation varies: Kiosk must work in both landscape and portrait.**
- Kiosk sleeps after 30 min idle ("Kiosk asleep, ask a Leader") and expires 22:00 America/New_York; a Leader wakes it from a phone (Wake Kiosk). Screen is held awake.
- Leader screens: Leaders' phones, during the Meeting (live view) and afterwards.
- Single route on the tablet; Back is harmless; no Leader actions on the tablet.

## Capabilities and Constraints

- Spec: [#17 Meeting Kiosk MVP](https://github.com/markgoho/troop-813/issues/17). ADRs `docs/adr/0001`–`0005`. Vocabulary: root `GLOSSARY.md` (Scout, Leader, Admin, Parent, Roster, Meeting, Event, Campout, Attendance, RSVP, HQ, Meeting Kiosk, Handoff); use these terms, avoid their listed alternatives.
- SvelteKit 5 static PWA (`apps/hq`), Go API on Cloud Run as the only Firestore writer, live reads via `onSnapshot`.
- Kiosk RSVP answers: Going / Not going / Ask me next week; at the last Meeting before an Event, "Not sure" plus a last-chance reminder. Summary with Change, then "Done, pass the tablet on".
- Copy fixed by the spec (e.g. Handoff prompt, last-chance reminder, data notice, "Kiosk asleep, ask a Leader") stays as specified. Visual work must not change behaviour.
- Names on the Kiosk: first name, plus last initial only when two active Scouts share a first name.

## Brand Commitments

- Name: "Troop 813 HQ" (short "813 HQ"). Troop 813 of Spencerport, NY, Scouts BSA.
- **Logo** (must be used): circular single-line badge, black line art: two outer rings around mountains, crescent moon, stars, pine trees, a tent and campfire, with "TROOP 813" lettered along the bottom arc. Monoline, hand-drawn feel.
- **Troop colour: burgundy.** A troop flyer uses `#7a004b`; the troop's usual burgundy is a bit darker than that (exact value not fixed). The manifest's `#1f4d2b` green is a placeholder, not a troop colour.
- **Pinned look (2026-11-08):** the prototypes on branches `prototype/kiosk-flow` and `prototype/leader-screens` (see [#34](https://github.com/markgoho/troop-813/issues/34)), with troop burgundy replacing the prototype's green accent; green stays only as a success/"Going" state. The user rejected the impeccable direction roll (and its safer re-roll) in favour of this.
- The Kiosk must not feel like a wall of names: the Handoff (one name, big) is the main experience; the name list (required by the spec) stays secondary.

## Evidence on Hand

- Real Scout names are youth data: never put them in designs, fixtures or screenshots. Use `bun run seed` fictional data.
- Logo: `src/lib/troop-813-logo.svg`, a clean monoline vector redrawn from the flyer (user-requested, 2026-11-08), drawn in `currentColor`. Use it through `src/lib/Logo.svelte`, which inlines it and keeps a fixed on-screen line width. A troop-supplied original vector would still be the authority if one turns up. No photography, testimonials or other assets exist; don't fabricate them.

## Product Principles

1. The Kiosk flow must be obvious to an 11-year-old holding it for the first time, with no Leader explaining it.
2. Never lose or double-count Attendance: state is on the server, the UI reflects it, and walking away is safe.
3. Leaders glance, then act: status first, fixes one tap away, every fix reversible and explained.
4. Scouts holding the tablet get no power beyond their own check-in and RSVPs.
5. Youth privacy: minimum data, no names outside the app.

## Accessibility & Inclusion

WCAG 2.2 AA. Kiosk: large tap targets for a shared tablet, no reliance on colour alone (checked-in state uses a check mark, not just grey).
