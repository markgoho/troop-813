<script lang="ts">
  // The Kiosk RSVP stage (spec stories 24–33): right after a Scout checks in,
  // one card per pending Event (not cancelled, Meeting before its start, no
  // saved answer), then a summary with a Change button on each Event.
  // Going / Not going save; "Ask me next week" (or "Not sure" at the last
  // Meeting before the Event) saves nothing, and walking away never counts as
  // an answer. "Done, pass the tablet on" hands back to the Handoff.
  //
  // With nothing pending the Scout still sees the summary of their answers
  // (story 30); only when there are no upcoming Events at all is there no
  // stage: onDone() right away. A reloaded Kiosk (`resume`) skips the
  // summary too: it only brings the stage back while cards are pending.
  // The cards are fixed when the data first loads, so answering one doesn't
  // reshuffle the rest.
  import Check from "phosphor-svelte/lib/Check";
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
    resume = false,
    onDone,
    onExpired,
    onAsleep,
    onScreen,
  }: {
    api: Api;
    live: Live;
    /** The Kiosk's Meeting (its date). */
    meetingId: string;
    /** The Scout who just checked in. */
    scout: NameListEntry;
    /** A reloaded Kiosk resuming: with nothing pending, skip the stage (no summary). */
    resume?: boolean;
    /** The Scout is finished: on to the Handoff. */
    onDone: () => void;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
    /** Which card is showing (the Kiosk's tap guard keys off changes). */
    onScreen?: (screen: string) => void;
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
    // Nothing to ask: skip the stage when there are no upcoming Events at
    // all, or on a reload resume. Otherwise the summary shows (story 30).
    const none = resume
      ? queue.length === 0
      : upcomingEvents(docsOf($events), meetingId).length === 0;
    if (none) onDone();
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

  // Each card and the summary is its own screen for the tap guard: answer
  // buttons stay put between cards, so a double tap would answer the next one.
  $effect(() => {
    if (queue !== null)
      onScreen?.(`rsvp:${scout.id}:${current?.id ?? "summary"}`);
  });

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

<div class="rsvp">
  {#if failed}
    <p role="alert" class="alert">Couldn't load the Events.</p>
    <button type="button" class="primary" onclick={onDone}>
      Done, pass the tablet on
    </button>
  {:else if !loaded || queue === null}
    <p class="status">Loading…</p>
  {:else if current}
    {@const event = current}
    <div class="head">
      <div class="event">
        <h1>{event.name}</h1>
        <p class="when">
          {dateRange(event.startDate, event.endDate)}{#if event.location}<span
              class="where">{event.location}</span
            >{/if}
        </p>
      </div>
      {#if changing === null && isLastChance(event.startDate, meetingId)}
        <p class="last-chance">
          This is the last chance: {event.name} is this coming weekend ({dateRange(
            event.startDate,
            event.endDate,
          )})
        </p>
      {/if}
    </div>
    <p class="ask">{scout.label}, are you going?</p>
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
    {#if message}<p role="alert" class="alert">{message}</p>{/if}
    {#if changing === null && queue.length > 0}
      <p class="progress">{index + 1} of {queue.length}</p>
    {/if}
  {:else}
    <h1>{scout.label}'s answers</h1>
    <p class="checked-in">
      <Check size="1.5rem" aria-hidden="true" />You're checked in
    </p>
    <ul aria-label="Upcoming Events" class="summary">
      {#each upcoming as event (event.id)}
        <li>
          <strong>{event.name}</strong>
          <span>{dateRange(event.startDate, event.endDate)}</span>
          <span class="answer {answerOf(event.id) ?? 'none'}"
            >{answerText(answerOf(event.id))}</span
          >
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
</div>

<style>
  /* Top-aligned, not centred like the other Kiosk screens: the question and
     answer buttons must sit in the same place on every card. The head slot
     is sized for the tallest card (Event, date, last-chance reminder), so a
     card without the reminder doesn't pull the buttons up. */
  .rsvp {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    padding-top: 1.5rem;
  }
  .head {
    display: grid;
    align-content: end; /* spare room sits above the title, not between it and the question */
    gap: 1.5rem;
    min-height: 11.5rem;
  }
  .progress {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .checked-in {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--ok);
    font-weight: 700;
  }
  .event {
    display: grid;
    gap: 0.5rem;
  }
  .event h1 {
    font-size: 3.5rem;
  }
  .when {
    color: var(--muted);
    font-size: 1.375rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0 1rem;
  }
  .last-chance {
    background: var(--warn-soft);
    color: var(--warn);
    border-radius: var(--radius-sm);
    padding: 0.875rem 1.125rem;
    font-weight: 700;
  }
  .ask {
    font-size: 1.75rem;
    font-weight: 700;
  }
  .summary {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.75rem;
  }
  .summary li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    align-items: center;
    gap: 0.25rem 1rem;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: 0.75rem 0.75rem 0.75rem 1.25rem;
  }
  .summary strong {
    font-size: 1.375rem;
  }
  .summary li > span:not(.answer) {
    grid-column: 1;
    grid-row: 2;
    color: var(--muted);
  }
  .summary li button {
    grid-row: 1 / span 2;
    min-height: 3.5rem;
  }
  .answer {
    grid-row: 1 / span 2;
    padding: 0.25rem 0.875rem;
    border-radius: 999px;
    font-weight: 700;
    white-space: nowrap;
  }
  .answer.going {
    background: var(--ok-soft);
    color: var(--ok);
  }
  .answer.notGoing {
    background: var(--bad-soft);
    color: var(--bad);
  }
  .answer.none {
    color: var(--muted);
    border: 2px dashed var(--line);
  }
  .primary {
    align-self: stretch;
    min-height: 5rem;
    font-size: 1.75rem;
  }
</style>
