import { render, screen } from "@testing-library/svelte";
import { describe, expect, test } from "vitest";
import HealthStatus from "./HealthStatus.svelte";
import type { Api } from "./api";

function fakeApi(health: Api["health"]): Api {
  return { health };
}

describe("HealthStatus", () => {
  test("shows the API status and today's date from Go", async () => {
    render(HealthStatus, {
      api: fakeApi(async () => ({ status: "ok", today: "2026-11-09" })),
    });

    expect(await screen.findByText("API ok")).toBeInTheDocument();
    expect(screen.getByText("Today is 2026-11-09")).toBeInTheDocument();
  });

  test("says the API is unreachable when the call fails", async () => {
    render(HealthStatus, {
      api: fakeApi(async () => {
        throw new Error("network down");
      }),
    });

    expect(await screen.findByText("API unreachable")).toBeInTheDocument();
  });
});
