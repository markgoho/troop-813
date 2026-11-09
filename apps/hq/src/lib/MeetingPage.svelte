<script lang="ts">
  // A Meeting's page (spec stories 77–79): a Scout checklist where ticking is
  // a Leader check-in and unticking asks, then undoes through the same
  // endpoint Tonight uses (so the RSVP rollback applies). "Delete this
  // Meeting" only while nobody is present, and never for tonight's.
  import { tick } from "svelte";
  import { resolve } from "$app/paths";
  import Announcer from "./Announcer.svelte";
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
  // Spoken, not shown: a check-in or its Undo.
  let announcement = $state("");
  const guard = guardedActions();
  const busy = $derived(guard.busy);
  const message = $derived(guard.message);

  /** Runs a guarded call; a success closes the inline confirm. */
  async function run(failure: string, action: () => Promise<unknown>) {
    const ok = await guard.run(failure, action);
    if (ok) confirming = null;
    return ok;
  }

  // Focus follows the inline swaps: into a confirm when it opens, back to its
  // trigger (the Scout's checkbox, the Delete button) when it is dismissed.
  const checkboxes: Record<string, HTMLInputElement | null> = {};
  let deleteButton: HTMLButtonElement | undefined = $state();
  let deleteKeep: HTMLButtonElement | undefined = $state();
  async function cancelUndo(id: string) {
    confirming = null;
    await tick();
    checkboxes[id]?.focus();
  }
  async function askDelete() {
    deleting = true;
    await tick();
    deleteKeep?.focus();
  }
  async function keepMeeting() {
    deleting = false;
    await tick();
    deleteButton?.focus();
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
      ).then((ok) => {
        if (ok) announcement = `${fullName(s)} checked in`;
      });
    }
  }

  async function undo(s: WithId<Scout>) {
    // The confirm (and the focus in it) closes: hand focus to the checkbox.
    const inRow = checkboxes[s.id]
      ?.closest("li")
      ?.contains(document.activeElement);
    const ok = await run(
      `Couldn't undo ${fullName(s)}'s check-in. Try again.`,
      () => api.undoCheckIn(meetingId, checkedIn[s.id] ?? ""),
    );
    if (!ok) return;
    announcement = `${fullName(s)}'s check-in undone`;
    if (inRow) {
      await tick();
      checkboxes[s.id]?.focus();
    }
  }

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

<svelte:head><title>{shortDay(meetingId)} · Troop 813 HQ</title></svelte:head>

<main>
  <h1>{shortDay(meetingId)}</h1>
  <Announcer text={announcement} />

  {#if $meeting.state === "error" || $scouts.state === "error"}
    <p role="alert">Couldn't load this Meeting. Reload to try again.</p>
  {:else if $meeting.state === "loading" || $scouts.state === "loading"}
    <p role="status">Loading…</p>
  {:else if !$meeting.doc}
    <p>No Meeting on {shortDay(meetingId)}.</p>
  {:else}
    <p class="counts stats">
      <strong>{presentCount} present</strong>
    </p>
    {#if isTonight}
      <p><a href={resolve("/(leader)")}>Open Tonight</a></p>
    {/if}

    {#if message}<p role="alert">{message}</p>{/if}

    <ul aria-label="Scouts">
      {#each checklist as s (s.id)}
        <li>
          <!-- The label only widens the tap target to 44px. -->
          <label class="check">
            <input
              bind:this={checkboxes[s.id]}
              type="checkbox"
              aria-label={fullName(s)}
              checked={s.id in checkedIn}
              onclick={(e) => toggle(e, s)}
            />
          </label>
          <span class="name"><ScoutLink scout={s} /></span>
          {#if confirming === s.id}
            <UndoConfirm
              name={fullName(s)}
              {rollback}
              {busy}
              onConfirm={() => undo(s)}
              onCancel={() => void cancelUndo(s.id)}
            />
          {/if}
        </li>
      {/each}
    </ul>

    {#if presentCount === 0 && !isTonight}
      {#if deleting}
        <p role="group" aria-labelledby="delete-yes">
          <button
            id="delete-yes"
            type="button"
            class="danger solid"
            disabled={busy}
            onclick={remove}
          >
            Yes, delete this Meeting
          </button>
          <button
            bind:this={deleteKeep}
            type="button"
            disabled={busy}
            onclick={() => void keepMeeting()}
          >
            Cancel
          </button>
        </p>
      {:else}
        <p>
          <button
            bind:this={deleteButton}
            type="button"
            class="danger"
            onclick={() => void askDelete()}
          >
            Delete this Meeting
          </button>
        </p>
      {/if}
    {/if}
  {/if}
</main>

<style>
  /* A 44px tap target around the 1.5rem checkbox; the negative margin keeps
     the row's height and the checkbox's place unchanged. */
  .check {
    display: grid;
    place-items: center;
    flex: none;
    width: 2.75rem;
    height: 2.75rem;
    margin: -0.625rem;
    cursor: pointer;
  }
</style>
