import { describe, expect, test } from "vitest";
import { defaultPastDate, pastDateError } from "./meetings";

describe("defaultPastDate", () => {
  test("is the latest Monday before today with no Meeting", () => {
    // Today is Mon 11/9; Mon 11/2 has a Meeting, 10/26 doesn't.
    expect(defaultPastDate("2026-11-09", new Set(["2026-11-02"]))).toBe(
      "2026-10-26",
    );
  });

  test("on a Monday with no Meeting yet, still skips today", () => {
    expect(defaultPastDate("2026-11-09", new Set())).toBe("2026-11-02");
  });

  test("midweek it is the Monday just gone", () => {
    expect(defaultPastDate("2026-11-12", new Set())).toBe("2026-11-09");
  });

  test("skips Mondays that have a Meeting, ignores non-Monday Meetings", () => {
    expect(
      defaultPastDate(
        "2026-11-12",
        new Set(["2026-11-09", "2026-11-03", "2026-11-02"]),
      ),
    ).toBe("2026-10-26");
  });
});

describe("pastDateError", () => {
  const today = "2026-11-09";
  const existing = new Set(["2026-11-02"]);

  test("accepts a past day with no Meeting", () => {
    expect(pastDateError("2026-10-26", today, existing)).toBeNull();
  });

  test.each([
    ["", "Pick a date."],
    [
      "2026-11-09",
      "That's today. The Admin starts today's Meeting on the Kiosk.",
    ],
    ["2026-11-16", "That date is in the future."],
    ["2026-11-02", "There is already a Meeting on Mon 11/2."],
  ])("refuses %j", (date, message) => {
    expect(pastDateError(date, today, existing)).toBe(message);
  });
});
