import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { createRawSnippet } from "svelte";
import { describe, expect, test, vi } from "vitest";
import RoleGate from "./RoleGate.svelte";
import { ApiError, type Api } from "./api";
import type { Me } from "./contract.gen";
import type { Session, SignedInUser } from "./session";

const NOTICE =
  "Troop 813 uses this to record Scout names, Meeting attendance and Event RSVPs. Leaders only.";

function fakeSession(user: SignedInUser | null): Session {
  return {
    onChange(cb) {
      cb(user);
      return () => {};
    },
    signIn: vi.fn(async () => {}),
    signOut: vi.fn(async () => {}),
    idToken: async () => (user ? "token" : null),
  };
}

function fakeApi(me: () => Promise<Me>): Api {
  return {
    health: async () => ({ status: "ok", today: "2026-11-09" }),
    me,
  };
}

const leaderScreens = createRawSnippet(() => ({
  render: () => "<p>Leader screens</p>",
}));

const signedIn: SignedInUser = { uid: "u-1" };

function renderGate(session: Session, api: Api) {
  return render(RoleGate, { session, api, children: leaderScreens });
}

describe("RoleGate", () => {
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
