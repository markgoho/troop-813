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
  <button type="button" class="quiet" onclick={() => void session.signOut()}>
    Sign out
  </button>
{/snippet}

<div class="kiosk">
  <header class="bar">
    <img src="/brand/troop-813-logo.png" alt="" class="logo" />
    <span>Troop 813</span>
  </header>
  <main class="stage">
    {#if user === undefined || stale}
      <p class="status">Loading…</p>
    {:else if user === null}
      <div class="lock">
        <img src="/brand/troop-813-logo.png" alt="" class="crest" />
        <h1>Troop 813 Meeting Kiosk</h1>
        <p class="sub">The Admin unlocks the Kiosk to start today's Meeting.</p>
        <button
          type="button"
          class="primary"
          onclick={() => void session.signIn()}
        >
          Sign in with Google
        </button>
      </div>
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
        <p class="status">Loading…</p>
      {:then caller}
        <div class="lock">
          <img src="/brand/troop-813-logo.png" alt="" class="crest" />
          <h1>Troop 813 Meeting Kiosk</h1>
          {#if caller.role === "admin"}
            <button
              type="button"
              class="primary"
              disabled={starting}
              onclick={() => void start()}
            >
              Start today's Meeting
            </button>
            {#if startError}<p role="alert" class="alert">{startError}</p>{/if}
          {:else}
            <p class="sub">Only the Admin can unlock the Kiosk.</p>
          {/if}
          {@render signOut()}
        </div>
      {:catch}
        <div class="lock">
          <p class="sub">Couldn't reach Troop 813 HQ. Try again in a moment.</p>
          {@render signOut()}
        </div>
      {/await}
    {/if}
  </main>
</div>

<style>
  /* The Kiosk is read at arm's length on a shared tablet: a larger base size,
     tall tap targets, one question per screen. Button and text styles here are
     :global so the Kiosk's child screens share one vocabulary. */
  .kiosk {
    min-height: 100dvh;
    display: grid;
    grid-template-rows: auto 1fr;
    font-size: 1.25rem;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.75rem max(1.5rem, env(safe-area-inset-left));
    font-family: var(--display);
    font-weight: 600;
    font-size: 1.125rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .logo {
    width: 2.25rem;
    height: 2.25rem;
  }
  .crest {
    width: 9rem;
    height: 9rem;
  }
  .logo,
  .crest {
    filter: var(--logo-filter);
  }
  .stage {
    width: 100%;
    max-width: 60rem;
    margin: 0 auto;
    padding: 1rem max(1.5rem, env(safe-area-inset-right)) 3rem;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 1.5rem;
  }
  .lock {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 1.25rem;
  }
  .lock h1 {
    font-size: 3rem;
  }

  /* Shared Kiosk vocabulary, at zero specificity so each screen can override. */
  :global(:where(.kiosk) h1) {
    font-size: 3rem;
  }
  :global(:where(.kiosk) .sub),
  :global(:where(.kiosk) .status) {
    color: var(--muted);
    font-size: 1.375rem;
  }
  :global(:where(.kiosk) .status) {
    text-align: center;
  }
  :global(:where(.kiosk) .alert) {
    background: var(--bad-soft);
    color: var(--bad);
    border-radius: var(--radius-sm);
    padding: 0.75rem 1rem;
    font-weight: 700;
  }
  :global(:where(.kiosk) button) {
    min-height: 4rem;
    padding: 0.75rem 1.5rem;
    border: 2px solid var(--line);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--ink);
    font-size: 1.375rem;
    font-weight: 700;
    transition:
      border-color 160ms var(--ease),
      background-color 160ms var(--ease),
      transform 160ms var(--ease);
  }
  :global(:where(.kiosk) button:not(:disabled):hover) {
    border-color: var(--accent);
  }
  :global(:where(.kiosk) button:not(:disabled):active) {
    transform: scale(0.98);
  }
  :global(:where(.kiosk) button:disabled) {
    opacity: 0.55;
  }
  :global(:where(.kiosk) button.primary),
  :global(:where(.kiosk) button.yes) {
    background: var(--fill);
    border-color: var(--fill);
    color: var(--fill-ink);
  }
  :global(:where(.kiosk) button.primary:not(:disabled):hover),
  :global(:where(.kiosk) button.yes:not(:disabled):hover) {
    background: var(--fill-hover);
    border-color: var(--fill-hover);
  }
  :global(:where(.kiosk) button.quiet) {
    border-color: transparent;
    background: transparent;
    color: var(--muted);
    font-weight: 400;
  }
  :global(:where(.kiosk) .choices) {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
  :global(:where(.kiosk) .choices button) {
    flex: 1 1 14rem;
  }
  /* Upright tablet: answers stack full-width, so long labels stay on one line. */
  @media (orientation: portrait) {
    :global(:where(.kiosk) .choices) {
      flex-direction: column;
    }
    :global(:where(.kiosk) .choices button) {
      flex-basis: auto;
    }
  }
</style>
