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

const DAY_MS = 86_400_000;

/** `date` (YYYY-MM-DD) moved by `days` calendar days, as YYYY-MM-DD. */
export function addDays(date: string, days: number): string {
  const t = Date.parse(`${date}T00:00:00Z`) + days * DAY_MS;
  return new Date(t).toISOString().slice(0, 10);
}

/** Whether `date` (YYYY-MM-DD) is a Monday: the Troop's Meeting night. */
export function isMonday(date: string): boolean {
  return new Date(`${date}T00:00:00Z`).getUTCDay() === 1;
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

const clockFormat = new Intl.DateTimeFormat("en-US", {
  timeZone: NY,
  hour: "numeric",
  minute: "2-digit",
});

/** "7:12 PM" in New York time. */
export function clockTime(at: Date): string {
  return clockFormat.format(at);
}

/** "Mon 11/9 7:12 PM" in New York time. */
export function moment(at: Date): string {
  return momentFormat.format(at).replace(/,/g, "");
}

const wallFormat = new Intl.DateTimeFormat("en-US", {
  timeZone: NY,
  hourCycle: "h23",
  year: "numeric",
  month: "numeric",
  day: "numeric",
  hour: "numeric",
  minute: "numeric",
  second: "numeric",
});

/** 22:00 in New York on a Meeting's date (YYYY-MM-DD): when its Kiosk session ends (Go's clock.MeetingEnds). */
export function meetingEnds(meetingId: string): Date {
  const wall = new Date(`${meetingId}T22:00:00Z`); // 22:00 read as if UTC
  const parts = Object.fromEntries(
    wallFormat.formatToParts(wall).map((p) => [p.type, Number(p.value)]),
  );
  // What New York's clock reads at `wall`, as UTC, minus `wall` = New York's offset.
  const shown = Date.UTC(
    parts.year ?? 0,
    (parts.month ?? 1) - 1,
    parts.day ?? 1,
    parts.hour,
    parts.minute,
    parts.second,
  );
  return new Date(wall.getTime() - (shown - wall.getTime()));
}
