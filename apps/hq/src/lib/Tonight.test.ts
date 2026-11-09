import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { Timestamp } from "firebase/firestore";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { RSVP } from "./contract.gen";
import type { WithId } from "./live";
import Tonight from "./Tonight.svelte";
import { ApiError } from "./api";
import { fakeApi } from "./testing/fakeApi";
import {
  attendanceDoc,
  fakeLive,
  kioskSession,
  meeting,
  ready,
  scout,
} from "./testing/fakeLive";

// Monday 2026-11-09, 19:30 in New York.
const now = () => new Date("2026-11-10T00:30:00Z");
const TODAY = "2026-11-09";

const roster = [
  scout("s1", "Avery", "Quill"),
  scout("s2", "Rowan", "Fenwick"),
  scout("s3", "Ellis", "Thorne"),
  scout("s4", "Juniper", "Marsh", false),
];

function setup(
  overrides: Parameters<typeof fakeApi>[0] = {},
  clock: () => Date = now,
) {
  const api = fakeApi({
    leaders: async () => ({
      leaders: [{ uid: "u-leader", displayName: "Lee Leader" }],
    }),
    rsvpRollbackPreview: async () => ({ items: [] }),
    ...overrides,
  });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  render(Tonight, { api, live, now: clock });
  return { api, live };
}

function tonight(
  live: ReturnType<typeof fakeLive>,
  checkedIn: Record<string, string>,
  attendance = [
    attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    attendanceDoc("a2", "s2", "leader", "2026-11-10T00:20:00Z", "u-leader"),
  ],
) {
  live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn }));
  live.attendanceStore(TODAY).set(ready(attendance));
}

function rsvpDoc(
  eventId: string,
  scoutId: string,
  answer: "going" | "notGoing",
  attendanceId: string | null,
): WithId<RSVP> {
  return {
    id: `${eventId}_${scoutId}`,
    eventId,
    scoutId,
    answer,
    history: [
      {
        answer,
        at: Timestamp.fromDate(new Date("2026-11-10T00:06:00Z")),
        by: "kiosk-sess-1",
        attendanceId,
      },
    ],
  };
}

const items = (name: string) =>
  within(screen.getByRole("list", { name }))
    .queryAllByRole("listitem")
    .map((li) => li.textContent?.replace(/\s+/g, " ").trim());

