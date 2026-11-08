<script lang="ts">
  // Roster tab (spec stories 54–58): Add Scout with rapid entry at the top,
  // then Active Scouts and Inactive Scouts, live from Firestore.
  // Rapid entry: Enter in last name submits (native form submit); on success
  // the form clears and focus returns to first name for the next Scout.
  import { ApiError, type Api } from "./api";
  import type { AttendanceCount, Scout } from "./contract.gen";
  import type { Live, WithId } from "./live";
  import ScoutLink from "./ScoutLink.svelte";

  let { api, live }: { api: Api; live: Live } = $props();

  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();

  // "Present X of Y" per Scout, from Go (it owns the start rule). Loaded when
  // the Roster opens; a failed load just leaves the counts off.
  let counts = $state<Record<string, AttendanceCount>>({});
  $effect(() => {
    api.scoutAttendance().then(
      (r) => (counts = r.scouts),
      () => {},
    );
  });

  let firstName = $state("");
  let lastName = $state("");
  let saving = $state(false);
  let message = $state<string | null>(null);
  let firstInput: HTMLInputElement | undefined = $state();

  const byName = (a: WithId<Scout>, b: WithId<Scout>) =>
    a.firstName.localeCompare(b.firstName) ||
    a.lastName.localeCompare(b.lastName);

  const sorted = $derived(
    $scouts.state === "ready" ? [...$scouts.docs].sort(byName) : [],
  );
  const active = $derived(sorted.filter((s) => s.active));
  const inactive = $derived(sorted.filter((s) => !s.active));

  async function add(e: SubmitEvent) {
    e.preventDefault();
    const first = firstName.trim();
    const last = lastName.trim();
    if (!first || !last || saving) return;
    saving = true;
    try {
      await api.addScout({ firstName: first, lastName: last });
      firstName = "";
      lastName = "";
      message = null;
      firstInput?.focus();
    } catch (err) {
      message =
        err instanceof ApiError && err.body?.code === "duplicate_scout"
          ? `${first} ${last} is already on the Roster.`
          : `Couldn't add ${first} ${last}. Try again.`;
    } finally {
      saving = false;
    }
  }
</script>

<main>
  <h1>Roster</h1>

  <form onsubmit={add} aria-label="Add Scout">
    <label>
      First name
      <input
        bind:this={firstInput}
        bind:value={firstName}
        autocomplete="off"
        autocapitalize="words"
        required
      />
    </label>
    <label>
      Last name
      <input
        bind:value={lastName}
        autocomplete="off"
        autocapitalize="words"
        required
      />
    </label>
    <button type="submit" disabled={saving}>Add Scout</button>
    {#if message}<p role="alert">{message}</p>{/if}
  </form>

  {#if $scouts.state === "loading"}
    <p>Loading…</p>
  {:else if $scouts.state === "error"}
    <p>Couldn't load the Roster. Reload to try again.</p>
  {:else if sorted.length === 0}
    <p>No Scouts yet.</p>
  {:else}
    <h2 id="active-heading">Active Scouts</h2>
    <ul aria-labelledby="active-heading">
      {#each active as s (s.id)}
        {@const c = counts[s.id]}
        <li>
          <ScoutLink scout={s} />
          {#if c}<span>Present {c.present} of {c.total}</span>{/if}
        </li>
      {/each}
    </ul>
    {#if inactive.length > 0}
      <h2 id="inactive-heading">Inactive Scouts</h2>
      <ul aria-labelledby="inactive-heading">
        {#each inactive as s (s.id)}<li><ScoutLink scout={s} /></li>{/each}
      </ul>
    {/if}
  {/if}
</main>
