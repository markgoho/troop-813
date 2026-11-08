<script lang="ts">
  // Meetings tab (spec stories 75, 80): every Meeting that has a doc, newest
  // first, with how many Scouts were present. Tonight's is marked Live and
  // opens Tonight. Holidays have no Meeting, so they simply don't appear.
  import { resolve } from "$app/paths";
  import { shortDay } from "./dates";
  import type { Live } from "./live";

  let { live, today }: { live: Live; today: string } = $props();

  // svelte-ignore state_referenced_locally
  const meetings = live.meetings();

  const rows = $derived(
    $meetings.state === "ready"
      ? $meetings.docs
          .map((m) => ({ id: m.id, present: Object.keys(m.checkedIn).length }))
          .sort((a, b) => b.id.localeCompare(a.id))
      : [],
  );
</script>

<main>
  <h1>Meetings</h1>
  <p><a href={resolve("/(leader)/meetings/new")}>Add a past Meeting</a></p>

  {#if $meetings.state === "error"}
    <p role="alert">Couldn't load the Meetings. Reload to try again.</p>
  {:else if $meetings.state === "loading"}
    <p>Loading…</p>
  {:else if rows.length === 0}
    <p>No Meetings yet.</p>
  {:else}
    <ul aria-label="Meetings">
      {#each rows as m (m.id)}
        <li>
          {#if m.id === today}
            <a href={resolve("/(leader)")}>{shortDay(m.id)}</a>
            <strong class="live pill">Live</strong>
          {:else}
            <a href={resolve("/(leader)/meetings/[id]", { id: m.id })}
              >{shortDay(m.id)}</a
            >
          {/if}
          <span class="present">{m.present} present</span>
        </li>
      {/each}
    </ul>
  {/if}
</main>
