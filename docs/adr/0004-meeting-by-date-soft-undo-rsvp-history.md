# Meetings keyed by date, soft-undo Attendance, RSVP history linked to check-ins

Status: accepted (2026-11-08). Decided in [Data model for Roster, Meetings, Events, Attendance, RSVPs](https://github.com/markgoho/troop-813/issues/7). Builds on [ADR 0001](0001-stack-and-hosting.md) (Firestore, no backups) and [ADR 0003](0003-roles-and-kiosk-session.md).

## Context

A Leader's Undo check-in must also undo the RSVP answers given during that check-in, while older answers and Leader edits stay. There are no backups, so a bad write or delete can't be recovered, and "why does this Scout show absent?" must be answerable from the data.

## Decision

- **A Meeting's ID is its date** (`meetings/2026-11-09`), so there's one Meeting per day. Start today's Meeting is idempotent: running it again issues a new Kiosk session, with no reshuffle. A skipped holiday is the absence of a doc, not a record. The same doc is the Kiosk's live state (Handoff order, not found, checked in).
- **Undo is a soft delete.** An Attendance record gets `undoneAt`/`undoneBy` and drops out of the Meeting's `checkedIn` map. Checking the Scout in again creates a new record.
- **One RSVP doc per (Event, Scout)**, holding the current `answer` plus a `history` of `{answer, at, by, attendanceId}`. Kiosk answers carry the check-in's `attendanceId`. Leader edits carry none. Undo removes that check-in's entries, and `answer` reverts to the latest remaining entry, or null if none is left.

## Considered options

- **Hard-delete Attendance and the RSVP doc on undo**: simpler, but it loses the answer the Scout had before that check-in, and it leaves no audit trail without backups.
- **A separate append-only RSVP log collection**: same information, but every read of "current answer" becomes a query. The history array is tiny (a few answers per Event per Scout).
- **Auto-ID Meetings**: allows two Meetings on one date, which no troop flow wants. It also makes "today's Meeting" a query instead of a key.

## Consequences

- `answer` is denormalized from `history` by Go. Go is the only writer, so it must keep the two consistent inside one transaction.
- Per-Scout Attendance history is a collection-group query on `attendance.scoutId` that skips undone records.
