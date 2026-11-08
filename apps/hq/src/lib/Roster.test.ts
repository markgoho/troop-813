import { render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";
import { describe, expect, test, vi } from "vitest";
import Roster from "./Roster.svelte";
import { ApiError } from "./api";
import { fakeApi } from "./testing/fakeApi";
import { fakeLive, scout } from "./testing/fakeLive";

function setup(addScout = vi.fn(async () => ({ id: "new-id" }))) {
  const api = fakeApi({ addScout });
  const live = fakeLive();
  render(Roster, { api, live });
  return {
    addScout,
    live,
    first: screen.getByLabelText("First name"),
    last: screen.getByLabelText("Last name"),
  };
}

function names(section: string) {
  const list = screen.getByRole("list", { name: section });
  return within(list)
    .queryAllByRole("listitem")
    .map((li) => li.querySelector(".name")?.textContent?.trim());
}

describe("Add Scout rapid entry", () => {
  test("Enter in last name submits, then the form clears and focus returns to first name", async () => {
    const { addScout, first, last } = setup();

    await userEvent.type(first, "Avery");
    await userEvent.type(last, "Quill{Enter}");

    expect(addScout).toHaveBeenCalledExactlyOnceWith({
      firstName: "Avery",
      lastName: "Quill",
    });
    await waitFor(() => expect(first).toHaveFocus());
    expect(first).toHaveValue("");
    expect(last).toHaveValue("");
  });

  test("Scouts can be entered back to back from the keyboard", async () => {
    const { addScout, first } = setup();

    await userEvent.type(first, "Avery{Tab}Quill{Enter}");
    await waitFor(() => expect(first).toHaveFocus());
    await userEvent.keyboard("Rowan{Tab}Pike{Enter}");

    await waitFor(() => expect(addScout).toHaveBeenCalledTimes(2));
    expect(addScout).toHaveBeenLastCalledWith({
      firstName: "Rowan",
      lastName: "Pike",
    });
  });

  test("a duplicate is blocked with a message and the typed names are kept", async () => {
    const { first, last } = setup(
      vi.fn(async () => {
        throw new ApiError(409, { code: "duplicate_scout" });
      }),
    );

    await userEvent.type(first, "avery ");
    await userEvent.type(last, "Quill{Enter}");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "avery Quill is already on the Roster.",
    );
    expect(first).toHaveValue("avery ");
    expect(last).toHaveValue("Quill");
  });

  test("the duplicate message clears on the next successful add", async () => {
    const addScout = vi
      .fn()
      .mockRejectedValueOnce(new ApiError(409, { code: "duplicate_scout" }))
      .mockResolvedValueOnce({ id: "x" });
    const { first, last } = setup(addScout);

    await userEvent.type(first, "Avery");
    await userEvent.type(last, "Quill{Enter}");
    await screen.findByRole("alert");
    await userEvent.clear(last);
    await userEvent.type(last, "Marsh{Enter}");

    await waitFor(() => expect(first).toHaveValue(""));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("blank names aren't sent", async () => {
    const { addScout, first, last } = setup();

    await userEvent.type(first, "  ");
    await userEvent.type(last, "Quill{Enter}");

    expect(addScout).not.toHaveBeenCalled();
  });

  test("any other failure says the Scout wasn't added", async () => {
    const { first, last } = setup(
      vi.fn(async () => {
        throw new ApiError(500, { code: "internal" });
      }),
    );

    await userEvent.type(first, "Avery");
    await userEvent.type(last, "Quill{Enter}");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Couldn't add Avery Quill. Try again.",
    );
  });
});

describe("Roster list", () => {
  test("lists Active Scouts, then Inactive, each by first then last name", async () => {
    const { live } = setup();
    live.scoutsStore.set({
      state: "ready",
      docs: [
        scout("s1", "Rowan", "Pike"),
        scout("s2", "Avery", "Quill"),
        scout("s3", "Quincy", "Harrow", false),
        scout("s4", "Avery", "Marsh"),
      ],
    });
    await tick();

    expect(names("Active Scouts")).toEqual([
      "Avery Marsh",
      "Avery Quill",
      "Rowan Pike",
    ]);
    expect(names("Inactive Scouts")).toEqual(["Quincy Harrow"]);
  });

  test("updates live as the Roster changes", async () => {
    const { live } = setup();
    live.scoutsStore.set({ state: "ready", docs: [] });
    await tick();
    expect(screen.getByText("No Scouts yet.")).toBeInTheDocument();

    live.scoutsStore.set({
      state: "ready",
      docs: [scout("s1", "Avery", "Quill")],
    });
    await tick();

    expect(await screen.findByText("Avery Quill")).toBeInTheDocument();
    expect(screen.queryByText("No Scouts yet.")).not.toBeInTheDocument();
  });

  test("says when the Roster can't be read", async () => {
    const { live } = setup();
    live.scoutsStore.set({ state: "error", error: new Error("denied") });
    await tick();

    expect(
      screen.getByText("Couldn't load the Roster. Reload to try again."),
    ).toBeInTheDocument();
  });
});
