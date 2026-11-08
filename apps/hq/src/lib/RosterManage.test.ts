import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import Roster from "./Roster.svelte";
import { ApiError, type Api } from "./api";
import type { RSVP } from "./contract.gen";
import type { WithId } from "./live";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, scout } from "./testing/fakeLive";

const IN_USE =
  "Has Attendance or RSVPs, so it can't be deleted. Deactivate instead.";

function setup(
  api: Partial<Api> = {},
  rsvps: WithId<RSVP>[] = [],
  extra = false,
) {
  const live = fakeLive();
  live.scoutsStore.set({
    state: "ready",
    docs: [
      scout("s1", "Avery", "Quill"),
      scout("s2", "Rowan", "Pike", false),
      ...(extra ? [scout("s3", "Quincy", "Harrow")] : []),
    ],
  });
  live.rsvpsStore.set({ state: "ready", docs: rsvps });
  render(Roster, { api: fakeApi(api), live });
}

const rsvpFor = (scoutId: string): WithId<RSVP> => ({
  id: `nov_${scoutId}`,
  eventId: "nov",
  scoutId,
  answer: "going",
  history: [],
});

const button = (name: string) => screen.getByRole("button", { name });

describe("Edit", () => {
  test("renames a Scout from an inline form", async () => {
    const editScout = vi.fn(async () => {});
    setup({ editScout });

    await userEvent.click(button("Edit Avery Quill"));
    const form = within(screen.getByRole("form", { name: "Edit Avery Quill" }));
    const first = form.getByLabelText("First name");
    await userEvent.clear(first);
    await userEvent.type(first, "Averie");
    await userEvent.click(form.getByRole("button", { name: "Save" }));

    expect(editScout).toHaveBeenCalledExactlyOnceWith("s1", {
      firstName: "Averie",
      lastName: "Quill",
    });
    expect(screen.queryByRole("form", { name: "Edit Avery Quill" })).toBeNull();
  });

  test("a name already on the Roster is refused with a message", async () => {
    const editScout = vi.fn(async () => {
      throw new ApiError(409, { code: "duplicate_scout" });
    });
    setup({ editScout });

    await userEvent.click(button("Edit Avery Quill"));
    await userEvent.click(button("Save"));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Avery Quill is already on the Roster.",
    );
  });

  test("Cancel leaves the Scout alone", async () => {
    const editScout = vi.fn();
    setup({ editScout });

    await userEvent.click(button("Edit Avery Quill"));
    await userEvent.click(button("Cancel"));

    expect(editScout).not.toHaveBeenCalled();
    expect(button("Edit Avery Quill")).toBeInTheDocument();
  });
});

describe("Deactivate / Reactivate", () => {
  test("active Scouts offer Deactivate, inactive ones Reactivate", async () => {
    const deactivateScout = vi.fn(async () => {});
    const reactivateScout = vi.fn(async () => {});
    setup({ deactivateScout, reactivateScout });

    await userEvent.click(button("Deactivate Avery Quill"));
    await userEvent.click(button("Reactivate Rowan Pike"));

    expect(deactivateScout).toHaveBeenCalledExactlyOnceWith("s1");
    expect(reactivateScout).toHaveBeenCalledExactlyOnceWith("s2");
  });

  test("Reactivate refused for an active namesake says so", async () => {
    const reactivateScout = vi.fn(async () => {
      throw new ApiError(409, { code: "duplicate_scout" });
    });
    setup({ reactivateScout });

    await userEvent.click(button("Reactivate Rowan Pike"));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Rowan Pike is already on the Roster.",
    );
  });
});

describe("Add Scout: duplicate of an inactive Scout", () => {
  test("offers Reactivate for the inactive match", async () => {
    const addScout = vi.fn(async () => {
      throw new ApiError(409, {
        code: "duplicate_scout",
        inactiveMatchId: "s2",
      });
    });
    const reactivateScout = vi.fn(async () => {});
    setup({ addScout, reactivateScout });

    await userEvent.type(screen.getByLabelText("First name"), "Rowan");
    await userEvent.type(screen.getByLabelText("Last name"), "Pike{Enter}");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Rowan Pike is on the Roster as an inactive Scout.",
    );
    const offers = screen.getAllByRole("button", {
      name: "Reactivate Rowan Pike",
    });
    // The offer sits next to the message; the list row has its own button too.
    await userEvent.click(offers[0] as HTMLElement);

    expect(reactivateScout).toHaveBeenCalledExactlyOnceWith("s2");
    await vi.waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(screen.getByLabelText("First name")).toHaveValue("");
  });
});

describe("Delete", () => {
  test("is shown but disabled, with the reason, while the Scout has RSVPs", () => {
    setup({}, [rsvpFor("s1")]);

    const del = button("Delete Avery Quill");
    expect(del).toBeDisabled();
    expect(del).toHaveAccessibleDescription(IN_USE);
  });

  test("deletes a Scout with no history after a confirm", async () => {
    const deleteScout = vi.fn(async () => {});
    setup({ deleteScout }, [rsvpFor("s1")], true);

    await userEvent.click(button("Delete Quincy Harrow"));
    expect(deleteScout).not.toHaveBeenCalled();
    await userEvent.click(button("Yes, delete Quincy Harrow"));

    expect(deleteScout).toHaveBeenCalledExactlyOnceWith("s3");
  });

  test("a 409 in_use (Attendance the page can't see) disables it with the reason", async () => {
    const deleteScout = vi.fn(async () => {
      throw new ApiError(409, { code: "in_use" });
    });
    setup({ deleteScout });

    await userEvent.click(button("Delete Avery Quill"));
    await userEvent.click(button("Yes, delete Avery Quill"));

    await vi.waitFor(() => expect(button("Delete Avery Quill")).toBeDisabled());
    expect(button("Delete Avery Quill")).toHaveAccessibleDescription(IN_USE);
  });
});
