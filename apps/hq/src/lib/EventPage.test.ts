import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { Timestamp } from "firebase/firestore";
import { describe, expect, test, vi } from "vitest";
import EventPage from "./EventPage.svelte";
import { ApiError } from "./api";
import type { Event, RSVP, RSVPEntry, Scout } from "./contract.gen";
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

const created = Timestamp.fromDate(new Date("2026-11-01T00:00:00Z"));
// Mon 11/9 7:12 PM and 8:05 PM in New York.
const kioskAt = Timestamp.fromDate(new Date("2026-11-10T00:12:00Z"));
const leaderAt = Timestamp.fromDate(new Date("2026-11-10T01:05:00Z"));

const scouts: WithId<Scout>[] = [
  {
    id: "s-blake",
    firstName: "Blake",
    lastName: "Fictional",
    active: true,
    createdAt: created,
  },
  {
    id: "s-avery",
    firstName: "Avery",
    lastName: "Fictional",
    active: true,
    createdAt: created,
  },
  {
    id: "s-casey",
    firstName: "Casey",
    lastName: "Fictional",
    active: true,
    createdAt: created,
  },
  {
    id: "s-drew",
    firstName: "Drew",
    lastName: "Former",
    active: false,
    createdAt: created,
  },
];

const campout: WithId<Event> = {
  id: "nov",
  name: "Fall Campout",
  startDate: "2026-11-13",
  endDate: "2026-11-15",
  location: "Camp Fictional",
  cancelled: false,
};

function doc(
  scoutId: string,
  history: RSVPEntry[],
  eventId = "nov",
): WithId<RSVP> {
  return {
    id: `${eventId}_${scoutId}`,
    eventId,
    scoutId,
    answer: history.at(-1)?.answer ?? null,
    history,
  };
}

const kioskGoing: RSVPEntry = {
  answer: "going",
  at: kioskAt,
  by: "kiosk-session-1",
  attendanceId: "att-1",
};
const leaderNotGoing: RSVPEntry = {
  answer: "notGoing",
  at: leaderAt,
  by: "u-leader",
  attendanceId: null,
};
const leaderCleared: RSVPEntry = {
  answer: null,
  at: leaderAt,
  by: "u-leader",
  attendanceId: null,
};

function setup(
  opts: {
    event?: WithId<Event>;
    rsvps?: WithId<RSVP>[];
    api?: Parameters<typeof fakeApi>[0];
  } = {},
) {
  const live = liveWith({
    events: [opts.event ?? campout],
    scouts,
    rsvps: opts.rsvps ?? [],
  });
  const api = fakeApi(opts.api);
  const onDeleted = vi.fn();
  render(EventPage, { api, live, eventId: "nov", onDeleted });
  return { live, api, onDeleted };
}

function scoutRow(name: string) {
  return screen.getByRole("radiogroup", { name }).closest("li") as HTMLElement;
}

