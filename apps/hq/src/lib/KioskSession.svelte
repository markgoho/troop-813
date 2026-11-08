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
  <h1>Kiosk asleep, ask a Leader</h1>
  <p>A Leader can wake it from their phone.</p>
{/if}
<div hidden={sleeping}>
  <KioskNameList
    {api}
    {live}
    {kiosk}
    {onExpired}
    onAsleep={() => (serverSaysAsleepFrom = lastActivity?.getTime() ?? 0)}
  />
</div>
