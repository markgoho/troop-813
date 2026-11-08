// Pure helpers for the Meetings tab (spec stories 75–80). Dates are plain
// YYYY-MM-DD New York days, the Meeting IDs.
import { addDays, isMonday, shortDay } from "./dates";

/** The latest Monday before `today` that has no Meeting: what "Add a past Meeting" proposes. */
export function defaultPastDate(today: string, existing: Set<string>): string {
  let d = addDays(today, -1);
  while (!isMonday(d) || existing.has(d)) d = addDays(d, -1);
  return d;
}

/** Why `date` can't be added as a past Meeting, or null. Go enforces the same rules. */
export function pastDateError(
  date: string,
  today: string,
  existing: Set<string>,
): string | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return "Pick a date.";
  if (date === today)
    return "That's today. The Admin starts today's Meeting on the Kiosk.";
  if (date > today) return "That date is in the future.";
  if (existing.has(date))
    return `There is already a Meeting on ${shortDay(date)}.`;
  return null;
}
