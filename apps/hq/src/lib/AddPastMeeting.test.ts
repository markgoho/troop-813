import { fireEvent, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import AddPastMeeting from "./AddPastMeeting.svelte";
import { ApiError } from "./api";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, meeting, ready } from "./testing/fakeLive";

const TODAY = "2026-11-09";

function setup(overrides: Parameters<typeof fakeApi>[0] = {}) {
  const api = fakeApi(overrides);
  const live = fakeLive();
  const m = meeting("2026-11-02");
  live.meetingsStore.set(ready(m.state === "ready" && m.doc ? [m.doc] : []));
  const onCreated = vi.fn();
  render(AddPastMeeting, { api, live, today: TODAY, onCreated });
  return { onCreated };
}

const picker = () => screen.getByLabelText("Meeting date");
const pick = (value: string) =>
  fireEvent.input(picker(), { target: { value } });

describe("AddPastMeeting", () => {
  test("defaults to the latest Monday with no Meeting", async () => {
    setup();

    expect(await screen.findByLabelText("Meeting date")).toHaveValue(
      "2026-10-26",
    );
  });

  test("adds the Meeting and reports its date", async () => {
    const createPastMeeting = vi.fn(async (date: string) => ({ id: date }));
    const { onCreated } = setup({ createPastMeeting });
    await screen.findByLabelText("Meeting date");

    await userEvent.click(screen.getByRole("button", { name: "Add Meeting" }));

    expect(createPastMeeting).toHaveBeenCalledWith("2026-10-26");
    await vi.waitFor(() =>
      expect(onCreated).toHaveBeenCalledWith("2026-10-26"),
    );
  });

  test.each([
    ["today", "2026-11-09", "That's today."],
    ["a future date", "2026-11-16", "That date is in the future."],
    ["a date with a Meeting", "2026-11-02", "There is already a Meeting"],
  ])("refuses %s without calling the API", async (_name, date, message) => {
    const createPastMeeting = vi.fn();
    setup({ createPastMeeting });
    await screen.findByLabelText("Meeting date");

    await pick(date);

    expect(await screen.findByRole("alert")).toHaveTextContent(message);
    expect(screen.getByRole("button", { name: "Add Meeting" })).toBeDisabled();
    expect(createPastMeeting).not.toHaveBeenCalled();
  });

  test("shows the server's refusal when a Meeting appeared meanwhile", async () => {
    const { onCreated } = setup({
      createPastMeeting: async () => {
        throw new ApiError(409, { code: "meeting_exists" });
      },
    });
    await screen.findByLabelText("Meeting date");

    await userEvent.click(screen.getByRole("button", { name: "Add Meeting" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "already a Meeting",
    );
    expect(onCreated).not.toHaveBeenCalled();
  });
});
