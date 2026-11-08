import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { Timestamp } from "firebase/firestore";
import { tick } from "svelte";
import { describe, expect, test, vi } from "vitest";
import Kiosk from "./Kiosk.svelte";
import { ApiError } from "./api";
import type { Event, RSVP, RSVPAnswer } from "./contract.gen";
import type { WithId } from "./live";
import type { SignedInUser } from "./session";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, meeting, ready, scout } from "./testing/fakeLive";
import { fakeSession } from "./testing/fakeSession";

// Monday 2026-11-09, 19:05 in New York.
const now = () => new Date("2026-11-10T00:05:00Z");
const TODAY = "2026-11-09";
const kioskUser: SignedInUser = {
  uid: "kiosk-sess-1",
  kiosk: { sessionId: "sess-1", meetingId: TODAY },
};

const roster = [scout("s1", "Avery", "Quill"), scout("s2", "Rowan", "Fenwick")];
const order = ["s2", "s1"];

const ev = (
  id: string,
  name: string,
  startDate: string,
  endDate: string,
  over: Partial<Event> = {},
): WithId<Event> => ({
  id,
  name,
  startDate,
  endDate,
  cancelled: false,
  ...over,
});

const events = [
  ev("fall", "Fall Campout", "2026-11-13", "2026-11-15"),
  ev("winter", "Winter Campout", "2026-12-11", "2026-12-13"),
  ev("jan", "Klondike Derby", "2027-01-23", "2027-01-23", { cancelled: true }),
  ev("oct", "October Campout", "2026-10-16", "2026-10-18"),
];

