import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import MeetingPage from "./MeetingPage.svelte";
import { ApiError } from "./api";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, meeting, ready, scout } from "./testing/fakeLive";

const TODAY = "2026-11-09";
const PAST = "2026-11-02";

const roster = [
  scout("s1", "Avery", "Quill"),
  scout("s2", "Rowan", "Fenwick"),
  scout("s3", "Juniper", "Marsh", false),
  scout("s4", "Ellis", "Thorne", false),
];

function setup(
  checkedIn: Record<string, string> | null,
  overrides: Parameters<typeof fakeApi>[0] = {},
  meetingId = PAST,
) {
  const api = fakeApi({
    rsvpRollbackPreview: async () => ({ items: [] }),
    ...overrides,
  });
  const live = fakeLive();
  live.scoutsStore.set(ready(roster));
  live
    .meetingStore(meetingId)
    .set(
      checkedIn === null
        ? { state: "ready", doc: null }
        : meeting(meetingId, { checkedIn }),
    );
  const onDeleted = vi.fn();
  render(MeetingPage, { api, live, meetingId, today: TODAY, onDeleted });
  return { onDeleted };
}

const box = (name: string) => screen.findByRole("checkbox", { name });

describe("MeetingPage checklist", () => {
  test("lists active Scouts by name, ticked when present; names link to the Scout", async () => {
    setup({ s2: "a2" });

    expect(await box("Avery Quill")).not.toBeChecked();
    expect(await box("Rowan Fenwick")).toBeChecked();
    expect(screen.getByRole("link", { name: "Avery Quill" })).toHaveAttribute(
      "href",
      "/roster/s1",
    );
    expect(screen.getByText("1 present")).toBeInTheDocument();
  });

  test("an inactive Scout appears only when they were present", async () => {
    setup({ s3: "a3" });

    expect(await box("Juniper Marsh")).toBeChecked();
    expect(
      screen.queryByRole("checkbox", { name: "Ellis Thorne" }),
    ).not.toBeInTheDocument();
  });

  test("ticking a Scout checks them in as a Leader", async () => {
    const leaderCheckIn = vi.fn(async () => ({ attendanceId: "a9" }));
    setup({}, { leaderCheckIn });

    await userEvent.click(await box("Avery Quill"));

    expect(leaderCheckIn).toHaveBeenCalledWith(PAST, "s1");
    expect(await screen.findByText("Avery Quill checked in")).toHaveClass(
      "sr-only",
    );
  });

  test("the undo confirm is a labelled group; Yes announces and returns focus to the checkbox", async () => {
    setup({ s1: "a1" }, { undoCheckIn: async () => {} });
    await userEvent.click(await box("Avery Quill"));
    expect(
      await screen.findByRole("group", {
        name: "Undo Avery Quill's check-in?",
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));
    expect(
      await screen.findByText("Avery Quill's check-in undone"),
    ).toHaveClass("sr-only");
    await waitFor(() =>
      expect(
        screen.getByRole("checkbox", { name: "Avery Quill" }),
      ).toHaveFocus(),
    );
  });

  test("unticking asks first, then undoes that Scout's current check-in", async () => {
    const undoCheckIn = vi.fn(async () => {});
    setup({ s1: "a1" }, { undoCheckIn });

    await userEvent.click(await box("Avery Quill"));

    expect(
      await screen.findByText("Undo Avery Quill's check-in?"),
    ).toBeVisible();
    expect(undoCheckIn).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));
    expect(undoCheckIn).toHaveBeenCalledWith(PAST, "a1");
  });

  test("the unmark confirm lists the RSVPs it rolls back", async () => {
    const rsvpRollbackPreview = vi.fn(async () => ({
      items: [
        {
          eventId: "fall",
          eventName: "Fall Campout",
          from: "going" as const,
          to: null,
        },
      ],
    }));
    setup({ s1: "a1" }, { rsvpRollbackPreview });

    await userEvent.click(await box("Avery Quill"));

    expect(rsvpRollbackPreview).toHaveBeenCalledExactlyOnceWith(PAST, "a1");
    const list = await screen.findByRole("list", {
      name: "RSVPs this rolls back",
    });
    expect(
      within(list)
        .getAllByRole("listitem")
        .map((li) => li.textContent),
    ).toEqual(["Fall Campout: Going → No answer"]);
  });

  test("if the rollback can't be previewed, the confirm still works and says so", async () => {
    const undoCheckIn = vi.fn(async () => {});
    setup(
      { s1: "a1" },
      {
        undoCheckIn,
        rsvpRollbackPreview: async () => {
          throw new ApiError(500, null);
        },
      },
    );

    await userEvent.click(await box("Avery Quill"));

    expect(
      await screen.findByText(/Couldn't load which RSVPs/),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Yes, undo" }));
    expect(undoCheckIn).toHaveBeenCalledExactlyOnceWith(PAST, "a1");
  });

  test("Cancel leaves them present", async () => {
    const undoCheckIn = vi.fn();
    setup({ s1: "a1" }, { undoCheckIn });

    await userEvent.click(await box("Avery Quill"));
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(undoCheckIn).not.toHaveBeenCalled();
    expect(screen.queryByText(/Undo Avery/)).not.toBeInTheDocument();
    expect(await box("Avery Quill")).toBeChecked();
  });

  test("a failed tick says so", async () => {
    setup(
      {},
      {
        leaderCheckIn: async () => {
          throw new ApiError(500, { code: "internal" });
        },
      },
    );

    await userEvent.click(await box("Avery Quill"));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Couldn't check in Avery Quill",
    );
  });

  test("a Meeting that doesn't exist says so", async () => {
    setup(null);

    expect(await screen.findByText("No Meeting on Mon 11/2.")).toBeVisible();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });
});

describe("MeetingPage delete", () => {
  test("Delete this Meeting shows only while nobody is present", async () => {
    setup({ s1: "a1" });
    await box("Avery Quill");

    expect(
      screen.queryByRole("button", { name: "Delete this Meeting" }),
    ).not.toBeInTheDocument();
  });

  test("it confirms, deletes and reports back", async () => {
    const deleteMeeting = vi.fn(async () => {});
    const { onDeleted } = setup({}, { deleteMeeting });

    await userEvent.click(
      await screen.findByRole("button", { name: "Delete this Meeting" }),
    );
    expect(deleteMeeting).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, delete this Meeting" }),
    );

    expect(deleteMeeting).toHaveBeenCalledWith(PAST);
    await vi.waitFor(() => expect(onDeleted).toHaveBeenCalled());
  });

  test("someone ticked meanwhile: the server's refusal is shown", async () => {
    setup(
      {},
      {
        deleteMeeting: async () => {
          throw new ApiError(409, { code: "in_use" });
        },
      },
    );

    await userEvent.click(
      await screen.findByRole("button", { name: "Delete this Meeting" }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Yes, delete this Meeting" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Someone is present",
    );
  });

  test("tonight's Meeting can't be deleted from here", async () => {
    setup({}, {}, TODAY);
    await box("Avery Quill");

    expect(
      screen.queryByRole("button", { name: "Delete this Meeting" }),
    ).not.toBeInTheDocument();
    expect(
      within(document.body).getByRole("link", { name: "Open Tonight" }),
    ).toHaveAttribute("href", "/");
  });
});
