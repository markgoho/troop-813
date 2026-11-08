import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import Tonight from "./Tonight.svelte";
import { ApiError } from "./api";
import { fakeApi } from "./testing/fakeApi";
import {
  attendanceDoc,
  fakeLive,
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

function setup(overrides: Parameters<typeof fakeApi>[0] = {}) {
  const api = fakeApi({
    leaders: async () => ({
      leaders: [{ uid: "u-leader", displayName: "Lee Leader" }],
    }),
    ...overrides,
  });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  render(Tonight, { api, live, now });
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
    expect(screen.getByText("Undo Avery Quill's check-in?")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));

    expect(undoCheckIn).toHaveBeenCalledExactlyOnceWith(TODAY, "a1");
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
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

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
