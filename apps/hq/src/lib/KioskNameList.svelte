<script lang="ts">
  // The Kiosk name list (spec stories 9–12): "Who's first?", every active
  // Scout by name-list label, checked-in Scouts greyed with a check mark.
  // Tap a name → "Are you <Scout>?" → Yes records Attendance (method nameList)
  // at once, then back to the list. Greyed comes from the live Meeting doc's
  // checkedIn, never from the API response, so an undo on a Leader's phone
  // or a reload shows up here by itself.
  //
  // #24 puts the Handoff screen where this returns to the list after a check-in.
  import { ApiError, type Api } from "./api";
  import { type NameListEntry, nameList } from "./kiosk-flow";
  import type { Live } from "./live";
  import type { KioskClaims } from "./session";

  let {
    api,
    live,
    kiosk,
    onExpired,
  }: {
    api: Api;
    live: Live;
    kiosk: KioskClaims;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
  } = $props();

  // The parent re-creates this component for a new session.
  // svelte-ignore state_referenced_locally
  const meeting = live.meeting(kiosk.meetingId);
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();

  const names = $derived(
    $scouts.state === "ready" ? nameList($scouts.docs) : [],
  );
  const checkedIn = $derived(
    $meeting.state === "ready" && $meeting.doc ? $meeting.doc.checkedIn : {},
  );
  const anyoneIn = $derived(Object.keys(checkedIn).length > 0);

  let confirming = $state<NameListEntry | null>(null);
  let saving = $state(false);
  let message = $state<string | null>(null);

  function pick(entry: NameListEntry) {
    message = null;
    confirming = entry;
  }

  async function confirm(entry: NameListEntry) {
    saving = true;
    message = null;
    try {
      await api.kioskCheckIn({ scoutId: entry.id, method: "nameList" });
      confirming = null;
    } catch (err) {
      const code = err instanceof ApiError ? err.body?.code : undefined;
      if (code === "session_expired") {
        onExpired();
      } else if (code === "session_asleep") {
        // The asleep screen and Wake arrive with #27.
        confirming = null;
        message = "Kiosk asleep, ask a Leader";
      } else {
        message = "Couldn't check you in. Try again.";
      }
    } finally {
      saving = false;
    }
  }
</script>

{#if $meeting.state === "loading" || $scouts.state === "loading"}
  <p>Loading…</p>
{:else if $meeting.state === "error" || $scouts.state === "error"}
  <p>Couldn't load the Meeting. Reload to try again.</p>
{:else if !$meeting.doc}
  <p>Today's Meeting hasn't started. Ask the Admin.</p>
{:else if confirming}
  {@const entry = confirming}
  <h1>Are you {entry.label}?</h1>
  <div class="choices">
    <button
      type="button"
      class="yes"
      disabled={saving}
      onclick={() => void confirm(entry)}
    >
      Yes, I'm {entry.label}
    </button>
    <button type="button" disabled={saving} onclick={() => (confirming = null)}>
      No
    </button>
  </div>
  {#if message}<p role="alert">{message}</p>{/if}
{:else}
  <h1>{anyoneIn ? "Tap your name" : "Who's first?"}</h1>
  {#if message}<p role="alert">{message}</p>{/if}
  <ul aria-label="Scouts" class="names">
    {#each names as entry (entry.id)}
      {@const done = entry.id in checkedIn}
      <li>
        <button
          type="button"
          class:done
          disabled={done}
          onclick={() => pick(entry)}
        >
          {done ? `✓ ${entry.label}` : entry.label}
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .names {
    list-style: none;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 0.75rem;
  }
  .names button {
    width: 100%;
  }
  .done {
    opacity: 0.45;
  }
  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
</style>
