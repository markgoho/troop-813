import { render, screen } from "@testing-library/svelte";
import { tick } from "svelte";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import Kiosk from "./Kiosk.svelte";
import { settle } from "./KioskSession.svelte";
import type { SignedInUser } from "./session";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, meeting, ready, scout } from "./testing/fakeLive";
import { fakeSession } from "./testing/fakeSession";

const TODAY = "2026-11-09";
const kioskUser: SignedInUser = {
  uid: "kiosk-sess-1",
  kiosk: { sessionId: "sess-1", meetingId: TODAY },
};

describe("Kiosk tap guard", () => {
  beforeEach(() => {
    settle.ms = 400;
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  });
  afterEach(() => {
    vi.useRealTimers();
    settle.ms = 400;
  });

  async function setup() {
    const kioskCheckIn = vi.fn(async () => ({ attendanceId: "a1" }));
    const live = fakeLive();
    live.scoutsStore.set(
      ready([scout("s1", "Rowan", "Fenwick"), scout("s2", "Avery", "Quill")]),
    );
    live.eventsStore.set(ready([]));
    live.rsvpsStore.set(ready([]));
    const store = live.meetingStore(TODAY);
    store.set(meeting(TODAY, { handoffOrder: ["s1", "s2"] }));
    render(Kiosk, {
      session: fakeSession(kioskUser, {}),
      api: fakeApi({ kioskCheckIn }),
      live,
      now: () => new Date("2026-11-10T00:05:00Z"),
    });
    await tick();
    await vi.advanceTimersByTimeAsync(400);
    return { kioskCheckIn, store };
  }

  const click = (name: string) => screen.getByRole("button", { name }).click();

  test("a tap right after a new screen is ignored; one after the window works", async () => {
    const { kioskCheckIn } = await setup();

    click("Rowan");
    await tick();
    // The laggy second tap lands on the confirm screen's Yes: swallowed.
    click("Yes, I'm Rowan");
    await tick();
    expect(kioskCheckIn).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(400);
    click("Yes, I'm Rowan");
    await tick();
    expect(kioskCheckIn).toHaveBeenCalledOnce();
  });

  test("the Handoff's full-width Yes ignores a tap meant for the screen before", async () => {
    const { kioskCheckIn, store } = await setup();
    click("Rowan");
    await vi.advanceTimersByTimeAsync(400);
    click("Yes, I'm Rowan");
    await vi.advanceTimersByTimeAsync(0);
    store.set(
      meeting(TODAY, { handoffOrder: ["s1", "s2"], checkedIn: { s1: "a1" } }),
    );
    await tick();
    expect(
      screen.getByRole("heading", { name: "Pass the tablet to Avery" }),
    ).toBeInTheDocument();
    kioskCheckIn.mockClear();

    click("Yes");
    await tick();
    expect(kioskCheckIn).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(400);
    click("Yes");
    await tick();
    expect(kioskCheckIn).toHaveBeenCalledWith({
      scoutId: "s2",
      method: "handoff",
    });
  });
});
