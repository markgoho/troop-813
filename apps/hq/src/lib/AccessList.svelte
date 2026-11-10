<script lang="ts">
  // Access tab (Admin only, spec stories 81–83): everyone who has signed in,
  // newest first (Go orders it), with a Make Leader / Leader ✓ toggle.
  // The Admin's own row is fixed; Go refuses it too. A Waiting request can be
  // dismissed (after an inline confirm), which hides it for good.
  import { tick } from "svelte";
  import Announcer from "./Announcer.svelte";
  import type { Api } from "./api";
  import type { UserSummary } from "./contract.gen";

  let { api, myUid }: { api: Api; myUid: string } = $props();

  let users = $state<UserSummary[] | null>(null);
  let failed = $state(false);
  let busy = $state<string | null>(null);
  let confirmingId = $state<string | null>(null);
  // Spoken, not shown: the toggle's result.
  let announcement = $state("");

  $effect(() => {
    api.users().then(
      (l) => (users = l.users),
      () => (failed = true),
    );
  });

  // Focus follows the inline confirm: into it when it opens, back to the
  // row's Dismiss when it closes.
  const dismissButtons = $state<Record<string, HTMLButtonElement | null>>({});
  let confirmKeep: HTMLButtonElement | undefined = $state();

  async function askDismiss(uid: string) {
    confirmingId = uid;
    await tick();
    confirmKeep?.focus();
  }

  async function keep(uid: string) {
    confirmingId = null;
    await tick();
    dismissButtons[uid]?.focus();
  }

  async function dismiss(u: UserSummary) {
    confirmingId = null;
    busy = u.uid;
    try {
      await api.dismissUser(u.uid);
      users = users?.filter((x) => x.uid !== u.uid) ?? null;
      announcement = `${u.displayName}'s request dismissed`;
    } catch {
      failed = true;
    } finally {
      busy = null;
    }
  }

  async function toggle(u: UserSummary) {
    const next = u.role === "leader" ? "parent" : "leader";
    busy = u.uid;
    try {
      await api.setRole(u.uid, next);
      u.role = next;
      announcement =
        next === "leader"
          ? `${u.displayName} is now a Leader`
          : `${u.displayName} is no longer a Leader`;
    } catch {
      failed = true;
    } finally {
      busy = null;
    }
  }
</script>

<main>
  <h1>Access</h1>
  <Announcer text={announcement} />
  {#if failed}
    <p role="alert">Couldn't update Access. Reload and try again.</p>
  {/if}
  {#if users === null && !failed}
    <p role="status">Loading…</p>
  {:else if users}
    <ul>
      {#each users as u (u.uid)}
        {@const fixed = u.uid === myUid || u.role === "admin"}
        <li>
          <span class="name">{u.displayName}</span>
          <span class="email">{u.email}</span>
          {#if u.role === "parent"}<span class="pill warn">Waiting</span>{/if}
          <span class="actions">
            {#if confirmingId === u.uid}
              <span
                class="confirm-group"
                role="group"
                aria-label="Dismiss {u.displayName}'s request?"
              >
                <button
                  type="button"
                  class="danger solid"
                  onclick={() => void dismiss(u)}
                >
                  Yes, dismiss
                </button>
                <button
                  bind:this={confirmKeep}
                  type="button"
                  onclick={() => void keep(u.uid)}
                >
                  Keep
                </button>
              </span>
            {:else}
              {#if u.role === "parent"}
                <button
                  bind:this={dismissButtons[u.uid]}
                  type="button"
                  class="danger"
                  disabled={busy === u.uid}
                  aria-label="Dismiss request: {u.displayName}"
                  onclick={() => void askDismiss(u.uid)}
                >
                  Dismiss
                </button>
              {/if}
              <button
                type="button"
                disabled={fixed || busy === u.uid}
                aria-label={u.role === "parent"
                  ? `Make Leader: ${u.displayName}`
                  : `Remove Leader: ${u.displayName}`}
                aria-describedby={fixed ? `fixed-${u.uid}` : undefined}
                onclick={() => void toggle(u)}
              >
                {u.role === "parent" ? "Make Leader" : "Leader ✓"}
              </button>
            {/if}
          </span>
          {#if fixed}
            <span id="fixed-{u.uid}" class="sr-only">
              {u.uid === myUid
                ? "You can't change your own access"
                : "Admins always have access"}
            </span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</main>

<style>
  .actions,
  .confirm-group {
    /* A real box (not display: contents) so the group role survives. */
    display: inline-flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  .actions {
    margin-left: auto;
  }
</style>
