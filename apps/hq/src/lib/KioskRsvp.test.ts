import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { Timestamp } from "firebase/firestore";
import { tick } from "svelte";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { settle } from "./KioskSession.svelte";
import Kiosk from "./Kiosk.svelte";
import { ApiError } from "./api";
import type { Event, Meeting, RSVP, RSVPAnswer } from "./contract.gen";
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
  meetingFields: Partial<Meeting> = {},
) {
  const kioskCheckIn = vi.fn(async () => ({ attendanceId: "a1" }));
  const kioskSetRsvp = vi.fn(async () => {});
  const api = fakeApi({ kioskCheckIn, kioskSetRsvp, ...apiOverrides });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  live.eventsStore.set(ready(evs));
  live.rsvpsStore.set(ready(rsvps));
  live
    .meetingStore(TODAY)
    .set(meeting(TODAY, { handoffOrder: order, ...meetingFields }));
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

// The tap guard is covered in KioskGuard.test.ts; these tests tap at once.
beforeEach(() => {
  settle.ms = 0;
});

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

  test("an asleep session while saving shows the asleep screen, not the failure message", async () => {
    const { kioskSetRsvp, session } = setup();
    kioskSetRsvp.mockRejectedValueOnce(
      new ApiError(401, { code: "session_asleep" }),
    );
    await checkInRowan();

    await click("Going");

    expect(
      await screen.findByText("Kiosk asleep, ask a Leader"),
    ).toBeInTheDocument();
    expect(screen.queryByText("Couldn't save that. Try again.")).toBeNull();
    expect(session.signOut).not.toHaveBeenCalled();
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
      await screen.findByRole("heading", { name: "Rowan's answers" }),
    ).toBeInTheDocument();
  });

  test("each card shows a quiet 'n of N' for this Scout's queue", async () => {
    setup();
    await checkInRowan();

    expect(await screen.findByText("1 of 2")).toBeInTheDocument();
    await click("Not sure"); // Fall: last chance
    expect(await screen.findByText("2 of 2")).toBeInTheDocument();
  });

  test("the summary says the Scout is checked in; Change cards show no counter", async () => {
    setup({}, events, [
      saved("fall", "s2", "notGoing"),
      saved("winter", "s2", "going"),
    ]);
    await checkInRowan();

    await screen.findByRole("heading", { name: "Rowan's answers" });
    expect(screen.getByText("You're checked in")).toBeInTheDocument();
    await click("Change Winter Campout");
    expect(screen.queryByText(/\d of \d/)).toBeNull();
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

  test("with every upcoming Event answered, still shows the summary (story 30)", async () => {
    setup({}, events, [
      saved("fall", "s2", "notGoing"),
      saved("winter", "s2", "going"),
    ]);
    await checkInRowan();

    expect(
      await screen.findByRole("heading", { name: "Rowan's answers" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeNull();
    await click("Done, pass the tablet on");
    expect(
      await screen.findByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
  });

  test("with no upcoming Events at all, goes straight to the Handoff", async () => {
    // One cancelled, one already begun.
    setup({}, [
      ev("jan", "Klondike Derby", "2027-01-23", "2027-01-23", {
        cancelled: true,
      }),
      ev("oct", "October Campout", "2026-10-16", "2026-10-18"),
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
    await screen.findByRole("heading", { name: "Rowan's answers" });
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
      await screen.findByRole("heading", { name: "Rowan's answers" }),
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

describe("Kiosk reload resumes from the Meeting doc (story 22)", () => {
  const rowanIn = { checkedIn: { s2: "a1" }, lastCheckIn: "s2" };

  test("mid-RSVP: the last Scout checked in still has pending Events, so their cards come back", async () => {
    setup({}, events, [], rowanIn);
    await tick();

    expect(
      await screen.findByRole("heading", { name: "Fall Campout" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Rowan, are you going?")).toBeInTheDocument();
  });

  test("mid-Handoff: nothing pending, so the Handoff names the next Scout after them", async () => {
    setup(
      {},
      events,
      [saved("fall", "s2", "going"), saved("winter", "s2", "notGoing")],
      rowanIn,
    );
    await tick();

    expect(
      await screen.findByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Rowan's answers" }),
    ).toBeNull();
  });

  test("a Leader undid the last check-in: no RSVP stage, the Handoff continues after them", async () => {
    setup({}, events, [], { checkedIn: {}, lastCheckIn: "s2" });
    await tick();

    expect(
      await screen.findByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
  });

  test("no check-in yet: the name list", async () => {
    setup({}, events, [], { lastCheckIn: null });
    await tick();

    expect(
      screen.getByRole("heading", { name: "Who's first?" }),
    ).toBeInTheDocument();
  });
});

describe("Kiosk RSVP tap guard", () => {
  test("a double tap on an answer doesn't also answer the next card", async () => {
    const { kioskSetRsvp } = setup();
    await checkInRowan();
    await screen.findByRole("heading", { name: "Fall Campout" });
    settle.ms = 400;
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    try {
      screen.getByRole("button", { name: "Going" }).click();
      await vi.advanceTimersByTimeAsync(0);
      await tick();
      expect(
        screen.getByRole("heading", { name: "Winter Campout" }),
      ).toBeInTheDocument();
      // The second tap of the double tap lands on the next card's Going.
      screen.getByRole("button", { name: "Going" }).click();
      await tick();
      expect(kioskSetRsvp).toHaveBeenCalledTimes(1);

      await vi.advanceTimersByTimeAsync(400);
      screen.getByRole("button", { name: "Going" }).click();
      await vi.advanceTimersByTimeAsync(0);
      expect(kioskSetRsvp).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
      settle.ms = 0;
    }
  });
});
