<script lang="ts">
  // The Kiosk RSVP stage (spec stories 24–33): right after a Scout checks in,
  // one card per pending Event (not cancelled, Meeting before its start, no
  // saved answer), then a summary with a Change button on each Event.
  // Going / Not going save; "Ask me next week" (or "Not sure" at the last
  // Meeting before the Event) saves nothing, and walking away never counts as
  // an answer. "Done, pass the tablet on" hands back to the Handoff.
  //
  // Nothing pending means no stage at all: onDone() right away, so the
  // Handoff isn't slowed by a summary the Scout has no reason to read.
  // The cards are fixed when the data first loads, so answering one doesn't
  // reshuffle the rest.
  import { ApiError, type Api } from "./api";
  import type { RSVPAnswer } from "./contract.gen";
  import { dateRange } from "./dates";
  import { docsOf } from "./events";
  import {
    isLastChance,
    type NameListEntry,
    pendingEvents,
    upcomingEvents,
  } from "./kiosk-flow";
  import type { Live } from "./live";

  let {
    api,
    live,
    meetingId,
    scout,
    onDone,
    onExpired,
    onAsleep,
  }: {
    api: Api;
    live: Live;
    /** The Kiosk's Meeting (its date). */
    meetingId: string;
    /** The Scout who just checked in. */
    scout: NameListEntry;
    /** The Scout is finished: on to the Handoff. */
    onDone: () => void;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
    /** The server says the session is asleep: the parent shows the asleep screen. */
    onAsleep: () => void;
  } = $props();

  // The parent re-creates this component per Scout.
  // svelte-ignore state_referenced_locally
  const events = live.events();
  // svelte-ignore state_referenced_locally
  const mine = live.scoutRsvps(scout.id);

  const loaded = $derived($events.state === "ready" && $mine.state === "ready");
  const failed = $derived($events.state === "error" || $mine.state === "error");

  // Answers sent from this screen, ahead of the live snapshot.
  let sent = $state<Record<string, RSVPAnswer>>({});
  // Event IDs to ask, fixed at first load; null until then.
  let queue = $state<string[] | null>(null);
  let index = $state(0);
  // The Event being changed from the summary, if any.
  let changing = $state<string | null>(null);
  let saving = $state(false);
  let message = $state<string | null>(null);

  $effect(() => {
    if (queue !== null || !loaded) return;
    queue = pendingEvents(docsOf($events), docsOf($mine), meetingId).map(
      (e) => e.id,
    );
    if (queue.length === 0) onDone();
  });

  const upcoming = $derived(upcomingEvents(docsOf($events), meetingId));
  const byId = $derived(new Map(upcoming.map((e) => [e.id, e])));
  const current = $derived(
    changing !== null
      ? (byId.get(changing) ?? null)
      : queue !== null && index < queue.length
        ? (byId.get(queue[index] ?? "") ?? null)
        : null,
  );

  function answerOf(eventId: string): RSVPAnswer | null {
    return (
      sent[eventId] ??
      docsOf($mine).find((r) => r.eventId === eventId)?.answer ??
      null
    );
  }

  const answerText = (a: RSVPAnswer | null) =>
    a === "going" ? "Going" : a === "notGoing" ? "Not going" : "No answer";

  /** Moves past the current card (a skipped Event too). */
  function advance() {
    message = null;
    if (changing !== null) changing = null;
    else index += 1;
  }

  async function answer(eventId: string, a: RSVPAnswer) {
    saving = true;
    message = null;
    try {
      await api.kioskSetRsvp({ eventId, scoutId: scout.id, answer: a });
      sent[eventId] = a;
      advance();
    } catch (err) {
      const code = err instanceof ApiError ? err.body?.code : undefined;
      if (code === "session_expired") {
        onExpired();
      } else if (code === "session_asleep") {
        // The parent's asleep screen covers this one; the card is still
        // here when a Leader wakes the Kiosk.
        onAsleep();
      } else {
        message = "Couldn't save that. Try again.";
      }
    } finally {
      saving = false;
    }
  }
</script>

{#if failed}
  <p role="alert">Couldn't load the Events.</p>
  <button type="button" class="primary" onclick={onDone}>
    Done, pass the tablet on
  </button>
{:else if !loaded || queue === null || (queue.length === 0 && changing === null)}
  <p>Loading…</p>
{:else if current}
  {@const event = current}
  <h1>{event.name}</h1>
  <p class="when">{dateRange(event.startDate, event.endDate)}</p>
  {#if event.location}<p>{event.location}</p>{/if}
  {#if changing === null && isLastChance(event.startDate, meetingId)}
    <p class="last-chance">
      This is the last chance: {event.name} is this coming weekend ({dateRange(
        event.startDate,
        event.endDate,
      )})
    </p>
  {/if}
  <p>{scout.label}, are you going?</p>
  <div class="choices">
    <button
      type="button"
      class="yes"
      disabled={saving}
      onclick={() => void answer(event.id, "going")}
    >
      Going
    </button>
    <button
      type="button"
      disabled={saving}
      onclick={() => void answer(event.id, "notGoing")}
    >
      Not going
    </button>
    {#if changing !== null}
      <button type="button" disabled={saving} onclick={advance}>Back</button>
    {:else}
      <button type="button" disabled={saving} onclick={advance}>
        {isLastChance(event.startDate, meetingId)
          ? "Not sure"
          : "Ask me next week"}
      </button>
    {/if}
  </div>
  {#if message}<p role="alert">{message}</p>{/if}
{:else}
  <h1>Your answers</h1>
  <ul aria-label="Upcoming Events" class="summary">
    {#each upcoming as event (event.id)}
      <li>
        <strong>{event.name}</strong>
        <span>{dateRange(event.startDate, event.endDate)}</span>
        <span class="answer">{answerText(answerOf(event.id))}</span>
        <button
          type="button"
          aria-label="Change {event.name}"
          onclick={() => (changing = event.id)}
        >
          Change
        </button>
      </li>
    {/each}
  </ul>
  <button type="button" class="primary" onclick={onDone}>
    Done, pass the tablet on
  </button>
{/if}

<style>
  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
  .last-chance {
    font-weight: 600;
  }
  .summary {
    list-style: none;
    padding: 0;
    display: grid;
    gap: 0.75rem;
  }
  .summary li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1rem;
  }
  .primary {
    margin-block: 1rem;
  }
</style>
