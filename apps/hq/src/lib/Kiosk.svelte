<script lang="ts">
  // The Meeting Kiosk (/kiosk, spec stories 1–12, 22, 40, 41). One route, no
  // links out, so Back is harmless; everything it shows comes from the session
  // and the live Meeting doc, so a reload or crash resumes where it was.
  //
  //   signed out             → the Admin's Google sign-in (unlock)
  //   Google sign-in         → Admin: "Start today's Meeting"; anyone else: refused
  //   Kiosk token for today  → the name list, or "Kiosk asleep" (KioskSession)
  //   Kiosk token, other day → dropped (signed out), back to unlock
  //
  // The Google sign-in is offered only while no Kiosk session for today is
  // live: a Scout holding the tablet must never get the Admin's account chooser.
  import { ApiError, type Api } from "./api";
  import { localDate } from "./dates";
  import KioskSession from "./KioskSession.svelte";
  import type { Live } from "./live";
  import type { Session, SignedInUser } from "./session";
  import { holdScreenAwake } from "./wakeLock";

  let {
    session,
    api,
    live,
    now = () => new Date(),
  }: { session: Session; api: Api; live: Live; now?: () => Date } = $props();

  // undefined until the session reports in (Firebase restores it asynchronously).
  let user = $state<SignedInUser | null | undefined>(undefined);
  $effect(() =>
    session.onChange((u) => {
      user = u;
    }),
  );

  // The tablet's clock, ticked once a second: a time-based expiry fires no
  // onSnapshot, so sleep (KioskSession) and the date check below run on this.
  // svelte-ignore state_referenced_locally
  let at = $state(now());
  $effect(() => {
    const timer = setInterval(() => {
      at = now();
    }, 1000);
    return () => clearInterval(timer);
  });

  // A Kiosk session from an earlier date ended at 22:00 that night, even on a
  // tablet left open past midnight.
  const stale = $derived(
    user?.kiosk !== undefined && user.kiosk.meetingId !== localDate(at),
  );
  $effect(() => {
    if (stale) void session.signOut();
  });

  const me = $derived(user && !user.kiosk ? api.me() : null);

  $effect(() => holdScreenAwake());

  let starting = $state(false);
  let startError = $state<string | null>(null);

  async function start() {
    starting = true;
    startError = null;
    try {
      const { token } = await api.startMeeting();
      // Replaces the Admin's Google session with the Kiosk's.
      await session.signInWithToken(token);
    } catch (err) {
      startError =
        err instanceof ApiError && err.body?.code === "too_late"
          ? "It's past 10 PM, too late to start today's Meeting."
          : "Couldn't start the Meeting. Try again.";
    } finally {
      starting = false;
    }
  }
</script>

{#snippet signOut()}
  <button type="button" onclick={() => void session.signOut()}>Sign out</button>
{/snippet}

<main class="kiosk">
  {#if user === undefined || stale}
    <p>Loading…</p>
  {:else if user === null}
    <h1>Troop 813 Meeting Kiosk</h1>
    <p>The Admin unlocks the Kiosk to start today's Meeting.</p>
    <button type="button" class="primary" onclick={() => void session.signIn()}>
      Sign in with Google
    </button>
  {:else if user.kiosk}
    {#key user.kiosk.sessionId}
      <KioskSession
        {api}
        {live}
        kiosk={user.kiosk}
        {at}
        onExpired={() => void session.signOut()}
      />
    {/key}
  {:else if me}
    {#await me}
      <p>Loading…</p>
    {:then caller}
      {#if caller.role === "admin"}
        <h1>Troop 813 Meeting Kiosk</h1>
        <button
          type="button"
          class="primary"
          disabled={starting}
          onclick={() => void start()}
        >
          Start today's Meeting
        </button>
        {#if startError}<p role="alert">{startError}</p>{/if}
      {:else}
        <h1>Troop 813 Meeting Kiosk</h1>
        <p>Only the Admin can unlock the Kiosk.</p>
      {/if}
      {@render signOut()}
    {:catch}
      <p>Couldn't reach Troop 813 HQ. Try again in a moment.</p>
      {@render signOut()}
    {/await}
  {/if}
</main>

<style>
  .kiosk {
    max-width: 40rem;
    margin: 0 auto;
    padding: 1.5rem;
    font-size: 1.25rem;
  }
  .kiosk :global(button) {
    font-size: inherit;
    min-height: 3rem;
    padding: 0.5rem 1.25rem;
  }
  .primary {
    display: block;
    margin-block: 1rem;
  }
</style>
