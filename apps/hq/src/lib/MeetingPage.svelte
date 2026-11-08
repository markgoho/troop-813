<script lang="ts">
  // A Meeting's page (spec stories 77–79): a Scout checklist where ticking is
  // a Leader check-in and unticking asks, then undoes through the same
  // endpoint Tonight uses (so the RSVP rollback applies). "Delete this
  // Meeting" only while nobody is present, and never for tonight's.
  import { resolve } from "$app/paths";
  import { ApiError, type Api } from "./api";
  import type { RSVPRollbackItem, Scout } from "./contract.gen";
  import { shortDay } from "./dates";
  import { guardedActions } from "./guarded.svelte";
  import type { Live, WithId } from "./live";
  import ScoutLink from "./ScoutLink.svelte";
  import { byName, fullName } from "./scouts";
  import UndoConfirm, { loadRollback } from "./UndoConfirm.svelte";

  let {
    api,
    live,
    meetingId,
    today,
    onDeleted,
  }: {
    api: Api;
    live: Live;
    meetingId: string;
    today: string;
    onDeleted: () => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const meeting = live.meeting(meetingId);
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();

  const checkedIn = $derived(
    $meeting.state === "ready" && $meeting.doc ? $meeting.doc.checkedIn : {},
  );
  const presentCount = $derived(Object.keys(checkedIn).length);
  // Active Scouts, plus anyone inactive who was present that day.
  const checklist = $derived(
    $scouts.state === "ready"
      ? $scouts.docs.filter((s) => s.active || s.id in checkedIn).sort(byName)
      : [],
  );
  const isTonight = $derived(meetingId === today);

  let confirming = $state<string | null>(null); // scout id awaiting Yes/Cancel
  let deleting = $state(false);
  // The answers confirming's Undo rolls back; null when the preview failed.
  let rollback = $state<RSVPRollbackItem[] | null>(null);
  const guard = guardedActions();
  const busy = $derived(guard.busy);
  const message = $derived(guard.message);

  /** Runs a guarded call; a success closes the inline confirm. */
  async function run(failure: string, action: () => Promise<unknown>) {
    const ok = await guard.run(failure, action);
    if (ok) confirming = null;
    return ok;
  }

  async function askUndo(s: WithId<Scout>) {
    // loadRollback never throws: a failed preview is null.
    const ok = await guard.run("", async () => {
      rollback = await loadRollback(api, meetingId, checkedIn[s.id] ?? "");
    });
    if (ok) confirming = s.id;
  }

  function toggle(e: MouseEvent, s: WithId<Scout>) {
    // The checkbox mirrors the live Meeting doc, never its own click.
    e.preventDefault();
    if (busy) return;
    if (s.id in checkedIn) {
      void askUndo(s);
    } else {
      confirming = null;
      void run(`Couldn't check in ${fullName(s)}. Try again.`, () =>
        api.leaderCheckIn(meetingId, s.id),
      );
    }
  }

  const undo = (s: WithId<Scout>) =>
    run(`Couldn't undo ${fullName(s)}'s check-in. Try again.`, () =>
      api.undoCheckIn(meetingId, checkedIn[s.id] ?? ""),
    );

  async function remove() {
    const ok = await guard.run(
      "Couldn't delete the Meeting. Try again.",
      () => api.deleteMeeting(meetingId),
      (err) =>
        err instanceof ApiError && err.body?.code === "in_use"
          ? "Someone is present, so this Meeting can't be deleted."
          : "Couldn't delete the Meeting. Try again.",
    );
    if (ok) onDeleted();
    else deleting = false;
  }
</script>

<main>
  <h1>{shortDay(meetingId)}</h1>

  {#if $meeting.state === "error" || $scouts.state === "error"}
    <p role="alert">Couldn't load this Meeting. Reload to try again.</p>
  {:else if $meeting.state === "loading" || $scouts.state === "loading"}
    <p>Loading…</p>
  {:else if !$meeting.doc}
    <p>No Meeting on {shortDay(meetingId)}.</p>
  {:else}
    <p class="counts"><strong>{presentCount} present</strong></p>
    {#if isTonight}
      <p><a href={resolve("/(leader)")}>Open Tonight</a></p>
    {/if}

    {#if message}<p role="alert">{message}</p>{/if}

    <ul aria-label="Scouts">
      {#each checklist as s (s.id)}
        <li>
          <input
            type="checkbox"
            aria-label={fullName(s)}
            checked={s.id in checkedIn}
            onclick={(e) => toggle(e, s)}
          />
          <span class="name"><ScoutLink scout={s} /></span>
          {#if confirming === s.id}
            <UndoConfirm
              name={fullName(s)}
              {rollback}
              {busy}
              onConfirm={() => undo(s)}
              onCancel={() => (confirming = null)}
            />
          {/if}
        </li>
      {/each}
    </ul>

    {#if presentCount === 0 && !isTonight}
      {#if deleting}
        <p>
          <button type="button" disabled={busy} onclick={remove}>
            Yes, delete this Meeting
          </button>
          <button
            type="button"
            disabled={busy}
            onclick={() => (deleting = false)}
          >
            Cancel
          </button>
        </p>
      {:else}
        <p>
          <button type="button" onclick={() => (deleting = true)}>
            Delete this Meeting
          </button>
        </p>
      {/if}
    {/if}
  {/if}
</main>
