<script lang="ts">
  // Roster tab (spec stories 54–58, 63–65): Add Scout with rapid entry at the
  // top, then Active Scouts and Inactive Scouts, live from Firestore. Each row
  // can be edited and deactivated / reactivated. Delete appears only while the
  // Scout has no Attendance or RSVPs (a typo entry): the user dropped story
  // 64's always-shown disabled Delete, since nearly every Scout has history
  // after a Meeting or two (2026-11-08).
  // Rapid entry: Enter in last name submits (native form submit); on success
  // the form clears and focus returns to first name for the next Scout.
  import { ApiError, type Api } from "./api";
  import type { AttendanceCount, Scout } from "./contract.gen";
  import type { Live, WithId } from "./live";
  import PencilSimple from "phosphor-svelte/lib/PencilSimple";
  import Trash from "phosphor-svelte/lib/Trash";
  import UserMinus from "phosphor-svelte/lib/UserMinus";
  import UserPlus from "phosphor-svelte/lib/UserPlus";
  import ScoutLink from "./ScoutLink.svelte";
  import { byName, fullName } from "./scouts";

  let { api, live }: { api: Api; live: Live } = $props();

  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();
  // svelte-ignore state_referenced_locally
  const rsvps = live.rsvps();

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
  // One notice at a time. `reactivate` offers Reactivate for an inactive namesake.
  let notice = $state<{
    text: string;
    reactivate?: { id: string; label: string };
  } | null>(null);
  let firstInput: HTMLInputElement | undefined = $state();

  let editingId = $state<string | null>(null);
  let editFirst = $state("");
  let editLast = $state("");
  let confirmingId = $state<string | null>(null);
  // Scouts the server refused to delete (Attendance newer than the loaded counts).
  let refused = $state<string[]>([]);

  const sorted = $derived(
    $scouts.state === "ready" ? [...$scouts.docs].sort(byName) : [],
  );
  const active = $derived(sorted.filter((s) => s.active));
  const inactive = $derived(sorted.filter((s) => !s.active));
  const withRsvps = $derived(
    new Set($rsvps.state === "ready" ? $rsvps.docs.map((r) => r.scoutId) : []),
  );
  // Attendance comes from Go (undone records count too, as the server's in_use
  // rule does); `refused` catches a check-in that landed after counts loaded.
  const inUse = (id: string) =>
    withRsvps.has(id) ||
    counts[id]?.hasAttendance === true ||
    refused.includes(id);

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
      notice = null;
      firstInput?.focus();
    } catch (err) {
      const name = `${first} ${last}`;
      if (err instanceof ApiError && err.body?.code === "duplicate_scout") {
        const id = err.body.inactiveMatchId;
        notice = id
          ? {
              text: `${name} is on the Roster as an inactive Scout.`,
              reactivate: { id, label: name },
            }
          : { text: `${name} is already on the Roster.` };
      } else {
        notice = { text: `Couldn't add ${name}. Try again.` };
      }
    } finally {
      saving = false;
    }
  }

  /** Runs a Roster call for one Scout, turning failures into the notice. */
  async function run(
    name: string,
    verb: string,
    call: () => Promise<void>,
  ): Promise<boolean> {
    try {
      await call();
      notice = null;
      return true;
    } catch (err) {
      notice =
        err instanceof ApiError && err.body?.code === "duplicate_scout"
          ? { text: `${name} is already on the Roster.` }
          : { text: `Couldn't ${verb} ${name}. Try again.` };
      return false;
    }
  }

  async function reactivateNamesake() {
    const target = notice?.reactivate;
    if (!target) return;
    const ok = await run(target.label, "reactivate", () =>
      api.reactivateScout(target.id),
    );
    if (ok) {
      firstName = "";
      lastName = "";
      firstInput?.focus();
    }
  }

  function startEdit(s: WithId<Scout>) {
    editingId = s.id;
    editFirst = s.firstName;
    editLast = s.lastName;
    confirmingId = null;
  }

  async function saveEdit(e: SubmitEvent, s: WithId<Scout>) {
    e.preventDefault();
    const first = editFirst.trim();
    const last = editLast.trim();
    if (!first || !last) return;
    const ok = await run(`${first} ${last}`, "rename", () =>
      api.editScout(s.id, { firstName: first, lastName: last }),
    );
    if (ok) editingId = null;
  }

  async function remove(s: WithId<Scout>) {
    confirmingId = null;
    try {
      await api.deleteScout(s.id);
      notice = null;
    } catch (err) {
      if (err instanceof ApiError && err.body?.code === "in_use") {
        refused = [...refused, s.id];
        notice = null;
      } else {
        notice = { text: `Couldn't delete ${fullName(s)}. Try again.` };
      }
    }
  }
</script>

