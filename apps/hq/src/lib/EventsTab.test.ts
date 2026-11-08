import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { Timestamp } from "firebase/firestore";
import { describe, expect, test, vi } from "vitest";
import EventsTab from "./EventsTab.svelte";
import type { Event, RSVP, RSVPAnswer, Scout } from "./contract.gen";
import type { WithId } from "./live";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, ready } from "./testing/fakeLive";

function liveWith(d: {
  events?: WithId<Event>[];
  scouts?: WithId<Scout>[];
  rsvps?: WithId<RSVP>[];
}) {
  const live = fakeLive();
  live.eventsStore.set(ready(d.events ?? []));
  live.scoutsStore.set(ready(d.scouts ?? []));
  live.rsvpsStore.set(ready(d.rsvps ?? []));
  return live;
}

const today = "2026-11-09";
const at = Timestamp.fromDate(new Date("2026-11-01T00:00:00Z"));

const scouts: WithId<Scout>[] = [
  {
    id: "s1",
    firstName: "Avery",
    lastName: "Fictional",
    active: true,
    createdAt: at,
  },
  {
    id: "s2",
    firstName: "Blake",
    lastName: "Fictional",
    active: true,
    createdAt: at,
  },
  {
    id: "s3",
    firstName: "Casey",
    lastName: "Fictional",
    active: false,
    createdAt: at,
  },
];

const events: WithId<Event>[] = [
  {
    id: "dec",
    name: "Winter Campout",
    startDate: "2026-12-11",
    endDate: "2026-12-13",
    cancelled: false,
  },
  {
    id: "nov",
    name: "Fall Campout",
    startDate: "2026-11-13",
    endDate: "2026-11-15",
    location: "Camp Fictional",
    cancelled: false,
  },
  {
    id: "oct",
    name: "October Campout",
    startDate: "2026-10-16",
    endDate: "2026-10-18",
    cancelled: false,
  },
  {
    id: "sep",
    name: "Service Day",
    startDate: "2026-09-12",
    endDate: "2026-09-12",
    cancelled: false,
  },
  {
    id: "jan",
    name: "Klondike Derby",
    startDate: "2027-01-23",
    endDate: "2027-01-23",
    cancelled: true,
  },
];

function rsvp(
  eventId: string,
  scoutId: string,
  answer: RSVPAnswer | null,
): WithId<RSVP> {
  return {
    id: `${eventId}_${scoutId}`,
    eventId,
    scoutId,
    answer,
    history: [{ answer, at, by: "u-leader", attendanceId: null }],
  };
}

function rowFor(name: string) {
  return screen
    .getByRole("link", { name: new RegExp(name) })
    .closest("li") as HTMLElement;
}

function names(list: string) {
  const section = screen.getByRole("region", { name: list });
  return within(section)
    .getAllByRole("listitem")
    .map(
      (li) => within(li).getByRole("link").querySelector(".name")?.textContent,
    );
}

describe("EventsTab", () => {
  test("splits Upcoming (soonest first) from Past (latest first)", () => {
    render(EventsTab, {
      api: fakeApi(),
      live: liveWith({ events, scouts }),
      today,
    });

    expect(names("Upcoming")).toEqual([
      "Fall Campout",
      "Winter Campout",
      "Klondike Derby",
    ]);
    expect(names("Past")).toEqual(["October Campout", "Service Day"]);
  });

  test("an Event that ends today is still Upcoming", () => {
    render(EventsTab, {
      api: fakeApi(),
      live: liveWith({ events, scouts }),
      today: "2026-11-15",
    });
    expect(names("Upcoming")).toContain("Fall Campout");
  });

  test("each row shows its dates and how many active Scouts are going", () => {
    const live = liveWith({
      events,
      scouts,
      rsvps: [
        rsvp("nov", "s1", "going"),
        rsvp("nov", "s2", "going"),
        rsvp("nov", "s3", "going"), // inactive: not counted
        rsvp("dec", "s1", "notGoing"),
        rsvp("dec", "s2", null),
      ],
    });
    render(EventsTab, { api: fakeApi(), live, today });

    const nov = rowFor("Fall Campout");
    expect(nov).toHaveTextContent("Fri 11/13 – Sun 11/15");
    expect(nov).toHaveTextContent("2 going");
    expect(rowFor("Winter Campout")).toHaveTextContent("0 going");
    expect(rowFor("Service Day")).toHaveTextContent("Sat 9/12");
  });

  test("the going count updates live", async () => {
    const live = liveWith({ events, scouts });
    render(EventsTab, { api: fakeApi(), live, today });
    expect(rowFor("Fall Campout")).toHaveTextContent("0 going");

    live.rsvpsStore.set(ready([rsvp("nov", "s1", "going")]));

    await vi.waitFor(() =>
      expect(rowFor("Fall Campout")).toHaveTextContent("1 going"),
    );
  });

  test("a cancelled Event is struck through with a Cancelled pill", () => {
    render(EventsTab, {
      api: fakeApi(),
      live: liveWith({ events, scouts }),
      today,
    });

    const jan = rowFor("Klondike Derby");
    expect(within(jan).getByText("Klondike Derby").closest("s")).not.toBeNull();
    expect(within(jan).getByText("Cancelled")).toBeInTheDocument();
    const nov = rowFor("Fall Campout");
    expect(within(nov).getByText("Fall Campout").closest("s")).toBeNull();
    expect(within(nov).queryByText("Cancelled")).not.toBeInTheDocument();
  });

  test("rows link to the Event page", () => {
    render(EventsTab, {
      api: fakeApi(),
      live: liveWith({ events, scouts }),
      today,
    });
    expect(screen.getByRole("link", { name: /Fall Campout/ })).toHaveAttribute(
      "href",
      "/events/nov",
    );
  });

  test("Add Event sends the form and clears it", async () => {
    const createEvent = vi.fn(async () => ({ id: "new" }));
    render(EventsTab, {
      api: fakeApi({ createEvent }),
      live: liveWith({ scouts }),
      today,
    });

    await userEvent.type(screen.getByLabelText("Name"), "Spring Campout");
    await userEvent.type(screen.getByLabelText("Start date"), "2027-04-16");
    await userEvent.type(screen.getByLabelText("End date"), "2027-04-18");
    await userEvent.type(
      screen.getByLabelText("Location (optional)"),
      "Lake Fictional",
    );
    await userEvent.click(screen.getByRole("button", { name: "Add Event" }));

    expect(createEvent).toHaveBeenCalledWith({
      name: "Spring Campout",
      startDate: "2027-04-16",
      endDate: "2027-04-18",
      location: "Lake Fictional",
    });
    await vi.waitFor(() =>
      expect(screen.getByLabelText("Name")).toHaveValue(""),
    );
  });

  test("Add Event refuses an end date before the start", async () => {
    const createEvent = vi.fn(async () => ({ id: "new" }));
    render(EventsTab, {
      api: fakeApi({ createEvent }),
      live: liveWith({ scouts }),
      today,
    });

    await userEvent.type(screen.getByLabelText("Name"), "Backwards Campout");
    await userEvent.type(screen.getByLabelText("Start date"), "2027-04-18");
    await userEvent.type(screen.getByLabelText("End date"), "2027-04-16");
    await userEvent.click(screen.getByRole("button", { name: "Add Event" }));

    expect(createEvent).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "The end date can't be before the start date.",
    );
  });
});
