<script module lang="ts">
  import type { Api } from "./api";
  import type { RSVPRollbackItem } from "./contract.gen";

  /**
   * The RSVP answers Undo of this check-in would roll back (story 78), for
   * the confirm. null when the preview failed: the confirm still works.
   */
  export async function loadRollback(
    api: Api,
    meetingId: string,
    attendanceId: string,
  ): Promise<RSVPRollbackItem[] | null> {
    try {
      return (await api.rsvpRollbackPreview(meetingId, attendanceId)).items;
    } catch {
      return null;
    }
  }
</script>

<script lang="ts">
  // The inline "Undo <Scout>'s check-in?" confirm shared by Tonight and a
  // Meeting's page, with the RSVPs the Undo rolls back.
  import type { RSVPAnswer } from "./contract.gen";

  let {
    name,
    rollback,
    busy,
    onConfirm,
    onCancel,
  }: {
    name: string;
    /** From loadRollback; null when the preview failed. */
    rollback: RSVPRollbackItem[] | null;
    busy: boolean;
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();

  const answerText = (a: RSVPAnswer | null) =>
    a === "going" ? "Going" : a === "notGoing" ? "Not going" : "No answer";
</script>

<span class="confirm">
  <span>Undo {name}'s check-in?</span>
  {#if rollback === null}
    <span>
      Couldn't load which RSVPs this rolls back; Undo still rolls them back.
    </span>
  {:else if rollback.length > 0}
    <span>This rolls back:</span>
    <ul aria-label="RSVPs this rolls back">
      {#each rollback as item (item.eventId)}
        <li>
          {item.eventName}: {answerText(item.from)} → {answerText(item.to)}
        </li>
      {/each}
    </ul>
  {/if}
  <button type="button" disabled={busy} onclick={onConfirm}>Yes, undo</button>
  <button type="button" disabled={busy} onclick={onCancel}>Cancel</button>
</span>
