<script lang="ts">
  // "Add a past Meeting" (spec story 76): a date picker that proposes the
  // latest Monday with no Meeting and refuses today, the future and dates that
  // already have a Meeting. Go enforces the same rules.
  import { ApiError, type Api } from "./api";
  import type { Live } from "./live";
  import { defaultPastDate, pastDateError } from "./meetings";

  let {
    api,
    live,
    today,
    onCreated,
  }: {
    api: Api;
    live: Live;
    today: string;
    onCreated: (date: string) => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const meetings = live.meetings();

  const existing = $derived(
    new Set($meetings.state === "ready" ? $meetings.docs.map((m) => m.id) : []),
  );
  const loaded = $derived($meetings.state === "ready");

  // The user's pick; until they pick, the proposed default.
  let picked = $state<string | null>(null);
  const date = $derived(picked ?? defaultPastDate(today, existing));
  const problem = $derived(
    loaded ? pastDateError(date, today, existing) : null,
  );

  let busy = $state(false);
  let failure = $state<string | null>(null);
  const error = $derived(problem ?? failure);

  async function add(e: SubmitEvent) {
    e.preventDefault();
    if (busy || problem) return;
    busy = true;
    failure = null;
    try {
      await api.createPastMeeting(date);
      onCreated(date);
    } catch (err) {
      failure =
        err instanceof ApiError && err.body?.code === "meeting_exists"
          ? "There is already a Meeting on that date."
          : "Couldn't add the Meeting. Try again.";
    } finally {
      busy = false;
    }
  }
</script>

<main>
  <h1>Add a past Meeting</h1>
  {#if $meetings.state === "loading"}
    <p>Loading…</p>
  {:else if $meetings.state === "error"}
    <p role="alert">Couldn't load the Meetings. Reload to try again.</p>
  {:else}
    <form onsubmit={add}>
      <label>
        Meeting date
        <input
          type="date"
          required
          value={date}
          oninput={(e) => {
            picked = e.currentTarget.value;
            failure = null;
          }}
        />
      </label>
      {#if error}<p role="alert">{error}</p>{/if}
      <button type="submit" disabled={busy || problem !== null}>
        Add Meeting
      </button>
    </form>
  {/if}
</main>
