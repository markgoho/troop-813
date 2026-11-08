import { render, screen } from "@testing-library/svelte";
import { describe, expect, test } from "vitest";
import LeaderTabs from "./LeaderTabs.svelte";

describe("LeaderTabs", () => {
  test("the Admin sees the Access tab", () => {
    render(LeaderTabs, { role: "admin" });
    expect(screen.getByRole("link", { name: "Access" })).toHaveAttribute(
      "href",
      "/access",
    );
  });

  test("a Leader who isn't the Admin doesn't", () => {
    render(LeaderTabs, { role: "leader" });
    expect(screen.getByRole("link", { name: "Tonight" })).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "Access" }),
    ).not.toBeInTheDocument();
  });
});