describe("EventPage", () => {
  test("the page title is the Event name", () => {
    setup();
    expect(document.title).toBe("Fall Campout · Troop 813 HQ");
  });

  test("each Scout's name links to their page", () => {
    setup({ rsvps: [] });

    expect(
      screen.getByRole("link", { name: "Avery Fictional" }),
    ).toHaveAttribute("href", "/roster/s-avery");
  });

  test("shows the Event and Going / Not going / No answer counts among active Scouts", () => {
    setup({
      rsvps: [
        doc("s-avery", [kioskGoing]),
        doc("s-blake", [leaderNotGoing]),
        doc("s-drew", [kioskGoing]), // inactive: not counted
        doc("s-casey", [kioskGoing], "dec"), // another Event
      ],
    });

    expect(
      screen.getByRole("heading", { name: "Fall Campout" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fri 11/13 – Sun 11/15")).toBeInTheDocument();
    expect(screen.getByText("Camp Fictional")).toBeInTheDocument();
    const counts = screen.getByRole("list", { name: "Counts" });
    expect(counts).toHaveTextContent("Going 1");
    expect(counts).toHaveTextContent("Not going 1");
    expect(counts).toHaveTextContent("No answer 1");
  });

  test("lists every active Scout by first name, with the switch on their answer", () => {
    setup({
      rsvps: [
        doc("s-avery", [kioskGoing]),
        doc("s-blake", [kioskGoing, leaderCleared]),
      ],
    });

    const groups = screen
      .getAllByRole("radiogroup")
      .map((g) => g.getAttribute("aria-label"));
    expect(groups).toEqual([
      "Avery Fictional",
      "Blake Fictional",
      "Casey Fictional",
    ]);
    expect(
      within(scoutRow("Avery Fictional")).getByRole("radio", { name: "Going" }),
    ).toBeChecked();
    expect(
      within(scoutRow("Blake Fictional")).getByRole("radio", {
        name: "No answer",
      }),
    ).toBeChecked();
    expect(
      within(scoutRow("Casey Fictional")).getByRole("radio", {
        name: "No answer",
      }),
    ).toBeChecked();
  });

  test("the switch sets, changes and clears an RSVP", async () => {
    const setRsvp = vi.fn(async () => {});
    setup({ rsvps: [doc("s-avery", [kioskGoing])], api: { setRsvp } });

    await userEvent.click(
      within(scoutRow("Casey Fictional")).getByRole("radio", { name: "Going" }),
    );
    await userEvent.click(
      within(scoutRow("Avery Fictional")).getByRole("radio", {
        name: "Not going",
      }),
    );
    await userEvent.click(
      within(scoutRow("Avery Fictional")).getByRole("radio", {
        name: "No answer",
      }),
    );

    expect(setRsvp.mock.calls).toEqual([
      ["nov", "s-casey", "going"],
      ["nov", "s-avery", "notGoing"],
      ["nov", "s-avery", null],
    ]);
  });

  test("a failed change puts the switch back and says so", async () => {
    const setRsvp = vi.fn(async () => {
      throw new ApiError(500, { code: "internal" });
    });
    setup({ rsvps: [doc("s-avery", [kioskGoing])], api: { setRsvp } });

    await userEvent.click(
      within(scoutRow("Avery Fictional")).getByRole("radio", {
        name: "Not going",
      }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Couldn't save Avery's RSVP. Try again.",
    );
    expect(
      within(scoutRow("Avery Fictional")).getByRole("radio", { name: "Going" }),
    ).toBeChecked();
  });

  test("each row says who answered last and when, with the full history on tap", async () => {
    setup({
      rsvps: [
        doc("s-avery", [kioskGoing, leaderNotGoing]),
        doc("s-blake", [kioskGoing]),
      ],
    });

    const avery = scoutRow("Avery Fictional");
    expect(avery).toHaveTextContent("Leader · Mon 11/9 8:05 PM");
    expect(scoutRow("Blake Fictional")).toHaveTextContent(
      "Kiosk · Mon 11/9 7:12 PM",
    );
    expect(scoutRow("Casey Fictional")).not.toHaveTextContent("·");

    await userEvent.click(within(avery).getByText("Leader · Mon 11/9 8:05 PM"));
    const history = within(avery).getByRole("list", { name: "History" });
    expect(
      within(history)
        .getAllByRole("listitem")
        .map((li) => li.textContent),
    ).toEqual([
      "Going, Kiosk · Mon 11/9 7:12 PM",
      "Not going, Leader · Mon 11/9 8:05 PM",
    ]);
  });

  test("a clear shows as cleared", () => {
    setup({ rsvps: [doc("s-avery", [kioskGoing, leaderCleared])] });
    expect(scoutRow("Avery Fictional")).toHaveTextContent(
      "Cleared by Leader · Mon 11/9 8:05 PM",
    );
  });

  test("Delete is disabled with the reason while the Event has RSVPs", () => {
    setup({ rsvps: [doc("s-avery", [kioskGoing, leaderCleared])] });

    const del = screen.getByRole("button", { name: "Delete Event" });
    expect(del).toBeDisabled();
    expect(del).toHaveAccessibleDescription(
      "Scouts have RSVP history for this Event, so it can't be deleted. Cancel it instead.",
    );
  });

  test("the delete confirm takes focus; Keep it returns it to Delete Event", async () => {
    setup();
    await userEvent.click(screen.getByRole("button", { name: "Delete Event" }));
    expect(screen.getByRole("button", { name: "Keep it" })).toHaveFocus();
    expect(
      screen.getByRole("group", { name: "Delete Fall Campout?" }),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Keep it" }));
    expect(screen.getByRole("button", { name: "Delete Event" })).toHaveFocus();
  });

  test("Delete works on an Event nobody has answered, after a confirm", async () => {
    const deleteEvent = vi.fn(async () => {});
    const { onDeleted } = setup({
      rsvps: [doc("s-avery", [kioskGoing], "dec")],
      api: { deleteEvent },
    });

    await userEvent.click(screen.getByRole("button", { name: "Delete Event" }));
    expect(deleteEvent).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, delete Fall Campout" }),
    );

    expect(deleteEvent).toHaveBeenCalledWith("nov");
    await vi.waitFor(() => expect(onDeleted).toHaveBeenCalledOnce());
  });

  test("Cancel and Uncancel; a cancelled Event is struck through with a Cancelled pill", async () => {
    const setEventCancelled = vi.fn(async () => {});
    const { live } = setup({ api: { setEventCancelled } });

    await userEvent.click(screen.getByRole("button", { name: "Cancel Event" }));
    expect(setEventCancelled).toHaveBeenCalledWith("nov", true);

    live.eventsStore.set(ready([{ ...campout, cancelled: true }]));
    const heading = await screen.findByRole("heading", {
      name: /Fall Campout/,
    });
    await vi.waitFor(() =>
      expect(
        within(heading).getByText("Fall Campout").closest("s"),
      ).not.toBeNull(),
    );
    expect(within(heading).getByText("Cancelled")).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: "Uncancel Event" }),
    );
    expect(setEventCancelled).toHaveBeenLastCalledWith("nov", false);
  });

  test("Edit saves the changed fields", async () => {
    const updateEvent = vi.fn(async () => {});
    setup({ api: { updateEvent } });

    await userEvent.click(screen.getByRole("button", { name: "Edit" }));
    const name = screen.getByLabelText("Name");
    expect(name).toHaveValue("Fall Campout");
    await userEvent.clear(name);
    await userEvent.type(name, "Fall Camporee");
    await userEvent.clear(screen.getByLabelText("Location (optional)"));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(updateEvent).toHaveBeenCalledWith("nov", {
      name: "Fall Camporee",
      startDate: "2026-11-13",
      endDate: "2026-11-15",
    });
    await vi.waitFor(() =>
      expect(screen.queryByLabelText("Name")).not.toBeInTheDocument(),
    );
  });

  test("a deleted Event says so", () => {
    const live = liveWith({ events: [], scouts });
    render(EventPage, {
      api: fakeApi(),
      live,
      eventId: "nov",
      onDeleted: vi.fn(),
    });
    expect(
      screen.getByText("This Event doesn't exist (it may have been deleted)."),
    ).toBeInTheDocument();
  });
});
