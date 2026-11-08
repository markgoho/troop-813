import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import AccessList from "./AccessList.svelte";
import type { Api } from "./api";
import type { UserSummary } from "./contract.gen";
import { fakeApi as baseFakeApi } from "./testing/fakeApi";

const users: UserSummary[] = [
  {
    uid: "u-blake",
    email: "blake.fictional@example.com",
    displayName: "Blake Fictional",
    role: "parent",
    createdAt: "2026-11-10T02:00:00Z",
  },
  {
    uid: "u-avery",
    email: "avery.fictional@example.com",
    displayName: "Avery Fictional",
    role: "leader",
    createdAt: "2026-11-10T01:00:00Z",
  },
  {
    uid: "u-admin",
    email: "admin.fictional@example.com",
    displayName: "Dana Fictional",
    role: "admin",
    createdAt: "2026-11-01T00:00:00Z",
  },
];

function fakeApi(): Api {
  return baseFakeApi({
    users: async () => ({ users: structuredClone(users) }),
    setRole: vi.fn(async () => {}),
  });
}

function row(name: string) {
  return screen.getByText(name).closest("li") as HTMLElement;
}

describe("AccessList", () => {
  test("lists everyone in the order Go returns (newest sign-in first)", async () => {
    render(AccessList, { api: fakeApi(), myUid: "u-admin" });
    await screen.findByText("Blake Fictional");
    const names = screen
      .getAllByRole("listitem")
      .map((li) => within(li).getByText(/Fictional$/).textContent);
    expect(names).toEqual([
      "Blake Fictional",
      "Avery Fictional",
      "Dana Fictional",
    ]);
  });

  test("Parents carry a Waiting pill; Leaders don't", async () => {
    render(AccessList, { api: fakeApi(), myUid: "u-admin" });
    await screen.findByText("Blake Fictional");
    expect(within(row("Blake Fictional")).getByText("Waiting")).toBeVisible();
    expect(
      within(row("Avery Fictional")).queryByText("Waiting"),
    ).not.toBeInTheDocument();
  });

  test("Make Leader promotes, Leader ✓ demotes", async () => {
    const api = fakeApi();
    render(AccessList, { api, myUid: "u-admin" });
    await screen.findByText("Blake Fictional");

    await userEvent.click(
      within(row("Blake Fictional")).getByRole("button", {
        name: "Make Leader",
      }),
    );
    expect(api.setRole).toHaveBeenCalledWith("u-blake", "leader");
    expect(
      await within(row("Blake Fictional")).findByRole("button", {
        name: "Leader ✓",
      }),
    ).toBeInTheDocument();
    expect(
      within(row("Blake Fictional")).queryByText("Waiting"),
    ).not.toBeInTheDocument();

    await userEvent.click(
      within(row("Avery Fictional")).getByRole("button", { name: "Leader ✓" }),
    );
    expect(api.setRole).toHaveBeenCalledWith("u-avery", "parent");
    expect(
      await within(row("Avery Fictional")).findByRole("button", {
        name: "Make Leader",
      }),
    ).toBeInTheDocument();
  });

  test("the Admin's own row is fixed", async () => {
    const api = fakeApi();
    render(AccessList, { api, myUid: "u-admin" });
    await screen.findByText("Dana Fictional");
    const own = within(row("Dana Fictional")).getByRole("button");
    expect(own).toBeDisabled();
    await userEvent.click(own);
    expect(api.setRole).not.toHaveBeenCalled();
  });
});
