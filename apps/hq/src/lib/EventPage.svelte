<script lang="ts">
  // Event page (spec stories 70–74): counts, every active Scout with a
  // Going / Not / — switch (— clears, so the Kiosk asks again), who answered
  // last with full history on tap, and Edit, Cancel / Uncancel, Delete.
  // Status first: counts, then the list, then the actions below it.
  // Reads are live (onSnapshot); every change goes through the API.
  import { tick } from "svelte";
  import Announcer from "./Announcer.svelte";
  import EventForm from "./EventForm.svelte";
  import ScoutLink from "./ScoutLink.svelte";
  import { ApiError, type Api } from "./api";
  import type {
    EventInput,
    RSVPAnswer,
    RSVPEntry,
    Scout,
  } from "./contract.gen";
  import { dateRange, moment } from "./dates";
  import { activeScouts, docsOf, tally } from "./events";
  import type { Live, WithId } from "./live";
  import { fullName } from "./scouts";

  let {
    api,
    live,
    eventId,
    onDeleted,
  }: {
    api: Api;
    live: Live;
    eventId: string;
    onDeleted: () => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const events = live.events();
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();
  // svelte-ignore state_referenced_locally
  const rsvps = live.rsvps();

  // Answers the Leader just picked, shown until the next RSVP snapshot.
  let pending = $state<Record<string, RSVPAnswer | null>>({});
  $effect(() => {
    void $rsvps;
    pending = {};
  });

  const event = $derived(docsOf($events).find((e) => e.id === eventId));
  const active = $derived(activeScouts(docsOf($scouts)));
  const mine = $derived(docsOf($rsvps).filter((r) => r.eventId === eventId));
  const byScout = $derived(new Map(mine.map((r) => [r.scoutId, r])));
  const counts = $derived(tally(eventId, active, mine));
  // Any RSVP doc, even a cleared one, holds history Delete would lose (Go refuses too).
  const inUse = $derived(mine.length > 0);

  let error = $state<string | null>(null);
  let editing = $state(false);
  let confirmingDelete = $state(false);
  let busy = $state(false);
  // Spoken, not shown: what a Cancel / Uncancel or an RSVP change did.
  let announcement = $state("");

  const choices: { value: RSVPAnswer | null; text: string; label: string }[] = [
    { value: "going", text: "Going", label: "Going" },
    { value: "notGoing", text: "Not", label: "Not going" },
    { value: null, text: "—", label: "No answer" },
  ];
  const answerText: Record<RSVPAnswer, string> = {
    going: "Going",
    notGoing: "Not going",
  };

  function shown(scoutId: string): RSVPAnswer | null {
    return scoutId in pending
      ? (pending[scoutId] ?? null)
      : (byScout.get(scoutId)?.answer ?? null);
  }

  async function choose(s: WithId<Scout>, answer: RSVPAnswer | null) {
    error = null;
    pending[s.id] = answer;
    try {
      await api.setRsvp(eventId, s.id, answer);
      announcement = `${fullName(s)}: ${answer === null ? "No answer" : answerText[answer]}`;
    } catch {
      delete pending[s.id];
      error = `Couldn't save ${s.firstName}'s RSVP. Try again.`;
    }
  }

  function who(e: RSVPEntry): string {
    return `${e.attendanceId === null ? "Leader" : "Kiosk"} · ${moment(e.at.toDate())}`;
  }

  async function act(f: () => Promise<void>, failed: string, done = "") {
    error = null;
    busy = true;
    try {
      await f();
      announcement = done;
    } catch (e) {
      error =
        e instanceof ApiError && e.body?.code === "in_use"
          ? "Scouts have RSVP history for this Event, so it can't be deleted. Cancel it instead."
          : failed;
    } finally {
      busy = false;
    }
  }

  // Focus follows the inline delete confirm: onto its safe button on open,
  // back to Delete Event on Keep it.
  let deleteButton: HTMLButtonElement | undefined = $state();
  let keepButton: HTMLButtonElement | undefined = $state();
  async function askDelete() {
    confirmingDelete = true;
    await tick();
    keepButton?.focus();
  }
  async function keepIt() {
    confirmingDelete = false;
    await tick();
    deleteButton?.focus();
  }

  async function save(input: EventInput) {
    await api.updateEvent(eventId, input);
    editing = false;
  }

  const setCancelled = (cancelled: boolean) =>
    act(
      () => api.setEventCancelled(eventId, cancelled),
      "Couldn't update the Event. Try again.",
      cancelled ? "Event cancelled" : "Event uncancelled",
    );

  const remove = () =>
    act(async () => {
      await api.deleteEvent(eventId);
      onDeleted();
    }, "Couldn't delete the Event. Try again.");
</script>

<svelte:head
  ><title>{event ? event.name : "Events"} · Troop 813 HQ</title></svelte:head
>

<main>
  <Announcer text={announcement} />
  {#if $events.state === "error"}
    <p role="alert">Couldn't load this Event. Reload to try again.</p>
  {:else if $events.state === "loading"}
    <p role="status">Loading…</p>
  {:else if !event}
    <p>This Event doesn't exist (it may have been deleted).</p>
  {:else}
    <h1>
      {#if event.cancelled}
        <s>{event.name}</s> <span class="pill bad">Cancelled</span>
      {:else}
        {event.name}
      {/if}
    </h1>
    <div class="when">
      <p>{dateRange(event.startDate, event.endDate)}</p>
      {#if event.location}<p>{event.location}</p>{/if}
    </div>

    {#if editing}
      <EventForm
        initial={{
          name: event.name,
          startDate: event.startDate,
          endDate: event.endDate,
          location: event.location ?? "",
        }}
        submitLabel="Save"
        onSave={save}
        onCancel={() => (editing = false)}
      />
    {/if}

    {#if error}<p role="alert">{error}</p>{/if}

    <ul aria-label="Counts" class="counts stats">
      <li>Going <span class="big">{counts.going}</span></li>
      <li>Not going <span class="big">{counts.notGoing}</span></li>
      <li>No answer <span class="big">{counts.noAnswer}</span></li>
    </ul>

    <ul class="scouts">
      {#each active as s (s.id)}
        {@const doc = byScout.get(s.id)}
        {@const last = doc?.history.at(-1)}
        <li>
          <span class="name"><ScoutLink scout={s} /></span>
          <div role="radiogroup" aria-label={fullName(s)}>
            {#each choices as c (c.label)}
              <label class="c-{c.value ?? 'clear'}">
                <input
                  type="radio"
                  name="rsvp-{s.id}"
                  aria-label={c.label}
                  checked={shown(s.id) === c.value}
                  onchange={() => void choose(s, c.value)}
                />
                {c.text}
              </label>
            {/each}
          </div>
          {#if doc && last}
            <details>
              <summary>
                {last.answer === null ? `Cleared by ${who(last)}` : who(last)}
              </summary>
              <ol aria-label="History">
                {#each doc.history as e, i (i)}
                  <li>
                    {e.answer === null ? "Cleared" : answerText[e.answer]}, {who(
                      e,
                    )}
                  </li>
                {/each}
              </ol>
            </details>
          {/if}
        </li>
      {/each}
    </ul>

    {#if !editing}
      <div class="actions row">
        <button type="button" onclick={() => (editing = true)}>Edit</button>
        <button
          type="button"
          disabled={busy}
          onclick={() => void setCancelled(!event.cancelled)}
        >
          {event.cancelled ? "Uncancel Event" : "Cancel Event"}
        </button>
        <button
          bind:this={deleteButton}
          type="button"
          disabled={inUse || busy}
          aria-describedby={inUse ? "delete-reason" : undefined}
          class="danger"
          onclick={() => void askDelete()}>Delete Event</button
        >
        {#if inUse}
          <p id="delete-reason" class="reason">
            Scouts have RSVP history for this Event, so it can't be deleted.
            Cancel it instead.
          </p>
        {:else if confirmingDelete}
          <span
            class="confirm-group"
            role="group"
            aria-label="Delete {event.name}?"
          >
            <button
              id="delete-yes"
              type="button"
              class="danger solid"
              disabled={busy}
              onclick={() => void remove()}
            >
              Yes, delete {event.name}
            </button>
            <button
              bind:this={keepButton}
              type="button"
              onclick={() => void keepIt()}
            >
              Keep it
            </button>
          </span>
        {/if}
      </div>
    {/if}
  {/if}
</main>

<style>
  .confirm-group {
    /* A real box (not display: contents) so the group role survives. */
    display: inline-flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .when {
    display: grid;
    gap: 0.25rem;
    color: var(--muted);
  }
  .counts li {
    color: var(--muted);
    font-size: 0.875rem;
    font-weight: 700;
  }
  /* Each answer in its own colour, as on the Kiosk: Going green, Not going
     red, No answer muted. */
  .counts li:nth-child(1) .big {
    color: var(--ok);
  }
  .counts li:nth-child(2) .big {
    color: var(--bad);
  }
  .counts li:nth-child(3) .big {
    color: var(--muted);
  }
  [role="radiogroup"] label.c-going:has(input:checked) {
    background: var(--ok);
    color: var(--on-ok);
  }
  /* "Not going" chosen reads red, "—" (cleared) neutral, as in the prototype. */
  [role="radiogroup"] label.c-notGoing:has(input:checked) {
    background: var(--bad);
    color: var(--on-bad);
  }
  [role="radiogroup"] label.c-clear:has(input:checked) {
    background: var(--bg);
    color: var(--ink);
  }
</style>
