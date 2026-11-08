<script lang="ts">
  // Scout page (spec stories 59–61): X/Y Meetings present since the start
  // date, each Meeting newest first, and the Scout's RSVPs for upcoming and
  // answered Events. History comes from Go (it owns the start rule); the
  // name and RSVPs are live.
  import { resolve } from "$app/paths";
  import type { Api } from "./api";
  import type { ScoutHistory } from "./contract.gen";
  import { dateRange, shortDay } from "./dates";
  import { docsOf, splitEvents } from "./events";
  import type { Live } from "./live";
  import { fullName } from "./scouts";

  let {
    api,
    live,
    scoutId,
    today,
  }: { api: Api; live: Live; scoutId: string; today: string } = $props();

  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();
  // svelte-ignore state_referenced_locally
  const events = live.events();
  // svelte-ignore state_referenced_locally
  const rsvps = live.rsvps();

  type Loaded =
    | { state: "loading" }
    | { state: "ready"; history: ScoutHistory }
    | { state: "missing" }
    | { state: "error" };

  let loaded = $state<Loaded>({ state: "loading" });

  $effect(() => {
    const id = scoutId;
    loaded = { state: "loading" };
    let current = true;
    api.scoutHistory(id).then(
      (history) => {
        if (current) loaded = { state: "ready", history };
      },
      (err: unknown) => {
        if (!current) return;
        const status = (err as { status?: number }).status;
        loaded = { state: status === 404 ? "missing" : "error" };
      },
    );
    return () => {
      current = false;
    };
  });

  const scout = $derived(docsOf($scouts).find((s) => s.id === scoutId));
  const missing = $derived(
    loaded.state === "missing" || ($scouts.state === "ready" && !scout),
  );

  const answers = $derived(
    new Map(
      docsOf($rsvps)
        .filter((r) => r.scoutId === scoutId)
        .map((r) => [r.eventId, r.answer]),
    ),
  );
  // Upcoming Events always; past Events only when this Scout answered.
  const shown = $derived.by(() => {
    const { upcoming, past } = splitEvents(docsOf($events), today);
    return [...upcoming, ...past.filter((e) => answers.get(e.id) != null)];
  });

  const label = { going: "Going", notGoing: "Not going" } as const;
</script>

<main>
  {#if missing}
    <p>This Scout isn't on the Roster.</p>
  {:else}
    {#if scout}
      <h1>{fullName(scout)}</h1>
    {:else if $scouts.state === "error"}
      <p role="alert">Couldn't load this Scout. Reload to try again.</p>
    {:else}
      <h1>Loading…</h1>
    {/if}

    {#if loaded.state === "error"}
      <p role="alert">Couldn't load attendance. Reload to try again.</p>
    {:else if loaded.state === "ready"}
      <p>
        {loaded.history.present}/{loaded.history.total} Meetings present since {shortDay(
          loaded.history.since,
        )}
      </p>
      {#if loaded.history.meetings.length === 0}
        <p>No Meetings yet.</p>
      {:else}
        <ul aria-label="Meetings">
          {#each loaded.history.meetings as m (m.date)}
            <li>{shortDay(m.date)} {m.present ? "Present" : "Absent"}</li>
          {/each}
        </ul>
      {/if}
    {/if}

    <h2>Events</h2>
    {#if shown.length === 0}
      <p>No Events.</p>
    {:else}
      <ul aria-label="Events">
        {#each shown as e (e.id)}
          {@const a = answers.get(e.id)}
          <li>
            <a href={resolve("/(leader)/events/[id]", { id: e.id })}>{e.name}</a
            >
            {dateRange(e.startDate, e.endDate)}
            {a ? label[a] : "No answer"}
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</main>
