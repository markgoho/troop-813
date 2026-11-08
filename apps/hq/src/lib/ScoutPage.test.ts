import { render, screen, waitFor, within } from "@testing-library/svelte";
import { tick } from "svelte";
import { Timestamp } from "firebase/firestore";
import { describe, expect, test, vi } from "vitest";
import ScoutPage from "./ScoutPage.svelte";
import { ApiError } from "./api";
import type { Event, RSVP, ScoutHistory } from "./contract.gen";
import type { WithId } from "./live";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, ready, scout } from "./testing/fakeLive";

const history: ScoutHistory = {
  since: "2026-10-12",
  present: 2,
  total: 3,
  meetings: [
    { date: "2026-10-26", present: true },
    { date: "2026-10-19", present: false },
    { date: "2026-10-12", present: true },
  ],
};

function event(
  id: string,
  name: string,
  startDate: string,
  endDate: string,
  cancelled = false,
): WithId<Event> {
  return { id, name, startDate, endDate, cancelled };
}

function rsvp(
  eventId: string,
  scoutId: string,
  answer: RSVP["answer"],
): WithId<RSVP> {
  const at = Timestamp.fromDate(new Date("2026-11-02T23:00:00Z"));
  return {
    id: `${eventId}_${scoutId}`,
    eventId,
    scoutId,
    answer,
    history: [{ answer, at, by: "u-leader", attendanceId: null }],
  };
}

function setup(scoutHistory = vi.fn(async () => history)) {
  const api = fakeApi({ scoutHistory });
  const live = fakeLive();
  live.scoutsStore.set(
    ready([scout("a1", "Avery", "Quill"), scout("r1", "Rowan", "Pike")]),
  );
  live.eventsStore.set(ready([]));
  live.rsvpsStore.set(ready([]));
  render(ScoutPage, { api, live, scoutId: "a1", today: "2026-11-09" });
  return { api, live, scoutHistory };
}

describe("Scout page", () => {
  test("shows the name and X/Y Meetings present since the start date", async () => {
    const { scoutHistory } = setup();

    expect(
      screen.getByRole("heading", { name: "Avery Quill" }),
    ).toBeInTheDocument();
    expect(scoutHistory).toHaveBeenCalledWith("a1");
    expect(
      await screen.findByText("2/3 Meetings present since Mon 10/12"),
    ).toBeInTheDocument();
  });

  test("lists each Meeting newest first, present or absent", async () => {
    setup();

    const list = await screen.findByRole("list", { name: "Meetings" });
    const rows = within(list)
      .getAllByRole("listitem")
      .map((li) => li.textContent?.replace(/\s+/g, " ").trim());
    expect(rows).toEqual([
      "Mon 10/26 Present",
      "Mon 10/19 Absent",
      "Mon 10/12 Present",
    ]);
  });

  test("says so when there are no Meetings yet", async () => {
    setup(
      vi.fn(async () => ({
        since: "2026-11-09",
        present: 0,
        total: 0,
        meetings: [],
      })),
    );

    expect(await screen.findByText("No Meetings yet.")).toBeInTheDocument();
  });

  test("lists upcoming Events and answered past Events with this Scout's RSVP", async () => {
    const { live } = setup();
    live.eventsStore.set(
      ready([
        event("past-answered", "Summer Camp", "2026-07-10", "2026-07-12"),
        event("past-silent", "Spring Hike", "2026-04-10", "2026-04-12"),
        event("next", "Fall Campout", "2026-11-13", "2026-11-15"),
        event("later", "Winter Campout", "2027-01-15", "2027-01-17"),
      ]),
    );
    live.rsvpsStore.set(
      ready([
        rsvp("past-answered", "a1", "going"),
        rsvp("next", "a1", "notGoing"),
        rsvp("later", "r1", "going"),
      ]),
    );
    await tick();

    const list = screen.getByRole("list", { name: "Events" });
    const rows = within(list)
      .getAllByRole("listitem")
      .map((li) => li.textContent?.replace(/\s+/g, " ").trim());
    expect(rows).toEqual([
      "Fall Campout Fri 11/13 – Sun 11/15 Not going",
      "Winter Campout Fri 1/15 – Sun 1/17 No answer",
      "Summer Camp Fri 7/10 – Sun 7/12 Going",
    ]);
    expect(screen.getByRole("link", { name: "Fall Campout" })).toHaveAttribute(
      "href",
      "/events/next",
    );
  });

  test("a cleared RSVP reads as No answer and does not make a past Event show", async () => {
    const { live } = setup();
    live.eventsStore.set(
      ready([event("past", "Summer Camp", "2026-07-10", "2026-07-12")]),
    );
    live.rsvpsStore.set(ready([rsvp("past", "a1", null)]));
    await tick();

    expect(screen.getByText("No Events.")).toBeInTheDocument();
  });

  test("a Scout who isn't on the Roster says so", async () => {
    const api = fakeApi({
      scoutHistory: vi.fn(async () => {
        throw new ApiError(404, { code: "not_found" });
      }),
    });
    const live = fakeLive();
    live.scoutsStore.set(ready([]));
    live.eventsStore.set(ready([]));
    live.rsvpsStore.set(ready([]));
    render(ScoutPage, { api, live, scoutId: "nope", today: "2026-11-09" });

    expect(
      await screen.findByText("This Scout isn't on the Roster."),
    ).toBeInTheDocument();
  });

  test("a failed history load offers no stale numbers", async () => {
    setup(
      vi.fn(async () => {
        throw new ApiError(500, { code: "internal" });
      }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Couldn't load attendance. Reload to try again.",
      ),
    );
  });
});
