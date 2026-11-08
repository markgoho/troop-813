<script lang="ts">
  // Event page (spec stories 70–74): counts, every active Scout with a
  // Going / Not / — switch (— clears, so the Kiosk asks again), who answered
  // last with full history on tap, and Edit, Cancel / Uncancel, Delete.
  // Reads are live (onSnapshot); every change goes through the API.
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
    } catch {
      delete pending[s.id];
      error = `Couldn't save ${s.firstName}'s RSVP. Try again.`;
    }
  }

  function who(e: RSVPEntry): string {
    return `${e.attendanceId === null ? "Leader" : "Kiosk"} · ${moment(e.at.toDate())}`;
  }

  async function act(f: () => Promise<void>, failed: string) {
    error = null;
    busy = true;
    try {
      await f();
    } catch (e) {
      error =
        e instanceof ApiError && e.body?.code === "in_use"
          ? "Scouts have RSVP history for this Event, so it can't be deleted. Cancel it instead."
          : failed;
    } finally {
      busy = false;
    }
  }

  async function save(input: EventInput) {
    await api.updateEvent(eventId, input);
    editing = false;
  }

  const setCancelled = (cancelled: boolean) =>
    act(
      () => api.setEventCancelled(eventId, cancelled),
      "Couldn't update the Event. Try again.",
    );

  const remove = () =>
    act(async () => {
      await api.deleteEvent(eventId);
      onDeleted();
    }, "Couldn't delete the Event. Try again.");
</script>

<main>
  {#if $events.state === "error"}
    <p role="alert">Couldn't load this Event. Reload to try again.</p>
  {:else if $events.state === "loading"}
    <p>Loading…</p>
  {:else if !event}
    <p>This Event doesn't exist (it may have been deleted).</p>
  {:else}
    <h1>
      {#if event.cancelled}
        <s>{event.name}</s> <span class="pill">Cancelled</span>
      {:else}
        {event.name}
      {/if}
    </h1>
    <p>{dateRange(event.startDate, event.endDate)}</p>
    {#if event.location}<p>{event.location}</p>{/if}

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
    {:else}
      <div class="actions">
        <button type="button" onclick={() => (editing = true)}>Edit</button>
        <button
          type="button"
          disabled={busy}
          onclick={() => void setCancelled(!event.cancelled)}
        >
          {event.cancelled ? "Uncancel Event" : "Cancel Event"}
        </button>
        <button
          type="button"
          disabled={inUse || busy}
          aria-describedby={inUse ? "delete-reason" : undefined}
          onclick={() => (confirmingDelete = true)}>Delete Event</button
        >
        {#if inUse}
          <p id="delete-reason" class="reason">
            Scouts have RSVP history for this Event, so it can't be deleted.
            Cancel it instead.
          </p>
        {:else if confirmingDelete}
          <button type="button" disabled={busy} onclick={() => void remove()}>
            Yes, delete {event.name}
          </button>
          <button type="button" onclick={() => (confirmingDelete = false)}>
            Keep it
          </button>
        {/if}
      </div>
    {/if}

    {#if error}<p role="alert">{error}</p>{/if}

    <ul aria-label="Counts" class="counts">
      <li>Going {counts.going}</li>
      <li>Not going {counts.notGoing}</li>
      <li>No answer {counts.noAnswer}</li>
    </ul>

    <ul class="scouts">
      {#each active as s (s.id)}
        {@const doc = byScout.get(s.id)}
        {@const last = doc?.history.at(-1)}
        <li>
          <ScoutLink scout={s} />
          <div role="radiogroup" aria-label="{s.firstName} {s.lastName}">
            {#each choices as c (c.label)}
              <label>
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
  {/if}
</main>

<style>
  .pill {
    border-radius: 999px;
    padding: 0 0.5em;
    background: #e5e7eb;
    font-size: 0.6em;
    vertical-align: middle;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5em;
    align-items: center;
  }
  .reason {
    color: #666;
    font-size: 0.875em;
    margin: 0;
  }
  .counts {
    display: flex;
    gap: 1.5em;
    list-style: none;
    padding: 0;
  }
  .scouts > li {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75em;
    align-items: baseline;
    padding: 0.25em 0;
  }
  [role="radiogroup"] {
    display: inline-flex;
    gap: 0.5em;
  }
</style>
