import { render, screen, within } from "@testing-library/svelte";
import { describe, expect, test } from "vitest";
import MeetingsTab from "./MeetingsTab.svelte";
import { fakeLive, meeting, ready } from "./testing/fakeLive";
import type { Meeting } from "./contract.gen";
import type { WithId } from "./live";

const TODAY = "2026-11-09";

function doc(id: string, scoutIds: string[]): WithId<Meeting> {
  const m = meeting(id, {
    checkedIn: Object.fromEntries(scoutIds.map((s, i) => [s, `a${i}`])),
  });
  if (m.state !== "ready" || !m.doc) throw new Error("unreachable");
  return m.doc;
}

function setup(docs: WithId<Meeting>[]) {
  const live = fakeLive();
  live.meetingsStore.set(ready(docs));
  render(MeetingsTab, { live, today: TODAY });
  return live;
}

const rows = () =>
  within(screen.getByRole("list", { name: "Meetings" }))
    .queryAllByRole("listitem")
    .map((li) => li.textContent?.replace(/\s+/g, " ").trim());

describe("MeetingsTab", () => {
  test("lists Meetings newest first with a present count, tonight's marked Live", async () => {
    setup([
      doc("2026-11-02", ["s1", "s2", "s3"]),
      doc("2026-11-09", ["s1"]),
      doc("2026-10-26", []),
    ]);

    expect(await screen.findByRole("list", { name: "Meetings" })).toBeVisible();
    expect(rows()).toEqual([
      "Mon 11/9 Live 1 present",
      "Mon 11/2 3 present",
      "Mon 10/26 0 present",
    ]);
  });

  test("tonight opens Tonight, a past Meeting opens its page", async () => {
    setup([doc("2026-11-02", []), doc("2026-11-09", [])]);

    expect(await screen.findByRole("link", { name: /11\/9/ })).toHaveAttribute(
      "href",
      "/",
    );
    expect(screen.getByRole("link", { name: /11\/2/ })).toHaveAttribute(
      "href",
      "/meetings/2026-11-02",
    );
  });

  test("offers Add a past Meeting", async () => {
    setup([]);

    expect(
      await screen.findByRole("link", { name: "Add a past Meeting" }),
    ).toHaveAttribute("href", "/meetings/new");
    expect(screen.getByText("No Meetings yet.")).toBeInTheDocument();
  });
});
