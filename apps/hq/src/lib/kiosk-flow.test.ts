import { describe, expect, test } from "vitest";
import { nameList } from "./kiosk-flow";
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
