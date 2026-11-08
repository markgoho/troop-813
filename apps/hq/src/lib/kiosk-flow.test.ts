import { describe, expect, test } from "vitest";
import {
  asleep,
  meetingOver,
  nameList,
  nextScout,
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

describe("nextScout", () => {
  test("is the next in handoffOrder after the latest check-in", () => {
    expect(nextScout(state({ checkedIn: { b: "x" } }), "b")).toBe("c");
  });

  test("starts at the top when there is no latest check-in", () => {
    expect(nextScout(state(), null)).toBe("a");
  });

  test("skips Scouts already checked in", () => {
    expect(nextScout(state({ checkedIn: { a: "1", b: "2" } }), "a")).toBe("c");
  });

  test("skips Scouts marked not found", () => {
    const m = state({ checkedIn: { a: "1" }, notFound: ["b"] });
    expect(nextScout(m, "a")).toBe("c");
  });

  test("wraps at the end of the order", () => {
    const m = state({ checkedIn: { b: "1", c: "2", d: "3" } });
    expect(nextScout(m, "d")).toBe("a");
  });

  test("is null when everyone is checked in or not found", () => {
    const m = state({ checkedIn: { a: "1", b: "2" }, notFound: ["c", "d"] });
    expect(nextScout(m, "b")).toBeNull();
  });

  test("a latest Scout not in the order starts from the top", () => {
    expect(nextScout(state({ checkedIn: { z: "1" } }), "z")).toBe("a");
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
