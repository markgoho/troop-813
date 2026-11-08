// Keeps the Kiosk tablet's screen on (spec story 40). The browser drops a
// screen wake lock whenever the page is hidden, so it is re-acquired each
// time the page becomes visible again. Browsers without the API just dim.

/** The bits of navigator / document this needs, so tests can pass fakes. */
export interface WakeLockHost {
  navigator: { wakeLock?: { request(type: "screen"): Promise<unknown> } };
  document: Pick<
    Document,
    "visibilityState" | "addEventListener" | "removeEventListener"
  >;
}

/** Holds a screen wake lock until the returned function is called. */
export function holdScreenAwake(
  host: WakeLockHost = { navigator, document },
): () => void {
  const wakeLock = host.navigator.wakeLock;
  if (!wakeLock) return () => {};
  let lock: { release?: () => Promise<void> } | null = null;
  let stopped = false;

  const acquire = () => {
    if (stopped || host.document.visibilityState !== "visible") return;
    wakeLock.request("screen").then(
      (l) => {
        lock = l as typeof lock;
        if (stopped) void lock?.release?.();
      },
      // Refused (battery saver, not visible yet): the next visibilitychange retries.
      () => {},
    );
  };
  const onVisibility = () => acquire();

  host.document.addEventListener("visibilitychange", onVisibility);
  acquire();
  return () => {
    stopped = true;
    host.document.removeEventListener("visibilitychange", onVisibility);
    void lock?.release?.();
  };
}
