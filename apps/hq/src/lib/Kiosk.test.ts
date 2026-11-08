import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";
import { describe, expect, test, vi } from "vitest";
import Kiosk from "./Kiosk.svelte";
import { ApiError } from "./api";
import type { Me } from "./contract.gen";
import type { SignedInUser } from "./session";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, meeting, ready, scout } from "./testing/fakeLive";
import { fakeSession } from "./testing/fakeSession";

// Monday 2026-11-09, 19:05 in New York.
const now = () => new Date("2026-11-10T00:05:00Z");
const TODAY = "2026-11-09";

const google: SignedInUser = { uid: "u-admin" };
const kioskUser: SignedInUser = {
  uid: "kiosk-sess-1",
  kiosk: { sessionId: "sess-1", meetingId: TODAY },
};

const roster = [
  scout("s1", "Avery", "Quill"),
  scout("s2", "Avery", "Marsh"),
  scout("s3", "Rowan", "Fenwick"),
  scout("s4", "Ellis", "Thorne", false),
];

function setup(
  user: SignedInUser | null,
  apiOverrides: Parameters<typeof fakeApi>[0] = {},
) {
  const session = fakeSession(user, { "kiosk-token": kioskUser });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  const api = fakeApi(apiOverrides);
  render(Kiosk, { session, api, live, now });
  return { session, live, api };
}

const asAdmin = async (): Promise<Me> => ({ uid: "u-admin", role: "admin" });

describe("Kiosk unlock", () => {
  test("signed out: offers the Admin's Google sign-in", async () => {
    const { session } = setup(null);

    await userEvent.click(
      screen.getByRole("button", { name: "Sign in with Google" }),
    );

    expect(session.signIn).toHaveBeenCalledOnce();
  });

  test("the Admin starts today's Meeting, Scouts check in from the name list", async () => {
    const startMeeting = vi.fn(async () => ({
      meetingId: TODAY,
      kioskSessionId: "sess-1",
      token: "kiosk-token",
    }));
    const kioskCheckIn = vi.fn(async () => ({ attendanceId: "a1" }));
    const { session, live } = setup(google, {
      me: asAdmin,
      startMeeting,
      kioskCheckIn,
    });

    await userEvent.click(
      await screen.findByRole("button", { name: "Start today's Meeting" }),
    );

    expect(startMeeting).toHaveBeenCalledOnce();
    expect(session.signInWithToken).toHaveBeenCalledWith("kiosk-token");
    // The Admin's Google session is replaced by the Kiosk's.
    expect(session.current()).toEqual(kioskUser);

    live.meetingStore(TODAY).set(meeting(TODAY));
    await tick();
    expect(
      screen.getByRole("heading", { name: "Who's first?" }),
    ).toBeInTheDocument();
    const names = within(screen.getByRole("list", { name: "Scouts" }));
    expect(names.getAllByRole("button").map((b) => b.textContent)).toEqual([
      "Avery M.",
      "Avery Q.",
      "Rowan",
    ]);

    await userEvent.click(names.getByRole("button", { name: "Avery Q." }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Avery Q." }),
    );

    expect(kioskCheckIn).toHaveBeenCalledWith({
      scoutId: "s1",
      method: "nameList",
    });
    // Back on the name list; the live Meeting doc greys the Scout out.
    live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn: { s1: "a1" } }));
    await tick();
    const avery = screen.getByRole("button", { name: /Avery Q\./ });
    expect(avery).toBeDisabled();
    expect(avery).toHaveTextContent("✓");
    expect(screen.getByRole("button", { name: "Rowan" })).toBeEnabled();
  });

  test("No on the confirm goes back to the name list without checking in", async () => {
    const { live } = setup(kioskUser);
    live.meetingStore(TODAY).set(meeting(TODAY));
    await tick();

    await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
    await userEvent.click(screen.getByRole("button", { name: "No" }));

    expect(screen.getByRole("list", { name: "Scouts" })).toBeInTheDocument();
  });

  for (const role of ["leader", "parent"] as const) {
    test(`a ${role} can't unlock the Kiosk`, async () => {
      const { session } = setup(google, {
        me: async () => ({ uid: "u-1", role }),
      });

      expect(
        await screen.findByText("Only the Admin can unlock the Kiosk."),
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Start today's Meeting" }),
      ).not.toBeInTheDocument();
      await userEvent.click(screen.getByRole("button", { name: "Sign out" }));
      expect(session.signOut).toHaveBeenCalledOnce();
    });
  }

  test("Start after 22:00 says it's too late", async () => {
    const { session } = setup(google, {
      me: asAdmin,
      startMeeting: async () => {
        throw new ApiError(409, { code: "too_late" });
      },
    });

    await userEvent.click(
      await screen.findByRole("button", { name: "Start today's Meeting" }),
    );

    expect(
      await screen.findByText(
        "It's past 10 PM, too late to start today's Meeting.",
      ),
    ).toBeInTheDocument();
    expect(session.signInWithToken).not.toHaveBeenCalled();
  });
});