describe("Tonight", () => {
  test("with no Meeting today it says the Admin starts it on the Kiosk", async () => {
    const { live } = setup();
    live.meetingStore(TODAY).set({ state: "ready", doc: null });

    expect(
      await screen.findByText(
        "No Meeting today: the Admin starts it on the Kiosk",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Add a past Meeting" }),
    ).toHaveAttribute("href", "/meetings/new");
  });

  test("shows checked-in and not-yet counts (active Scouts only for not-yet)", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1", s2: "a2" });

    expect(await screen.findByText("2 checked in")).toBeInTheDocument();
    expect(screen.getByText("1 not yet")).toBeInTheDocument();
  });

  test("Checked in lists newest first with time, method and which Leader", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1", s2: "a2" });

    await screen.findByText("2 checked in");
    await waitFor(() =>
      expect(items("Checked in")).toEqual([
        "Rowan Fenwick 7:20 PM Leader: Lee Leader Undo",
        "Avery Quill 7:05 PM Name list Undo",
      ]),
    );
  });

  test("names in both lists link to the Scout's page", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1", s2: "a2" });

    expect(
      await screen.findByRole("link", { name: "Avery Quill" }),
    ).toHaveAttribute("href", "/roster/s1");
    expect(screen.getByRole("link", { name: "Ellis Thorne" })).toHaveAttribute(
      "href",
      "/roster/s3",
    );
  });

  test("a Handoff check-in says Handoff", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "handoff", "2026-11-10T00:05:00Z"),
    ]);

    await screen.findByText("1 checked in");
    expect(items("Checked in")).toEqual(["Avery Quill 7:05 PM Handoff Undo"]);
  });

  test("Not checked in lists active Scouts not yet in, with a Check in button", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1" });

    await screen.findByText("1 checked in");
    expect(items("Not checked in")).toEqual([
      "Ellis Thorne Check in",
      "Rowan Fenwick Check in",
    ]);
  });

  test("Not checked in labels Scouts the Kiosk couldn't find", async () => {
    const { live } = setup();
    live
      .meetingStore(TODAY)
      .set(meeting(TODAY, { checkedIn: { s1: "a1" }, notFound: ["s3"] }));
    live
      .attendanceStore(TODAY)
      .set(
        ready([attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z")]),
      );

    await screen.findByText("1 checked in");
    expect(items("Not checked in")).toEqual([
      "Ellis Thorne Not found at the Kiosk Check in",
      "Rowan Fenwick Check in",
    ]);
  });

  test("Check in calls the API as a Leader for that Scout", async () => {
    const leaderCheckIn = vi.fn(async () => ({ attendanceId: "a9" }));
    const { live } = setup({ leaderCheckIn });
    tonight(live, { s1: "a1" });

    await userEvent.click(
      await screen.findByRole("button", { name: "Check in Ellis Thorne" }),
    );

    expect(leaderCheckIn).toHaveBeenCalledExactlyOnceWith(TODAY, "s3");
  });

  test("sets the page title", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1" });
    await screen.findByRole("button", { name: "Check in Ellis Thorne" });
    expect(document.title).toBe("Tonight · Troop 813 HQ");
  });

  test("announces a Leader check-in and its undo for screen readers", async () => {
    const { live } = setup({
      leaderCheckIn: vi.fn(async () => ({ attendanceId: "a9" })),
      undoCheckIn: vi.fn(async () => {}),
    });
    tonight(live, { s1: "a1", s2: "a2" });

    await userEvent.click(
      await screen.findByRole("button", { name: "Check in Ellis Thorne" }),
    );
    expect(await screen.findByText("Ellis Thorne checked in")).toHaveClass(
      "sr-only",
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Undo Avery Quill" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));
    expect(
      await screen.findByText("Avery Quill's check-in undone"),
    ).toBeInTheDocument();
  });

  test("announces the Find a Scout result", async () => {
    const { live } = setup();
    tonight(live, {});
    await userEvent.type(await screen.findByLabelText("Find a Scout"), "e");
    expect(await screen.findByText("3 Scouts match")).toHaveClass("sr-only");
    await userEvent.type(screen.getByLabelText("Find a Scout"), "zzz");
    expect(screen.getAllByText("No Scout matches “ezzz”.")).toHaveLength(2);
  });

  test("the Undo confirm takes focus, and Cancel hands it back to Undo", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1", s2: "a2" });
    await userEvent.click(
      await screen.findByRole("button", { name: "Undo Avery Quill" }),
    );
    expect(screen.getByRole("button", { name: "Yes, undo" })).toHaveFocus();
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.getByRole("button", { name: "Undo Avery Quill" }),
    ).toHaveFocus();
  });

  test("a failed Check in shows a message", async () => {
    const { live } = setup({
      leaderCheckIn: async () => {
        throw new ApiError(500, null);
      },
    });
    tonight(live, {});

    await userEvent.click(
      await screen.findByRole("button", { name: "Check in Ellis Thorne" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Couldn't check in Ellis Thorne. Try again.",
    );
  });

  test("Undo asks inline first, and only the confirm calls the API", async () => {
    const undoCheckIn = vi.fn(async () => {});
    const { live } = setup({ undoCheckIn });
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);

    await userEvent.click(
      await screen.findByRole("button", { name: "Undo Avery Quill" }),
    );
    expect(undoCheckIn).not.toHaveBeenCalled();
    expect(
      await screen.findByText("Undo Avery Quill's check-in?"),
    ).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));

    expect(undoCheckIn).toHaveBeenCalledExactlyOnceWith(TODAY, "a1");
  });

  test("the Undo confirm names the RSVP answers it will roll back", async () => {
    const rsvpRollbackPreview = vi.fn(async () => ({
      items: [
        {
          eventId: "fall",
          eventName: "Fall Campout",
          from: "going" as const,
          to: "notGoing" as const,
        },
        {
          eventId: "winter",
          eventName: "Winter Campout",
          from: "notGoing" as const,
          to: null,
        },
      ],
    }));
    const { live } = setup({ rsvpRollbackPreview });
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);

    await userEvent.click(
      await screen.findByRole("button", { name: "Undo Avery Quill" }),
    );

    expect(rsvpRollbackPreview).toHaveBeenCalledExactlyOnceWith(TODAY, "a1");
    const list = await screen.findByRole("list", {
      name: "RSVPs this rolls back",
    });
    expect(
      within(list)
        .getAllByRole("listitem")
        .map((li) => li.textContent),
    ).toEqual([
      "Fall Campout: Going → Not going",
      "Winter Campout: Not going → No answer",
    ]);
  });

  test("if the rollback can't be previewed, the confirm still works and says so", async () => {
    const undoCheckIn = vi.fn(async () => {});
    const { live } = setup({
      undoCheckIn,
      rsvpRollbackPreview: async () => {
        throw new ApiError(500, null);
      },
    });
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);

    await userEvent.click(
      await screen.findByRole("button", { name: "Undo Avery Quill" }),
    );

    expect(
      await screen.findByText(/Couldn't load which RSVPs/),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));
    expect(undoCheckIn).toHaveBeenCalledExactlyOnceWith(TODAY, "a1");
  });

  test("Checked in shows how many RSVPs each check-in gave", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1", s2: "a2" });
    live.rsvpsStore.set(
      ready([
        rsvpDoc("fall", "s1", "going", "a1"),
        rsvpDoc("winter", "s1", "notGoing", "a1"),
        // A Leader's entry and another check-in's entry don't count.
        rsvpDoc("spring", "s1", "going", null),
        rsvpDoc("fall", "s2", "going", "a-other"),
      ]),
    );

    await waitFor(() =>
      expect(items("Checked in")).toEqual([
        "Rowan Fenwick 7:20 PM Leader: Lee Leader Undo",
        "Avery Quill 7:05 PM Name list 2 RSVPs Undo",
      ]),
    );
  });

  test("Cancel on the confirm keeps the check-in", async () => {
    const undoCheckIn = vi.fn(async () => {});
    const { live } = setup({ undoCheckIn });
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);

    await userEvent.click(
      await screen.findByRole("button", { name: "Undo Avery Quill" }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "Cancel" }),
    );

    expect(undoCheckIn).not.toHaveBeenCalled();
    expect(
      screen.queryByText("Undo Avery Quill's check-in?"),
    ).not.toBeInTheDocument();
    expect(items("Checked in")).toEqual(["Avery Quill 7:05 PM Name list Undo"]);
  });

  test("an undo shows live: the Scout leaves Checked in and can be checked in again", async () => {
    const { live } = setup();
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);
    await screen.findByText("1 checked in");

    // Go's undo: soft delete plus removal from checkedIn.
    tonight(live, {}, [
      attendanceDoc(
        "a1",
        "s1",
        "nameList",
        "2026-11-10T00:05:00Z",
        "kiosk-sess-1",
        "2026-11-10T00:25:00Z",
      ),
    ]);

    await waitFor(() => expect(items("Checked in")).toEqual([]));
    expect(screen.getByText("0 checked in")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check in Avery Quill" }));
  });

  test("an undone record that is no longer current is not listed", async () => {
    const { live } = setup();
    tonight(live, { s1: "a2" }, [
      attendanceDoc(
        "a1",
        "s1",
        "nameList",
        "2026-11-10T00:05:00Z",
        "kiosk-sess-1",
        "2026-11-10T00:10:00Z",
      ),
      attendanceDoc("a2", "s1", "leader", "2026-11-10T00:15:00Z", "u-leader"),
    ]);

    await screen.findByText("1 checked in");
    await waitFor(() =>
      expect(items("Checked in")).toEqual([
        "Avery Quill 7:15 PM Leader: Lee Leader Undo",
      ]),
    );
  });
});

