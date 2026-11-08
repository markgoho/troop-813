// What the Events tab and Event page derive from the live docs.
import type { Event, RSVP, Scout } from "./contract.gen";
import type { LiveDocs, WithId } from "./live";

/** A live collection's docs, or none while loading or after an error. */
export function docsOf<T>(l: LiveDocs<T>): WithId<T>[] {
  return l.state === "ready" ? l.docs : [];
}

/**
 * Upcoming: not over yet (endDate >= today), soonest first. Past: latest
 * first. Cancelled Events stay in their list, struck through.
 */
export function splitEvents(events: WithId<Event>[], today: string) {
  const upcoming = events
    .filter((e) => e.endDate >= today)
    .sort((a, b) => a.startDate.localeCompare(b.startDate));
  const past = events
    .filter((e) => e.endDate < today)
    .sort((a, b) => b.startDate.localeCompare(a.startDate));
  return { upcoming, past };
}

/** Active Scouts by first, then last name. */
export function activeScouts(scouts: WithId<Scout>[]): WithId<Scout>[] {
  return scouts
    .filter((s) => s.active)
    .sort(
      (a, b) =>
        a.firstName.localeCompare(b.firstName) ||
        a.lastName.localeCompare(b.lastName),
    );
}

export interface Tally {
  going: number;
  notGoing: number;
  noAnswer: number;
}

/** Answers for one Event among the given (active) Scouts. */
export function tally(
  eventId: string,
  scouts: WithId<Scout>[],
  rsvps: WithId<RSVP>[],
): Tally {
  const answers = new Map(
    rsvps
      .filter((r) => r.eventId === eventId)
      .map((r) => [r.scoutId, r.answer]),
  );
  const t: Tally = { going: 0, notGoing: 0, noAnswer: 0 };
  for (const s of scouts) {
    const a = answers.get(s.id) ?? null;
    if (a === "going") t.going++;
    else if (a === "notGoing") t.notGoing++;
    else t.noAnswer++;
  }
  return t;
}

/** Problems with an Event form, in the words the form shows; null if none. Go checks the same. */
export function eventFormError(f: {
  name: string;
  startDate: string;
  endDate: string;
}): string | null {
  if (!f.name.trim()) return "Give the Event a name.";
  if (!f.startDate || !f.endDate) return "Pick a start and an end date.";
  if (f.endDate < f.startDate)
    return "The end date can't be before the start date.";
  return null;
}
