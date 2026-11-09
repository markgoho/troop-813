<script lang="ts" module>
  /** How long taps are swallowed after the Kiosk shows a new screen. Tests set 0. */
  export const settle = { ms: 400 };
</script>

<script lang="ts">
  // A live Kiosk session (spec stories 37–39): the name list, or "Kiosk asleep,
  // ask a Leader" once the session has been idle 30 minutes or 22:00 has
  // passed. The tablet decides by its own clock (`at`, ticked by the parent)
  // over the session doc's lastActivityAt; a Wake on a Leader's phone moves
  // lastActivityAt, onSnapshot delivers it, and the list is back with nobody
  // touching the tablet.
  //
  // The name list stays mounted while asleep (just hidden), so a Scout who
  // was mid-confirm when the tablet slept finds their screen again.
  import type { Api } from "./api";
  import { asleep } from "./kiosk-flow";
  import KioskNameList from "./KioskNameList.svelte";
  import type { Live } from "./live";
  import type { KioskClaims } from "./session";

  let {
    api,
    live,
    kiosk,
    at,
    onExpired,
  }: {
    api: Api;
    live: Live;
    kiosk: KioskClaims;
    /** The tablet's clock, ticked every second by the parent. */
    at: Date;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
  } = $props();

  // The parent re-creates this component for a new session.
  // svelte-ignore state_referenced_locally
  const session = live.kioskSession(kiosk.sessionId);

  // undefined until the session doc arrives: awake, so no flash of "asleep".
  const lastActivity = $derived(
    $session.state === "ready" && $session.doc
      ? $session.doc.lastActivityAt.toDate()
      : undefined,
  );

  // Tap guard: a double or laggy tap on one screen's button lands on the
  // next screen's button in the same spot (RSVP "Done" -> Handoff "Yes") and
  // would act for the wrong Scout. For `settle.ms` after the screen changes,
  // clicks are swallowed in the capture phase, before any button sees them.
  // Nothing changes visually; keyboard Enter after the window works as usual.
  let settling = false;
  let shown: string | null = null;
  let settleTimer: ReturnType<typeof setTimeout> | undefined;
  function onScreen(screen: string) {
    if (screen === shown) return;
    const first = shown === null;
    shown = screen;
    if (first || settle.ms <= 0) return;
    settling = true;
    clearTimeout(settleTimer);
    settleTimer = setTimeout(() => (settling = false), settle.ms);
  }
  function swallowWhileSettling(e: Event) {
    if (!settling) return;
    e.stopPropagation();
    e.preventDefault();
  }

  // The server said "asleep" (401 session_asleep) while this tablet's clock
  // said awake: believe the server until lastActivityAt moves (a Wake). The
  // value is lastActivityAt then (0 if the doc hadn't arrived).
  let serverSaysAsleepFrom = $state<number | null>(null);

  const sleeping = $derived(
    (lastActivity !== undefined &&
      asleep(
        { meetingId: kiosk.meetingId, lastActivityAt: lastActivity },
        at,
      )) ||
      (serverSaysAsleepFrom !== null &&
        serverSaysAsleepFrom === (lastActivity?.getTime() ?? 0)),
  );
</script>

{#if sleeping}
  <div class="asleep">
    <svg viewBox="0 0 24 24" aria-hidden="true" class="moon">
      <path d="M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5Z" />
    </svg>
    <h1>Kiosk asleep, ask a Leader</h1>
    <p class="sub">A Leader can wake it from their phone.</p>
  </div>
{/if}
<div class="flow" hidden={sleeping} onclickcapture={swallowWhileSettling}>
  <KioskNameList
    {api}
    {live}
    {kiosk}
    {onExpired}
    {onScreen}
    onAsleep={() => (serverSaysAsleepFrom = lastActivity?.getTime() ?? 0)}
  />
</div>

<style>
  .asleep {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 1rem;
  }
  .moon {
    width: 4rem;
    height: 4rem;
    fill: none;
    stroke: var(--accent);
    stroke-width: 1.5;
    stroke-linejoin: round;
  }
  .flow {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  .flow[hidden] {
    display: none;
  }
</style>