describe("Tonight Kiosk status", () => {
  const MIN = 60_000;
  // The Tonight tab's clock is 19:30 EST; sessions are keyed by Meeting ID there.
  const session = (
    live: ReturnType<typeof fakeLive>,
    lastActivityAt: string,
    checkedIn: Record<string, string> = {},
    attendance = [] as ReturnType<typeof attendanceDoc>[],
  ) => {
    live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn }));
    live.attendanceStore(TODAY).set(ready(attendance));
    live
      .kioskSessionStore(TODAY)
      .set(kioskSession("sess-1", TODAY, lastActivityAt));
  };

  afterEach(() => {
    vi.useRealTimers();
  });

  test("awake: shows the latest check-in time, not the session's activity time", async () => {
    const { live } = setup();
    // The session was last active at 7:25 (e.g. a not-found mark); the latest check-in was 7:12.
    session(live, "2026-11-10T00:25:00Z", { s1: "a1", s2: "a2" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
      attendanceDoc("a2", "s2", "handoff", "2026-11-10T00:12:00Z"),
      // Undone (and no longer in checkedIn): doesn't count.
      attendanceDoc(
        "a3",
        "s3",
        "handoff",
        "2026-11-10T00:20:00Z",
        "kiosk-sess-1",
        "2026-11-10T00:21:00Z",
      ),
    ]);

    expect(await screen.findByText("Kiosk awake")).toBeInTheDocument();
    expect(screen.getByText(/last check-in 7:12 PM/)).toBeInTheDocument();
    expect(screen.queryByText(/last activity/)).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Wake Kiosk" }),
    ).not.toBeInTheDocument();
  });

  test("awake with nobody checked in yet", async () => {
    const { live } = setup();
    session(live, "2026-11-10T00:12:00Z");

    expect(await screen.findByText("Kiosk awake")).toBeInTheDocument();
    expect(screen.getByText(/no check-ins yet/)).toBeInTheDocument();
  });

  test("asleep after 30 idle minutes: one Wake Kiosk button", async () => {
    const wakeKiosk = vi.fn(async () => {});
    const { live } = setup({ wakeKiosk });
    session(live, "2026-11-10T00:00:00Z"); // 19:00, 30 minutes before now

    expect(await screen.findByText("Kiosk asleep")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Wake Kiosk" }));

    expect(wakeKiosk).toHaveBeenCalledOnce();
  });

  test("goes awake by itself when the Wake lands in the session doc", async () => {
    const { live } = setup({ wakeKiosk: async () => {} });
    session(live, "2026-11-10T00:00:00Z");
    await screen.findByText("Kiosk asleep");

    live
      .kioskSessionStore(TODAY)
      .set(kioskSession("sess-1", TODAY, "2026-11-10T00:30:00Z"));

    expect(await screen.findByText("Kiosk awake")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Wake Kiosk" }),
    ).not.toBeInTheDocument();
  });

  test("falls asleep on the timer while the tab is open", async () => {
    vi.useFakeTimers();
    let at = now();
    const { live } = setup({}, () => at);
    session(live, "2026-11-10T00:05:00Z"); // 19:05; now is 19:30
    await vi.advanceTimersByTimeAsync(0);
    expect(screen.getByText("Kiosk awake")).toBeInTheDocument();

    at = new Date(at.getTime() + 5 * MIN);
    await vi.advanceTimersByTimeAsync(5 * MIN);

    // The banner, and once more through the polite region.
    expect(screen.getAllByText("Kiosk asleep")).toHaveLength(2);
  });

  test("a failed Wake says so", async () => {
    const { live } = setup({
      wakeKiosk: async () => {
        throw new ApiError(500, { code: "internal" });
      },
    });
    session(live, "2026-11-10T00:00:00Z");

    await userEvent.click(
      await screen.findByRole("button", { name: "Wake Kiosk" }),
    );

    expect(
      await screen.findByText("Couldn't wake the Kiosk. Try again."),
    ).toBeInTheDocument();
  });

  test("past 22:00 the Kiosk is done for the night: no Wake button", async () => {
    const late = () => new Date("2026-11-10T03:01:00Z"); // 22:01 EST
    const { live } = setup({}, late);
    session(live, "2026-11-10T02:58:00Z"); // 21:58, fresh

    expect(
      await screen.findByText("Kiosk closed for the night"),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Wake Kiosk" }),
    ).not.toBeInTheDocument();
  });

  test("with no live session there is no status", async () => {
    const { live } = setup();
    live.meetingStore(TODAY).set(meeting(TODAY));
    live.attendanceStore(TODAY).set(ready([]));
    live.kioskSessionStore(TODAY).set({ state: "ready", doc: null });

    await screen.findByText("0 checked in");
    expect(screen.queryByText(/Kiosk (awake|asleep)/)).not.toBeInTheDocument();
  });
});

