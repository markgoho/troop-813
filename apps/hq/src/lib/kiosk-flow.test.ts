import { describe, expect, test } from "vitest";
import { Timestamp } from "firebase/firestore";
import type { Event, RSVP, RSVPAnswer } from "./contract.gen";
import type { WithId } from "./live";
import {
  asleep,
  isLastChance,
  meetingOver,
  nameList,
  nextScout,
  pendingEvents,
  upcomingEvents,
  wasNext,
} from "./kiosk-flow";

const state = (
  over: Partial<{
    handoffOrder: string[];
    checkedIn: Record<string, string>;
    notFound: string[];
  }> = {},
) => ({
  handoffOrder: ["a", "b", "c", "d"],
  checkedIn: {},
  notFound: [],
  ...over,
});

const all = new Set(["a", "b", "c", "d"]);

describe("nextScout", () => {
  test("is the next in handoffOrder after the latest check-in", () => {
    expect(nextScout(state({ checkedIn: { b: "x" } }), "b", all)).toBe("c");
  });

  test("starts at the top when there is no latest check-in", () => {
    expect(nextScout(state(), null, all)).toBe("a");
  });

  test("skips Scouts already checked in", () => {
    expect(nextScout(state({ checkedIn: { a: "1", b: "2" } }), "a", all)).toBe(
      "c",
    );
  });

  test("skips Scouts marked not found", () => {
    const m = state({ checkedIn: { a: "1" }, notFound: ["b"] });
    expect(nextScout(m, "a", all)).toBe("c");
  });

  test("wraps at the end of the order", () => {
    const m = state({ checkedIn: { b: "1", c: "2", d: "3" } });
    expect(nextScout(m, "d", all)).toBe("a");
  });

  test("is null when everyone is checked in or not found", () => {
    const m = state({ checkedIn: { a: "1", b: "2" }, notFound: ["c", "d"] });
    expect(nextScout(m, "b", all)).toBeNull();
  });

  test("a latest Scout not in the order starts from the top", () => {
    expect(nextScout(state({ checkedIn: { z: "1" } }), "z", all)).toBe("a");
  });

  test("skips Scouts who are no longer active, even ones still in the order", () => {
    const active = new Set(["a", "d"]);
    expect(nextScout(state(), "a", active)).toBe("d");
    expect(nextScout(state(), "d", active)).toBe("a");
  });

  test("continues from the right place after a Scout who has since gone inactive", () => {
    // b went inactive after being checked in; the order still knows where b was.
    const active = new Set(["a", "c", "d"]);
    expect(nextScout(state({ checkedIn: { b: "1" } }), "b", active)).toBe("c");
  });

  test("is null when no active Scout is left", () => {
    expect(nextScout(state(), "a", new Set())).toBeNull();
  });
});

describe("wasNext", () => {
  test("is the Scout offered at the Handoff while still waiting", () => {
    expect(wasNext("b", state({ checkedIn: { a: "1" } }))).toBe("b");
  });

  test("is null once that Scout has checked in", () => {
    expect(wasNext("b", state({ checkedIn: { b: "1" } }))).toBeNull();
  });

  test("is null once that Scout is marked not found", () => {
    expect(wasNext("b", state({ notFound: ["b"] }))).toBeNull();
  });

  test("is null when nobody was offered", () => {
    expect(wasNext(null, state())).toBeNull();
  });
});
import { scout } from "./testing/fakeLive";

describe("nameList", () => {
  test("first names only, sorted, active Scouts only", () => {
    const list = nameList([
      scout("s1", "Rowan", "Quill"),
      scout("s2", "Juniper", "Marsh"),
      scout("s3", "Ellis", "Fenwick", false),
    ]);

    expect(list).toEqual([
      { id: "s2", label: "Juniper" },
      { id: "s1", label: "Rowan" },
    ]);
  });

  test("adds the last initial only when two active Scouts share a first name", () => {
    const list = nameList([
      scout("s1", "Avery", "Quill"),
      scout("s2", "Avery", "Marsh"),
      scout("s3", "Rowan", "Quill"),
    ]);

    expect(list).toEqual([
      { id: "s2", label: "Avery M." },
      { id: "s1", label: "Avery Q." },
      { id: "s3", label: "Rowan" },
    ]);
  });

  test("the match ignores case and spaces", () => {
    const list = nameList([
      scout("s1", "avery ", "Quill"),
      scout("s2", "Avery", "marsh"),
    ]);

    expect(list.map((n) => n.label)).toEqual(["Avery m.", "avery Q."]);
  });

  test("an inactive Scout sharing a first name adds no initial", () => {
    const list = nameList([
      scout("s1", "Avery", "Quill"),
      scout("s2", "Avery", "Marsh", false),
    ]);

    expect(list).toEqual([{ id: "s1", label: "Avery" }]);
  });
});

