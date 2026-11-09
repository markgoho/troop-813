<script lang="ts">
  // Access tab (Admin only, spec stories 81–83): everyone who has signed in,
  // newest first (Go orders it), with a Make Leader / Leader ✓ toggle.
  // The Admin's own row is fixed; Go refuses it too.
  import Announcer from "./Announcer.svelte";
  import type { Api } from "./api";
  import type { UserSummary } from "./contract.gen";

  let { api, myUid }: { api: Api; myUid: string } = $props();

  let users = $state<UserSummary[] | null>(null);
  let failed = $state(false);
  let busy = $state<string | null>(null);
  // Spoken, not shown: the toggle's result.
  let announcement = $state("");

  $effect(() => {
    api.users().then(
      (l) => (users = l.users),
      () => (failed = true),
    );
  });

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
