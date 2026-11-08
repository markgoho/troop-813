// kiosk-flow (spec #17 Modules): the Meeting Kiosk's decisions as pure
// functions of (Meeting doc, Roster, Events, RSVPs, now). Screens only render
// what these return; Go never recomputes them.
//
// So far: the name list, the Handoff and asleep. #26 adds pendingEvents /
// isLastChance.
import type { Scout } from "./contract.gen";
import { meetingEnds } from "./dates";
import type { WithId } from "./live";

/** One row of the Kiosk name list. */
export interface NameListEntry {
  id: string;
  label: string;
}

/** The slice of the Meeting doc the Handoff decisions read. */
export interface HandoffState {
  handoffOrder: string[];
  checkedIn: Record<string, string>;
  notFound: string[];
}

/**
 * Who the Kiosk names next (story 13): the next in handoffOrder after the
 * Scout who just checked in (`after`), skipping anyone checked in or not
 * found, wrapping at the end. null `after` (or one not in the order) starts at
 * the top. null result when nobody is left.
 */
export function nextScout(
  m: HandoffState,
  after: string | null,
): string | null {
  const n = m.handoffOrder.length;
  const start = after === null ? -1 : m.handoffOrder.indexOf(after);
  for (let step = 1; step <= n; step++) {
    const id = m.handoffOrder[(start + step) % n];
    if (id !== undefined && !(id in m.checkedIn) && !m.notFound.includes(id)) {
      return id;
    }
  }
  return null;
}

/**
 * The name list's "<Scout> was next" fallback: the Scout the Handoff offered
 * (`offered`) when the chain broke, while still waiting. A Scout who has
 * checked in, or was marked not found, is no longer due.
 */
export function wasNext(
  offered: string | null,
  m: HandoffState,
): string | null {
  if (
    offered === null ||
    offered in m.checkedIn ||
    m.notFound.includes(offered)
  ) {
    return null;
  }
  return offered;
}

const key = (s: string) => s.trim().toLowerCase();

/**
 * The name list: active Scouts by first name, with the last initial added
 * only when two active Scouts share a first name (story 10). Sorted by label.
 */
export function nameList(scouts: WithId<Scout>[]): NameListEntry[] {
  const active = scouts.filter((s) => s.active);
  const count = new Map<string, number>();
  for (const s of active) {
    count.set(key(s.firstName), (count.get(key(s.firstName)) ?? 0) + 1);
  }
  return active
    .map((s) => {
      const first = s.firstName.trim();
      const shared = (count.get(key(s.firstName)) ?? 0) > 1;
      const initial = s.lastName.trim().charAt(0);
      return { id: s.id, label: shared ? `${first} ${initial}.` : first };
    })
    .sort((a, b) => a.label.localeCompare(b.label));
}

/** How long a Kiosk session stays awake without a Kiosk write or a Wake (ADR 0003; Go's auth.KioskIdle). */
export const KIOSK_IDLE_MS = 30 * 60 * 1000;

/** The slice of kioskSessions/{id} the sleep decision reads. */
export interface SleepState {
  meetingId: string;
  lastActivityAt: Date;
}

/** True from 22:00 New York on the Meeting's date: no Wake can revive the Kiosk. */
export function meetingOver(meetingId: string, now: Date): boolean {
  return now.getTime() >= meetingEnds(meetingId).getTime();
}

/**
 * Whether the Kiosk is asleep (stories 37, 38): 30 minutes since its last
 * write or Wake, or past 22:00 on the Meeting date. The tablet runs this on
 * its own timer; Go enforces the same rule on every call.
 */
export function asleep(session: SleepState, now: Date): boolean {
  return (
    now.getTime() >= session.lastActivityAt.getTime() + KIOSK_IDLE_MS ||
    meetingOver(session.meetingId, now)
  );
}
