<script lang="ts">
  // Sends each person where they belong (spec story 86): signed out → sign-in,
  // Kiosk token → onKiosk (/kiosk), Parent → waiting page, Leader or Admin →
  // the Leader screens (children). The role comes from Go (GET /api/me) on
  // every load, never from the client.
  import type { Snippet } from "svelte";
  import type { Api } from "./api";
  import type { Me } from "./contract.gen";
  import type { Session, SignedInUser } from "./session";

  // children gets the caller (uid, role) so Leader screens can tell the Admin apart.
  let {
    session,
    api,
    onKiosk,
    children,
  }: {
    session: Session;
    api: Api;
    /** Called when the tablet's Kiosk session lands here; the layout navigates to /kiosk. */
    onKiosk: () => void;
    children: Snippet<[Me]>;
  } = $props();

  // undefined until the session reports in (Firebase restores it asynchronously).
  let user = $state<SignedInUser | null | undefined>(undefined);
  $effect(() =>
    session.onChange((u) => {
      user = u;
    }),
  );

  const kiosk = $derived(user?.kiosk !== undefined);
  $effect(() => {
    if (kiosk) onKiosk();
  });

  // A Kiosk token never asks /api/me (it would be refused: Kiosk isn't a Leader).
  const me = $derived(user && !user.kiosk ? api.me() : null);
</script>

{#snippet notice()}
  <p class="notice">
    Troop 813 uses this to record Scout names, Meeting attendance and Event
    RSVPs. Leaders only.
  </p>
{/snippet}

{#snippet signOut()}
  <button type="button" class="quiet" onclick={() => void session.signOut()}>
    Sign out
  </button>
{/snippet}

{#if user === undefined || kiosk}
  <p class="status">Loading…</p>
{:else if me === null}
  <main class="gate">
    <img src="/brand/troop-813-logo.png" alt="" class="crest" />
    <h1>Troop 813 HQ</h1>
    <button type="button" class="primary" onclick={() => void session.signIn()}>
      Sign in with Google
    </button>
    {@render notice()}
  </main>
{:else}
  {#await me}
    <p class="status">Loading…</p>
  {:then caller}
    {#if caller.role === "leader" || caller.role === "admin"}
      {@render children(caller)}
    {:else}
      <main class="gate">
        <img src="/brand/troop-813-logo.png" alt="" class="crest" />
        <h1>Troop 813 HQ</h1>
        <p>You're signed in. To use HQ, ask the Admin for access.</p>
        {@render signOut()}
        {@render notice()}
      </main>
    {/if}
  {:catch}
    <main class="gate">
      <p>Couldn't reach Troop 813 HQ. Try again in a moment.</p>
      {@render signOut()}
    </main>
  {/await}
{/if}

<style>
  /* Sign-in, waiting and error pages: centred, logo first. */
  .gate {
    min-height: 100dvh;
    max-width: 26rem;
    margin: 0 auto;
    padding: 2rem 1.25rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    text-align: center;
  }
  .crest {
    width: 7.5rem;
    height: 7.5rem;
    filter: var(--logo-filter);
  }
  h1 {
    font-size: 2.5rem;
  }
  .status {
    padding: 2rem;
    text-align: center;
    color: var(--muted);
  }
  .notice {
    color: var(--muted);
    font-size: 0.875rem;
  }
  button {
    min-height: 3rem;
    padding: 0.625rem 1.5rem;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--surface);
    font-weight: 700;
  }
  .primary {
    background: var(--fill);
    border-color: var(--fill);
    color: var(--fill-ink);
  }
  .quiet {
    border-color: transparent;
    background: transparent;
    color: var(--muted);
    font-weight: 400;
  }
</style>