describe("Tonight Find a Scout", () => {
  function ready3() {
    const { live } = setup();
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);
  }
  const notIn = () => screen.getByRole("list", { name: "Not checked in" });

  test("typing filters Not checked in by first or last name, any case", async () => {
    ready3();
    const box = await screen.findByRole("searchbox", { name: "Find a Scout" });
    expect(within(notIn()).getAllByRole("listitem")).toHaveLength(2);

    await userEvent.type(box, "  fen ");
    const items = within(notIn()).getAllByRole("listitem");
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveTextContent("Rowan Fenwick");

    await userEvent.clear(box);
    await userEvent.type(box, "ELL");
    expect(within(notIn()).getAllByRole("listitem")).toHaveLength(1);
    expect(within(notIn()).getByText(/Ellis/)).toBeInTheDocument();
  });

  test("no match says so; clearing restores the list", async () => {
    ready3();
    const box = await screen.findByRole("searchbox", { name: "Find a Scout" });
    await userEvent.type(box, "xy");
    // Shown, and spoken once through the polite region.
    expect(screen.getAllByText("No Scout matches “xy”.")).toHaveLength(2);
    expect(within(notIn()).queryAllByRole("listitem")).toHaveLength(0);

    await userEvent.clear(box);
    expect(screen.queryByText(/No Scout matches/)).not.toBeInTheDocument();
    expect(within(notIn()).getAllByRole("listitem")).toHaveLength(2);
  });

  test("filter leaves the Checked in list alone and Check in still works", async () => {
    const leaderCheckIn = vi.fn(async () => ({}) as never);
    const { live } = setup({ leaderCheckIn });
    tonight(live, { s1: "a1" }, [
      attendanceDoc("a1", "s1", "nameList", "2026-11-10T00:05:00Z"),
    ]);
    await userEvent.type(
      await screen.findByRole("searchbox", { name: "Find a Scout" }),
      "rowan",
    );
    expect(
      within(screen.getByRole("list", { name: "Checked in" })).getAllByRole(
        "listitem",
      ),
    ).toHaveLength(1);
    await userEvent.click(
      screen.getByRole("button", { name: "Check in Rowan Fenwick" }),
    );
    expect(leaderCheckIn).toHaveBeenCalledWith(TODAY, "s2");
  });
});