const event = (
  id: string,
  startDate: string,
  over: Partial<Event> = {},
): WithId<Event> => ({
  id,
  name: id,
  startDate,
  endDate: startDate,
  cancelled: false,
  ...over,
});

const saved = (eventId: string, answer: RSVPAnswer | null): WithId<RSVP> => ({
  id: `${eventId}_s1`,
  eventId,
  scoutId: "s1",
  answer,
  history: [
    {
      answer,
      at: Timestamp.fromDate(new Date("2026-11-01T00:00:00Z")),
      by: "u-leader",
      attendanceId: null,
    },
  ],
});

const MEETING = "2026-11-09";

describe("pendingEvents", () => {
  test("are Events not cancelled that start after the Meeting date, soonest first", () => {
    const events = [
      event("later", "2026-12-11"),
      event("soon", "2026-11-13"),
      event("cancelled", "2026-11-20", { cancelled: true }),
      event("today", MEETING),
      event("past", "2026-10-16"),
    ];

    expect(pendingEvents(events, [], MEETING).map((e) => e.id)).toEqual([
      "soon",
      "later",
    ]);
  });

  test("leave out Events with a saved answer, but a cleared one is asked again", () => {
    const events = [
      event("a", "2026-11-13"),
      event("b", "2026-11-20"),
      event("c", "2026-11-27"),
    ];
    const rsvps = [
      saved("a", "going"),
      saved("b", "notGoing"),
      saved("c", null),
    ];

    expect(pendingEvents(events, rsvps, MEETING).map((e) => e.id)).toEqual([
      "c",
    ]);
  });
});

describe("upcomingEvents", () => {
  test("include answered Events too, but never cancelled or begun ones", () => {
    const events = [
      event("a", "2026-11-13"),
      event("x", "2026-11-20", { cancelled: true }),
      event("p", MEETING),
    ];

    expect(upcomingEvents(events, MEETING).map((e) => e.id)).toEqual(["a"]);
  });
});

describe("isLastChance", () => {
  test("is true exactly 7 days out, and false at 8", () => {
    expect(isLastChance("2026-11-16", MEETING)).toBe(true);
    expect(isLastChance("2026-11-17", MEETING)).toBe(false);
  });

  test("is true for a weekend Event this week", () => {
    expect(isLastChance("2026-11-13", MEETING)).toBe(true);
  });

  test("counts days across a month and year end", () => {
    expect(isLastChance("2027-01-03", "2026-12-27")).toBe(true);
    expect(isLastChance("2027-01-04", "2026-12-27")).toBe(false);
  });
});

describe("asleep", () => {
  const last = new Date("2026-11-10T00:00:00Z"); // 19:00 EST on 2026-11-09
  const session = { meetingId: "2026-11-09", lastActivityAt: last };
  const at = (ms: number) => new Date(last.getTime() + ms);
  const MIN = 60_000;

  test("awake until 30 minutes after the last Kiosk activity", () => {
    expect(asleep(session, at(0))).toBe(false);
    expect(asleep(session, at(30 * MIN - 1))).toBe(false);
  });

  test("asleep from exactly 30 minutes", () => {
    expect(asleep(session, at(30 * MIN))).toBe(true);
    expect(asleep(session, at(31 * MIN))).toBe(true);
  });

  test("asleep from 22:00 New York on the Meeting date, however recent the activity", () => {
    const recent = {
      ...session,
      lastActivityAt: new Date("2026-11-10T02:50:00Z"),
    }; // 21:50 EST
    expect(asleep(recent, new Date("2026-11-10T02:59:59Z"))).toBe(false);
    expect(asleep(recent, new Date("2026-11-10T03:00:00Z"))).toBe(true);
  });

  test("22:00 follows daylight time", () => {
    const summer = {
      meetingId: "2026-07-01",
      lastActivityAt: new Date("2026-07-02T01:50:00Z"), // 21:50 EDT
    };
    expect(asleep(summer, new Date("2026-07-02T01:59:59Z"))).toBe(false);
    expect(asleep(summer, new Date("2026-07-02T02:00:00Z"))).toBe(true);
  });
});

describe("meetingOver", () => {
  test("true from 22:00 New York on the Meeting date", () => {
    expect(meetingOver("2026-11-09", new Date("2026-11-10T02:59:59Z"))).toBe(
      false,
    );
    expect(meetingOver("2026-11-09", new Date("2026-11-10T03:00:00Z"))).toBe(
      true,
    );
  });
});
