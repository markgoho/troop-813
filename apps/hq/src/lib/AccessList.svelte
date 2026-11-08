<script lang="ts">
  // Access tab (Admin only, spec stories 81–83): everyone who has signed in,
  // newest first (Go orders it), with a Make Leader / Leader ✓ toggle.
  // The Admin's own row is fixed; Go refuses it too.
  import type { Api } from "./api";
  import type { UserSummary } from "./contract.gen";

  let { api, myUid }: { api: Api; myUid: string } = $props();

  let users = $state<UserSummary[] | null>(null);
  let failed = $state(false);
  let busy = $state<string | null>(null);

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
    } catch {
      failed = true;
    } finally {
      busy = null;
    }
  }
</script>

<main>
  <h1>Access</h1>
  {#if failed}
    <p role="alert">Couldn't update Access. Reload and try again.</p>
  {/if}
  {#if users === null && !failed}
    <p>Loading…</p>
  {:else if users}
    <ul>
      {#each users as u (u.uid)}
        <li>
          <span>{u.displayName}</span>
          <span class="email">{u.email}</span>
          {#if u.role === "parent"}<span class="pill">Waiting</span>{/if}
          <button
            type="button"
            disabled={u.uid === myUid || u.role === "admin" || busy === u.uid}
            aria-pressed={u.role !== "parent"}
            onclick={() => void toggle(u)}
          >
            {u.role === "parent" ? "Make Leader" : "Leader ✓"}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</main>

<style>
  .email {
    color: #666;
    font-size: 0.875em;
  }
  .pill {
    border-radius: 999px;
    padding: 0 0.5em;
    background: #fde68a;
  }
</style>
