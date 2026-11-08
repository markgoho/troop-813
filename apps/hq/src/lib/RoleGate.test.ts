import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { createRawSnippet } from "svelte";
import { describe, expect, test, vi } from "vitest";
import RoleGate from "./RoleGate.svelte";
import { ApiError, type Api } from "./api";
import type { Me } from "./contract.gen";
import type { Session, SignedInUser } from "./session";
import { fakeApi as baseFakeApi } from "./testing/fakeApi";
import { fakeSession } from "./testing/fakeSession";

const NOTICE =
  "Troop 813 uses this to record Scout names, Meeting attendance and Event RSVPs. Leaders only.";

function fakeApi(me: () => Promise<Me>): Api {
  return baseFakeApi({ me });
}

const leaderScreens = createRawSnippet(() => ({
  render: () => "<p>Leader screens</p>",
}));

const signedIn: SignedInUser = { uid: "u-1" };

function renderGate(session: Session, api: Api, onKiosk = vi.fn()) {
  return render(RoleGate, { session, api, onKiosk, children: leaderScreens });
}

describe("RoleGate", () => {
  test("a Kiosk session is sent to the Kiosk without asking who it is", () => {
    const onKiosk = vi.fn();
    const me = vi.fn(async (): Promise<Me> => {
      throw new Error("not called");
    });
    renderGate(
      fakeSession({
        uid: "kiosk-s1",
        kiosk: { sessionId: "s1", meetingId: "2026-11-09" },
      }),
      fakeApi(me),
      onKiosk,
    );

    expect(onKiosk).toHaveBeenCalledOnce();
    expect(me).not.toHaveBeenCalled();
    expect(screen.queryByText("Leader screens")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Sign in with Google" }),
    ).not.toBeInTheDocument();
  });

  test("signed out: offers Google sign-in with the data notice", async () => {
    const session = fakeSession(null);
    renderGate(
      session,
      fakeApi(async () => {
        throw new Error("not called");
      }),
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Sign in with Google" }),
    );

    expect(session.signIn).toHaveBeenCalledOnce();
    expect(screen.getByText(NOTICE)).toBeInTheDocument();
    expect(screen.queryByText("Leader screens")).not.toBeInTheDocument();
  });

  test("a Parent lands on the waiting page with the notice and Sign out", async () => {
    const session = fakeSession(signedIn);
    renderGate(
      session,
      fakeApi(async () => ({ uid: "u-1", role: "parent" })),
    );

    expect(
      await screen.findByText(/ask the Admin for access/),
    ).toBeInTheDocument();
    expect(screen.getByText(NOTICE)).toBeInTheDocument();
    expect(screen.queryByText("Leader screens")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(session.signOut).toHaveBeenCalledOnce();
  });

  for (const role of ["leader", "admin"] as const) {
    test(`a${role === "admin" ? "n" : ""} ${role} gets the Leader screens`, async () => {
      renderGate(
        fakeSession(signedIn),
        fakeApi(async () => ({ uid: "u-1", role })),
      );

      expect(await screen.findByText("Leader screens")).toBeInTheDocument();
      expect(screen.queryByText(NOTICE)).not.toBeInTheDocument();
    });
  }

  test("when the API can't say who you are, says so and offers Sign out", async () => {
    renderGate(
      fakeSession(signedIn),
      fakeApi(async () => {
        throw new ApiError(500, { code: "internal" });
      }),
    );

    expect(
      await screen.findByText(
        "Couldn't reach Troop 813 HQ. Try again in a moment.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sign out" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Leader screens")).not.toBeInTheDocument();
  });
});