<main>
  <h1>Roster</h1>

  <form onsubmit={add} aria-label="Add Scout" class="add">
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
  </form>

  {#if notice}
    <p role="alert">{notice.text}</p>
    {#if notice.reactivate}
      <button
        type="button"
        aria-label="Reactivate {notice.reactivate.label}"
        onclick={() => void reactivateNamesake()}>Reactivate</button
      >
    {/if}
  {/if}

  {#if $scouts.state === "loading"}
    <p>Loading…</p>
  {:else if $scouts.state === "error"}
    <p>Couldn't load the Roster. Reload to try again.</p>
  {:else if sorted.length === 0}
    <p>No Scouts yet.</p>
  {:else}
    <h2 id="active-heading">Active Scouts</h2>
    <ul aria-labelledby="active-heading">
      {#each active as s (s.id)}{@render scoutRow(s)}{/each}
    </ul>
    {#if inactive.length > 0}
      <h2 id="inactive-heading">Inactive Scouts</h2>
      <ul aria-labelledby="inactive-heading">
        {#each inactive as s (s.id)}{@render scoutRow(s)}{/each}
      </ul>
    {/if}
  {/if}
</main>

{#snippet scoutRow(s: WithId<Scout>)}
  <li>
    {#if editingId === s.id}
      <form
        aria-label="Edit {fullName(s)}"
        onsubmit={(e) => void saveEdit(e, s)}
      >
        <label>
          First name
          <input bind:value={editFirst} autocomplete="off" required />
        </label>
        <label>
          Last name
          <input bind:value={editLast} autocomplete="off" required />
        </label>
        <button type="submit">Save</button>
        <button type="button" onclick={() => (editingId = null)}>Cancel</button>
      </form>
    {:else}
      <span class="name"><ScoutLink scout={s} /></span>
      {#if s.active}
        {@const c = counts[s.id]}
        {#if c}<span class="meta">Present {c.present} of {c.total}</span>{/if}
      {/if}
      <span class="actions">
        <button
          type="button"
          aria-label="Edit {fullName(s)}"
          class="icon"
          onclick={() => startEdit(s)}
          ><PencilSimple weight="thin" aria-hidden="true" /><span class="label"
            >Edit</span
          ></button
        >
        <button
          type="button"
          aria-label="{s.active ? 'Deactivate' : 'Reactivate'} {fullName(s)}"
          onclick={() =>
            void run(fullName(s), s.active ? "deactivate" : "reactivate", () =>
              s.active ? api.deactivateScout(s.id) : api.reactivateScout(s.id),
            )}
          class="icon"
          >{#if s.active}<UserMinus
              weight="thin"
              aria-hidden="true"
            />{:else}<UserPlus weight="thin" aria-hidden="true" />{/if}<span
            class="label">{s.active ? "Deactivate" : "Reactivate"}</span
          ></button
        >
        {#if confirmingId === s.id}
          <button
            type="button"
            class="danger solid"
            onclick={() => void remove(s)}
          >
            Yes, delete {fullName(s)}
          </button>
          <button type="button" onclick={() => (confirmingId = null)}>
            Keep
          </button>
        {:else if !inUse(s.id)}
          <button
            type="button"
            aria-label="Delete {fullName(s)}"
            class="danger icon"
            onclick={() => (confirmingId = s.id)}
            ><Trash weight="thin" aria-hidden="true" /><span class="label"
              >Delete</span
            ></button
          >
        {/if}
      </span>
    {/if}
  </li>
{/snippet}

<style>
  /* A Scout row: name and "Present x of y" on one line, the actions on the
     next, the can't-delete reason under them. */
  .name {
    flex: 1 1 auto;
  }
  .meta {
    margin-left: auto;
    font-variant-numeric: tabular-nums;
  }
  .actions {
    flex-basis: 100%;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  /* Phones: the row's actions are icon buttons (their accessible names come
     from aria-label); the words return when there is room. */
  .icon {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
  }
  .icon :global(svg) {
    width: 1.5rem;
    height: 1.5rem;
    flex: none;
  }
  @media (max-width: 30rem) {
    /* One shape for every row: name over "Present x of y" on the left, the
       icon actions on the right, whether or not Delete is offered. */
    li:has(> .actions) {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      column-gap: 0.75rem;
      row-gap: 0.125rem;
    }
    li:has(> .actions) > .name,
    li:has(> .actions) > .meta {
      grid-column: 1;
      margin-left: 0;
    }
    .actions {
      grid-column: 2;
      grid-row: 1 / span 2;
      flex-basis: auto;
      justify-content: flex-end;
    }
    .icon {
      min-width: 2.75rem;
      justify-content: center;
      padding-inline: 0.5rem;
    }
    .icon .label {
      display: none;
    }
  }
  /* Rapid entry: first and last name side by side, Add Scout under them. */
  .add {
    grid-template-columns: 1fr 1fr;
  }
  .add button {
    grid-column: 1 / -1;
  }
</style>