const saved = (
  eventId: string,
  scoutId: string,
  answer: RSVPAnswer | null,
): WithId<RSVP> => ({
  id: `${eventId}_${scoutId}`,
  eventId,
  scoutId,
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

/** A Kiosk on the name list with `evs` and `rsvps` loaded. */
function setup(
  apiOverrides: Parameters<typeof fakeApi>[0] = {},
  evs: WithId<Event>[] = events,
  rsvps: WithId<RSVP>[] = [],
) {
  const kioskCheckIn = vi.fn(async () => ({ attendanceId: "a1" }));
  const kioskSetRsvp = vi.fn(async () => {});
  const api = fakeApi({ kioskCheckIn, kioskSetRsvp, ...apiOverrides });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  live.eventsStore.set(ready(evs));
  live.rsvpsStore.set(ready(rsvps));
  live.meetingStore(TODAY).set(meeting(TODAY, { handoffOrder: order }));
  const session = fakeSession(kioskUser);
  render(Kiosk, { session, api, live, now });
  return { api, live, session, kioskCheckIn, kioskSetRsvp };
}

async function checkInRowan() {
  await tick();
  await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
  await userEvent.click(screen.getByRole("button", { name: "Yes, I'm Rowan" }));
}

const click = (name: string | RegExp) =>
  userEvent.click(screen.getByRole("button", { name }));

describe("Kiosk RSVP cards", () => {
  test("after a check-in, asks one card per pending Event, soonest first, skipping cancelled and begun Events", async () => {
    setup();
    await checkInRowan();

    expect(
      await screen.findByRole("heading", { name: "Fall Campout" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fri 11/13 – Sun 11/15")).toBeInTheDocument();
    expect(screen.queryByText("Klondike Derby")).toBeNull();
    expect(screen.queryByText("October Campout")).toBeNull();
  });

  test("Going and Not going save and move to the next card", async () => {
    const { kioskSetRsvp } = setup();
    await checkInRowan();

    await click("Going");

    expect(kioskSetRsvp).toHaveBeenCalledExactlyOnceWith({
      eventId: "fall",
      scoutId: "s2",
      answer: "going",
    });
    expect(
      await screen.findByRole("heading", { name: "Winter Campout" }),
    ).toBeInTheDocument();
    await click("Not going");
    expect(kioskSetRsvp).toHaveBeenLastCalledWith({
      eventId: "winter",
      scoutId: "s2",
      answer: "notGoing",
    });
  });

  test("Ask me next week saves nothing; the last Meeting before an Event says so and offers Not sure instead", async () => {
    const { kioskSetRsvp } = setup();
    await checkInRowan();

    // Fall starts 4 days out: the last chance.
    expect(
      screen.getByText(
        "This is the last chance: Fall Campout is this coming weekend (Fri 11/13 – Sun 11/15)",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Not sure" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Ask me next week" }),
    ).toBeNull();
    await click("Not sure");

    // Winter is a month out: plain "Ask me next week", no warning.
    expect(
      await screen.findByRole("heading", { name: "Winter Campout" }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/last chance/)).toBeNull();
    expect(screen.queryByRole("button", { name: "Not sure" })).toBeNull();
    await click("Ask me next week");

    expect(kioskSetRsvp).not.toHaveBeenCalled();
    expect(
      await screen.findByRole("heading", { name: "Your answers" }),
    ).toBeInTheDocument();
  });

  test("Events already answered are not asked again", async () => {
    setup({}, events, [saved("fall", "s2", "going")]);
    await checkInRowan();

    expect(
      await screen.findByRole("heading", { name: "Winter Campout" }),
    ).toBeInTheDocument();
  });

  test("an Event another Scout answered is still asked of this Scout", async () => {
    setup({}, events, [saved("fall", "s1", "going")]);
    await checkInRowan();

    expect(
      await screen.findByRole("heading", { name: "Fall Campout" }),
    ).toBeInTheDocument();
  });

  test("with nothing pending, goes straight to the Handoff", async () => {
    setup({}, events, [
      saved("fall", "s2", "notGoing"),
      saved("winter", "s2", "going"),
    ]);
    await checkInRowan();

    expect(
      await screen.findByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
  });

  test("a failed save stays on the card and says so", async () => {
    const { kioskSetRsvp } = setup();
    kioskSetRsvp.mockRejectedValueOnce(new ApiError(500, { code: "internal" }));
    await checkInRowan();

    await click("Going");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Couldn't save that. Try again.",
    );
    expect(
      screen.getByRole("heading", { name: "Fall Campout" }),
    ).toBeInTheDocument();
    await click("Going");
    expect(
      await screen.findByRole("heading", { name: "Winter Campout" }),
    ).toBeInTheDocument();
  });

  test("an expired session drops back to the unlock screen", async () => {
    const { kioskSetRsvp, session } = setup();
    kioskSetRsvp.mockRejectedValueOnce(
      new ApiError(401, { code: "session_expired" }),
    );
    await checkInRowan();

    await click("Going");

    await vi.waitFor(() => expect(session.signOut).toHaveBeenCalledOnce());
  });
});

describe("Kiosk RSVP summary", () => {
  async function throughCards() {
    const ctx = setup();
    await checkInRowan();
    await click("Going"); // Fall
    await click("Ask me next week"); // Winter
    await screen.findByRole("heading", { name: "Your answers" });
    return ctx;
  }

  const rows = () =>
    within(screen.getByRole("list", { name: "Upcoming Events" }))
      .getAllByRole("listitem")
      .map((li) => li.textContent?.replace(/\s+/g, " ").trim());

  test("lists every upcoming Event with the answer, or none, and a Change button", async () => {
    await throughCards();

    expect(rows()).toEqual([
      "Fall Campout Fri 11/13 – Sun 11/15 Going Change",
      "Winter Campout Fri 12/11 – Sun 12/13 No answer Change",
    ]);
  });

  test("Change opens that Event's card; the new answer is saved and shown", async () => {
    const { kioskSetRsvp } = await throughCards();

    await click("Change Winter Campout");
    expect(
      screen.getByRole("heading", { name: "Winter Campout" }),
    ).toBeInTheDocument();
    await click("Not going");

    expect(kioskSetRsvp).toHaveBeenLastCalledWith({
      eventId: "winter",
      scoutId: "s2",
      answer: "notGoing",
    });
    expect(
      await screen.findByRole("heading", { name: "Your answers" }),
    ).toBeInTheDocument();
    expect(rows()[1]).toBe(
      "Winter Campout Fri 12/11 – Sun 12/13 Not going Change",
    );
  });

  test("Change can be backed out of without saving", async () => {
    const { kioskSetRsvp } = await throughCards();
    kioskSetRsvp.mockClear();

    await click("Change Fall Campout");
    await click("Back");

    expect(kioskSetRsvp).not.toHaveBeenCalled();
    expect(rows()[0]).toContain("Going");
  });

  test("Done, pass the tablet on starts the Handoff", async () => {
    await throughCards();

    await click("Done, pass the tablet on");

    expect(
      await screen.findByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
  });
});
