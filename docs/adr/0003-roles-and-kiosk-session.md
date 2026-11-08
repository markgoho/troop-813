# Roles and the Kiosk session: Firestore roles, Google unlock, Leader menu on the phone

Status: accepted (2026-11-08). Decided in [Leader auth and Kiosk session model](https://github.com/markgoho/troop-813/issues/6). Builds on [ADR 0001](0001-stack-and-hosting.md).

## Context

The Meeting Kiosk runs on one tablet that Scouts pass around. Only the Admin unlocks it. Leaders and Parents sign in with Google on their own devices. A Leader must be able to fix check-ins and end the Meeting, but anything a Leader can do on the tablet, a Scout holding it can try.

## Decision

- **Roles live in Firestore**, in `users/{uid}`, created at first Google sign-in with role `parent`. The Admin promotes Parent↔Leader on an Admin dashboard. `admin` is set only by a bootstrap script. Go reads the role on every Leader API call, so a demotion takes effect on the next request. No Firebase custom claims for Leader roles.
- **Unlock = Admin Google sign-in on the tablet**, then Start today's Meeting (any weekday, created if absent). Go writes `kioskSessions/{id}` and mints a custom token (`role: kiosk`, `kioskSessionId`, `meetingId`). The PWA then signs the Admin's session out.
- **One Kiosk credential.** The Kiosk's Firebase ID token authorizes both Firestore reads and API calls. Go checks that the session doc is active on every call. Firebase refresh tokens never expire, so revocation is server-side: End Meeting, a new unlock, or 4 h after unlock.
- **Kiosk scope**: it reads Roster names, upcoming Events, this Meeting's live state and the current Scout's RSVPs. It writes check-ins to this Meeting, and RSVPs only for Scouts checked in during this session. It can do nothing else.
- **No Leader actions on the tablet.** Undo check-in, Leader check-in and End Meeting live in the live Meeting view of the Leader screens on the Admin's or a Leader's phone. The tablet locks through `onSnapshot` when its session is revoked.

## Considered and rejected

- **Google re-auth on the tablet for the Leader menu**: Chrome keeps the Google account signed in, so re-auth turns into a one-tap account chooser any Scout can use. Forced re-prompt isn't reliable.
- **Per-Leader or shared PIN on the tablet**: a second credential to build and manage, when the Leader's phone is already at hand.
- **Phone approves the tablet (QR/code)**: the most secure, but the most to build for a single-Admin unlock.
- **Allowlist by email in config or custom claims**: needs a redeploy or a token refresh to change, and doesn't fit sign-in-then-promote.
- **Separate opaque Kiosk session token**: two credentials to keep in sync for no gain.

## Consequences

- If the Admin misses a Meeting, there's no Kiosk. Attendance has to be entered afterwards from the Leader screens.
- Every Leader and Kiosk API call costs a Firestore read for the role or session check, which is trivial at troop scale.
- The Admin's Google account stays signed in to Chrome in the Kiosk tablet user. That's acceptable because unpinning needs the Android PIN, and the app holds no Leader session after unlock.
