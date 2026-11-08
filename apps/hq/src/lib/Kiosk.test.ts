import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";
import { afterEach, describe, expect, test, vi } from "vitest";
import Kiosk from "./Kiosk.svelte";
import { ApiError } from "./api";
import type { Me } from "./contract.gen";
import type { SignedInUser } from "./session";
import { fakeApi } from "./testing/fakeApi";
import {
  fakeLive,
  kioskSession,
  meeting,
  ready,
  scout,
} from "./testing/fakeLive";
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
  clock: () => Date = now,
) {
  const session = fakeSession(user, { "kiosk-token": kioskUser });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  // No Events to RSVP to: a check-in goes straight to the Handoff (see KioskRsvp.test.ts).
  live.eventsStore.set(ready([]));
  live.rsvpsStore.set(ready([]));
  const api = fakeApi(apiOverrides);
  render(Kiosk, { session, api, live, now: clock });
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

describe("Kiosk Handoff", () => {
  // Order: Rowan (s3), Avery Q. (s1), Avery M. (s2).
  const order = ["s3", "s1", "s2"];

  /** Rowan checks in by name list; returns the pieces the tests drive. */
  async function afterRowan(apiOverrides: Parameters<typeof fakeApi>[0] = {}) {
    const kioskCheckIn = vi.fn(async () => ({ attendanceId: "a1" }));
    const kioskNotFound = vi.fn(async () => {});
    const ctx = setup(kioskUser, {
      kioskCheckIn,
      kioskNotFound,
      ...apiOverrides,
    });
    const store = ctx.live.meetingStore(TODAY);
    store.set(meeting(TODAY, { handoffOrder: order }));
    await tick();
    await userEvent.click(screen.getByRole("button", { name: "Rowan" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Rowan" }),
    );
    store.set(meeting(TODAY, { handoffOrder: order, checkedIn: { s3: "a1" } }));
    await tick();
    return { ...ctx, store, kioskCheckIn, kioskNotFound };
  }

  test("after a check-in, names the next Scout in the order", async () => {
    await afterRowan();

    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery Q." }),
    ).toBeInTheDocument();
    expect(screen.getByText("Are you Avery Q.?")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Scouts" })).toBeNull();
  });

  test("Yes checks the named Scout in by handoff, then names the next", async () => {
    const { kioskCheckIn, store } = await afterRowan();

    await userEvent.click(screen.getByRole("button", { name: "Yes" }));

    expect(kioskCheckIn).toHaveBeenLastCalledWith({
      scoutId: "s1",
      method: "handoff",
    });
    store.set(
      meeting(TODAY, {
        handoffOrder: order,
        checkedIn: { s3: "a1", s1: "a2" },
      }),
    );
    await tick();
    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery M." }),
    ).toBeInTheDocument();
  });

  test("No, I'm someone else goes to the name list and doesn't mark anyone not found", async () => {
    const { kioskNotFound } = await afterRowan();

    await userEvent.click(
      screen.getByRole("button", { name: "No, I'm someone else" }),
    );

    expect(screen.getByRole("list", { name: "Scouts" })).toBeInTheDocument();
    expect(kioskNotFound).not.toHaveBeenCalled();
    // The named Scout is still tappable, and is flagged as due.
    expect(screen.getByRole("button", { name: "Avery Q." })).toBeEnabled();
    expect(screen.getByText("Avery Q. was next")).toBeInTheDocument();
  });

  test("tapping a name after the fallback checks in by name list and hands off from there", async () => {
    const { kioskCheckIn, store } = await afterRowan();
    await userEvent.click(
      screen.getByRole("button", { name: "No, I'm someone else" }),
    );

    await userEvent.click(screen.getByRole("button", { name: "Avery M." }));
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, I'm Avery M." }),
    );

    expect(kioskCheckIn).toHaveBeenLastCalledWith({
      scoutId: "s2",
      method: "nameList",
    });
    store.set(
      meeting(TODAY, {
        handoffOrder: order,
        checkedIn: { s3: "a1", s2: "a3" },
      }),
    );
    await tick();
    // Next after Avery M. wraps to Rowan (in) → Avery Q., who was due.
    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery Q." }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Avery Q. was next")).toBeNull();
  });

  test("I can't find <Scout> marks them not found and names a different Scout", async () => {
    const { kioskNotFound } = await afterRowan();

    await userEvent.click(
      screen.getByRole("button", { name: "I can't find Avery Q." }),
    );

    expect(kioskNotFound).toHaveBeenCalledWith({ scoutId: "s1" });
    expect(
      await screen.findByRole("heading", {
        name: "Pass the tablet to Avery M.",
      }),
    ).toBeInTheDocument();
  });

  test("a not-found Scout is not named again, but can tap their own name", async () => {
    const { store } = await afterRowan();
    store.set(
      meeting(TODAY, {
        handoffOrder: order,
        checkedIn: { s3: "a1", s2: "a2" },
        notFound: ["s1"],
      }),
    );
    await tick();

    // Only the not-found Scout is left: nobody to name, so the name list.
    expect(
      screen.getByRole("heading", { name: "Tap your name" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Avery Q. was next")).toBeNull();
    expect(screen.getByRole("button", { name: "Avery Q." })).toBeEnabled();
  });

  test("with nobody left to name, shows the name list", async () => {
    const { store } = await afterRowan();
    store.set(
      meeting(TODAY, {
        handoffOrder: order,
        checkedIn: { s3: "a1", s1: "a2", s2: "a3" },
      }),
    );
    await tick();

    expect(screen.getByRole("list", { name: "Scouts" })).toBeInTheDocument();
  });

  test("a failed Yes stays on the Handoff and says so", async () => {
    const { kioskCheckIn } = await afterRowan();
    kioskCheckIn.mockRejectedValueOnce(new ApiError(500, { code: "internal" }));

    await userEvent.click(screen.getByRole("button", { name: "Yes" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Try again");
    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery Q." }),
    ).toBeInTheDocument();
  });

  test("a failed I can't find stays on the Handoff and says so", async () => {
    const { kioskNotFound } = await afterRowan();
    kioskNotFound.mockRejectedValueOnce(
      new ApiError(500, { code: "internal" }),
    );

    await userEvent.click(
      screen.getByRole("button", { name: "I can't find Avery Q." }),
    );

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery Q." }),
    ).toBeInTheDocument();
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

  test("a check-in on an asleep session shows the asleep screen", async () => {
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

describe("Kiosk sleep", () => {
  const MIN = 60_000;
  const ASLEEP = "Kiosk asleep, ask a Leader";
  // The session's last activity: 19:05 EST on TODAY (the `now` above).
  const LAST = "2026-11-10T00:05:00Z";

  afterEach(() => {
    vi.useRealTimers();
  });

  /** A Kiosk on a clock the test moves; the timer ticks in fake time. */
  function sleepSetup(lastActivityAt = LAST) {
    vi.useFakeTimers();
    let at = now();
    const ctx = setup(kioskUser, {}, () => at);
    ctx.live.meetingStore(TODAY).set(meeting(TODAY));
    ctx.live
      .kioskSessionStore("sess-1")
      .set(kioskSession("sess-1", TODAY, lastActivityAt));
    return {
      ...ctx,
      /** Moves the clock forward and lets the Kiosk's timer notice. */
      async advance(ms: number) {
        at = new Date(at.getTime() + ms);
        await vi.advanceTimersByTimeAsync(ms);
      },
    };
  }

  test("awake at 29 minutes idle, asleep at 30, with no tap", async () => {
    const { advance } = sleepSetup();
    await tick();

    await advance(29 * MIN);
    expect(screen.queryByText(ASLEEP)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rowan" })).toBeInTheDocument();

    await advance(MIN);
    expect(screen.getByText(ASLEEP)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Rowan" }),
    ).not.toBeInTheDocument();
  });

  test("a Wake (the session's lastActivityAt moves) brings the name list back by itself", async () => {
    const { advance, live } = sleepSetup();
    await tick();
    await advance(31 * MIN);
    expect(screen.getByText(ASLEEP)).toBeInTheDocument();

    live
      .kioskSessionStore("sess-1")
      .set(kioskSession("sess-1", TODAY, "2026-11-10T00:36:00Z"));
    await tick();

    expect(screen.queryByText(ASLEEP)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rowan" })).toBeInTheDocument();
  });

  test("a Kiosk that woke up mid-confirm picks up where it was", async () => {
    const { advance, live } = sleepSetup();
    await tick();
    await userEvent
      .setup({ advanceTimers: vi.advanceTimersByTime })
      .click(screen.getByRole("button", { name: "Rowan" }));
    await advance(31 * MIN);
    expect(screen.getByText(ASLEEP)).toBeInTheDocument();

    live
      .kioskSessionStore("sess-1")
      .set(kioskSession("sess-1", TODAY, "2026-11-10T00:36:00Z"));
    await tick();

    expect(
      screen.getByRole("heading", { name: "Are you Rowan?" }),
    ).toBeInTheDocument();
  });

  test("past 22:00 the Kiosk is asleep however fresh its last write", async () => {
    // Active at 21:55 EST; 22:00 is five minutes later.
    const { advance } = sleepSetup("2026-11-10T02:55:00Z");
    await tick();
    // Move the clock from 19:05 to 21:59.
    await advance(174 * MIN);
    expect(screen.queryByText(ASLEEP)).not.toBeInTheDocument();

    await advance(MIN);
    expect(screen.getByText(ASLEEP)).toBeInTheDocument();
  });

  test("the server saying asleep shows the screen, until a Wake changes lastActivityAt", async () => {
    vi.useFakeTimers();
    const kioskCheckIn = vi.fn(async () => {
      throw new ApiError(401, { code: "session_asleep" });
    });
    const { live, session } = setup(kioskUser, { kioskCheckIn });
    live.meetingStore(TODAY).set(meeting(TODAY));
    // The tablet's clock thinks it is awake (skew), the server does not.
    live.kioskSessionStore("sess-1").set(kioskSession("sess-1", TODAY, LAST));
    await tick();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    await user.click(screen.getByRole("button", { name: "Rowan" }));
    await user.click(screen.getByRole("button", { name: "Yes, I'm Rowan" }));
    expect(screen.getByText(ASLEEP)).toBeInTheDocument();
    expect(session.signOut).not.toHaveBeenCalled();

    live
      .kioskSessionStore("sess-1")
      .set(kioskSession("sess-1", TODAY, "2026-11-10T00:10:00Z"));
    await tick();
    expect(screen.queryByText(ASLEEP)).not.toBeInTheDocument();
  });

  test("a tablet left open past midnight drops back to the unlock screen on its own", async () => {
    const { advance, session } = sleepSetup();
    await tick();

    await advance(5 * 60 * MIN); // 00:05 the next day

    expect(session.signOut).toHaveBeenCalled();
  });
});
