// kiosk-flow (spec #17 Modules): the Meeting Kiosk's decisions as pure
// functions of (Meeting doc, Roster, Events, RSVPs, now). Screens only render
// what these return; Go never recomputes them.
//
// So far: the name list. Later tickets add nextScout and the was-next
// fallback (#24), pendingEvents / isLastChance (#26) and asleep (#27).
import type { Scout } from "./contract.gen";
import type { WithId } from "./live";

/** One row of the Kiosk name list. */
export interface NameListEntry {
  id: string;
  label: string;
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
