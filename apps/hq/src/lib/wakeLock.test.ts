import { describe, expect, test, vi } from "vitest";
import { holdScreenAwake, type WakeLockHost } from "./wakeLock";

function fakeHost(visible = true) {
  const doc = new EventTarget() as EventTarget & {
    visibilityState: DocumentVisibilityState;
  };
  doc.visibilityState = visible ? "visible" : "hidden";
  const release = vi.fn(async () => {});
  const request = vi.fn(async () => ({ release }));
  const host = {
    navigator: { wakeLock: { request } },
    document: doc,
  } as unknown as WakeLockHost;
  const show = () => {
    doc.visibilityState = "visible";
    doc.dispatchEvent(new Event("visibilitychange"));
  };
  const hide = () => {
    doc.visibilityState = "hidden";
    doc.dispatchEvent(new Event("visibilitychange"));
  };
  return { host, request, release, show, hide };
}

describe("holdScreenAwake", () => {
  test("requests a screen lock right away when visible", () => {
    const { host, request } = fakeHost();
    holdScreenAwake(host);
    expect(request).toHaveBeenCalledExactlyOnceWith("screen");
  });

  test("re-acquires the lock each time the page becomes visible again", () => {
    const { host, request, hide, show } = fakeHost();
    holdScreenAwake(host);
    hide();
    show();
    hide();
    show();
    expect(request).toHaveBeenCalledTimes(3);
  });

  test("waits until visible when started hidden", () => {
    const { host, request, show } = fakeHost(false);
    holdScreenAwake(host);
    expect(request).not.toHaveBeenCalled();
    show();
    expect(request).toHaveBeenCalledOnce();
  });

  test("stopping releases the lock and stops re-acquiring", async () => {
    const { host, request, release, show } = fakeHost();
    const stop = holdScreenAwake(host);
    await Promise.resolve();
    await Promise.resolve();
    stop();
    show();
    expect(release).toHaveBeenCalledOnce();
    expect(request).toHaveBeenCalledOnce();
  });

  test("does nothing where the API is missing", () => {
    const { host } = fakeHost();
    const stop = holdScreenAwake({ ...host, navigator: {} });
    expect(() => stop()).not.toThrow();
  });
});
