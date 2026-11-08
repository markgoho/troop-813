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
  <button type="button" onclick={() => void session.signOut()}>Sign out</button>
{/snippet}

{#if user === undefined || kiosk}
  <p>Loading…</p>
{:else if me === null}
  <main>
    <h1>Troop 813 HQ</h1>
    <button type="button" onclick={() => void session.signIn()}>
      Sign in with Google
    </button>
    {@render notice()}
  </main>
{:else}
  {#await me}
    <p>Loading…</p>
  {:then caller}
    {#if caller.role === "leader" || caller.role === "admin"}
      {@render children(caller)}
    {:else}
      <main>
        <h1>Troop 813 HQ</h1>
        <p>You're signed in. To use HQ, ask the Admin for access.</p>
        {@render signOut()}
        {@render notice()}
      </main>
    {/if}
  {:catch}
    <main>
      <p>Couldn't reach Troop 813 HQ. Try again in a moment.</p>
      {@render signOut()}
    </main>
  {/await}
{/if}
