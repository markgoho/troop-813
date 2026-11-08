// Dates as HQ shows them. Every time rule is America/New_York (spec), and
// Event dates are plain YYYY-MM-DD with no time zone.

const NY = "America/New_York";

/** Today's date in New York as YYYY-MM-DD (Go's clock.LocalDate). */
export function localDate(now: Date): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: NY,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(now);
}

const dayFormat = new Intl.DateTimeFormat("en-US", {
  timeZone: "UTC",
  weekday: "short",
  month: "numeric",
  day: "numeric",
});

/** "Fri 11/13" for 2026-11-13. */
export function shortDay(date: string): string {
  return dayFormat.format(new Date(`${date}T00:00:00Z`)).replace(",", "");
}

/** "Fri 11/13 – Sun 11/15", or "Sat 12/5" for a one-day Event. */
export function dateRange(start: string, end: string): string {
  return start === end
    ? shortDay(start)
    : `${shortDay(start)} – ${shortDay(end)}`;
}

const momentFormat = new Intl.DateTimeFormat("en-US", {
  timeZone: NY,
  weekday: "short",
  month: "numeric",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
});

/** "Mon 11/9 7:12 PM" in New York time. */
export function moment(at: Date): string {
  return momentFormat.format(at).replace(/,/g, "");
}
