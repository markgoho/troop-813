<script lang="ts">
  // Events tab (spec stories 67–69): Add Event, then Upcoming and Past, each
  // row with "N going" among active Scouts. Live through onSnapshot.
  import { resolve } from "$app/paths";
  import EventForm from "./EventForm.svelte";
  import type { Api } from "./api";
  import type { Event, EventInput } from "./contract.gen";
  import { dateRange } from "./dates";
  import { activeScouts, docsOf, splitEvents, tally } from "./events";
  import type { Live, WithId } from "./live";

  let { api, live, today }: { api: Api; live: Live; today: string } = $props();

  // svelte-ignore state_referenced_locally
  const events = live.events();
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();
  // svelte-ignore state_referenced_locally
  const rsvps = live.rsvps();

  const active = $derived(activeScouts(docsOf($scouts)));
  const lists = $derived(
    $events.state === "ready" ? splitEvents($events.docs, today) : null,
  );

  async function add(input: EventInput) {
    await api.createEvent(input);
  }
</script>

{#snippet list(title: string, items: WithId<Event>[])}
  <section aria-labelledby="events-{title}">
    <h2 id="events-{title}">{title}</h2>
    {#if items.length === 0}
      <p>None.</p>
    {:else}
      <ul>
        {#each items as e (e.id)}
          <li>
            <a href={resolve("/(leader)/events/[id]", { id: e.id })}>
              {#if e.cancelled}
                <s class="name">{e.name}</s>
                <span class="pill bad">Cancelled</span>
              {:else}
                <span class="name">{e.name}</span>
              {/if}
            </a>
            <span class="meta">{dateRange(e.startDate, e.endDate)}</span>
            <span class="pill ok"
              >{tally(e.id, active, docsOf($rsvps)).going} going</span
            >
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/snippet}

<main>
  <h1>Events</h1>
  <EventForm submitLabel="Add Event" onSave={add} />
  {#if lists}
    {@render list("Upcoming", lists.upcoming)}
    {@render list("Past", lists.past)}
  {:else if $events.state === "error"}
    <p role="alert">Couldn't load Events. Reload to try again.</p>
  {:else}
    <p>Loading…</p>
  {/if}
</main>
