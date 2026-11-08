import { describe, expect, test } from "vitest";
import { nameList, nextScout, wasNext } from "./kiosk-flow";

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