describe("Kiosk session", () => {
  test("a reload with a live Kiosk session resumes the name list from the Meeting doc", async () => {
    const { live } = setup(kioskUser);
    live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn: { s3: "a9" } }));
    await tick();

    expect(screen.getByRole("button", { name: /Rowan/ })).toBeDisabled();
    expect(screen.queryByRole("heading", { name: "Who's first?" })).toBeNull();
    expect(
      screen.queryByRole("button", { name: "Sign in with Google" }),
    ).toBeNull();
  });

  test("a Leader's undo shows live: the Scout is selectable again", async () => {
    const { live } = setup(kioskUser);
    live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn: { s3: "a9" } }));
    await tick();
    expect(screen.getByRole("button", { name: /Rowan/ })).toBeDisabled();

    // Go's undo removes the Scout from checkedIn.
    live.meetingStore(TODAY).set(meeting(TODAY, { checkedIn: {} }));
    await tick();

    expect(screen.getByRole("button", { name: "Rowan" })).toBeEnabled();
  });

  test("a Kiosk session from an earlier date is dropped, so the Admin can unlock again", async () => {
    const { session } = setup({
      uid: "kiosk-old",
      kiosk: { sessionId: "old", meetingId: "2026-11-02" },
    });
    await tick();

    expect(session.signOut).toHaveBeenCalledOnce();
    expect(
      screen.getByRole("button", { name: "Sign in with Google" }),
    ).toBeInTheDocument();
  });

  test("a check-in on an expired session drops it, back to the unlock screen", async () => {
    const { session, live } = setup(kioskUser, {
      kioskCheckIn: async () => {
        throw new ApiError(401, { code: "session_expired" });
      },
    });
    live.meetingStore(TODAY).set(meeting(TODAY));
    await tick();

    await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Rowan" }),
    );

    expect(session.signOut).toHaveBeenCalledOnce();
    expect(
      await screen.findByRole("button", { name: "Sign in with Google" }),
    ).toBeInTheDocument();
  });

  test("a check-in on an asleep session says to ask a Leader", async () => {
    const { session, live } = setup(kioskUser, {
      kioskCheckIn: async () => {
        throw new ApiError(401, { code: "session_asleep" });
      },
    });
    live.meetingStore(TODAY).set(meeting(TODAY));
    await tick();

    await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Rowan" }),
    );

    expect(
      await screen.findByText("Kiosk asleep, ask a Leader"),
    ).toBeInTheDocument();
    expect(session.signOut).not.toHaveBeenCalled();
  });

  test("a failed check-in stays on the confirm and says so", async () => {
    const { live } = setup(kioskUser, {
      kioskCheckIn: async () => {
        throw new ApiError(500, { code: "internal" });
      },
    });
    live.meetingStore(TODAY).set(meeting(TODAY));
    await tick();

    await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Rowan" }),
    );

    expect(
      await screen.findByText("Couldn't check you in. Try again."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Yes, I'm Rowan" }),
    ).toBeEnabled();
  });
});
