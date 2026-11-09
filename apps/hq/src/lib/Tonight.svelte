<script lang="ts">
  // Tonight tab (spec stories 44–52): today's Meeting live. Counts, the
  // Checked in list (newest first, with Undo and an inline confirm) and the
  // Not checked in list (Leader check-in). A phone never starts a Meeting.
  import { tick } from "svelte";
  import { resolve } from "$app/paths";
  import Announcer from "./Announcer.svelte";
  import type { Api } from "./api";
  import type { Attendance, RSVPRollbackItem, Scout } from "./contract.gen";
  import { docsOf } from "./events";
  import { focusAfterRemoval, focusedRow, nextRowButton } from "./focus";
  import { clockTime, localDate } from "./dates";
  import { guardedActions } from "./guarded.svelte";
  import { asleep, meetingOver } from "./kiosk-flow";
  import type { Live, WithId } from "./live";
  import ScoutLink from "./ScoutLink.svelte";
  import { byName, fullName } from "./scouts";
  import UndoConfirm, { loadRollback } from "./UndoConfirm.svelte";

  let {
    api,
    live,
    now = () => new Date(),
  }: { api: Api; live: Live; now?: () => Date } = $props();

  // svelte-ignore state_referenced_locally
  const meetingId = localDate(now());
  // svelte-ignore state_referenced_locally
  const meeting = live.meeting(meetingId);
  // svelte-ignore state_referenced_locally
  const attendance = live.attendance(meetingId);
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();
  // svelte-ignore state_referenced_locally
  const rsvps = live.rsvps();

  // svelte-ignore state_referenced_locally
  const kioskSession = live.currentKioskSession(meetingId);

  // Ticked once a second: the Kiosk falls asleep by the clock, with no
  // snapshot to say so.
  // svelte-ignore state_referenced_locally
  let at = $state(now());
  $effect(() => {
    const timer = setInterval(() => {
      at = now();
    }, 1000);
    return () => clearInterval(timer);
  });

  // The Kiosk header: awake (with the latest check-in), asleep (Wake), or over
  // for the night. null while loading or when no Kiosk session is live.
  const kiosk = $derived.by(() => {
    if ($kioskSession.state !== "ready" || !$kioskSession.doc) return null;
    const lastActivityAt = $kioskSession.doc.lastActivityAt.toDate();
    if (meetingOver(meetingId, at)) return { state: "over" } as const;
    return asleep({ meetingId, lastActivityAt }, at)
      ? ({ state: "asleep" } as const)
      : ({ state: "awake" } as const);
  });

  // Spoken, not shown: Leader actions and the Kiosk falling asleep (polite
  // region), and the Find a Scout result (its own region, debounced, so
  // typing never overwrites an action or chatters per keystroke).
  let announcement = $state("");
  let wasAwake = false;
  $effect(() => {
    const state = kiosk?.state;
    if (wasAwake && state === "asleep") announcement = "Kiosk asleep";
    wasAwake = state === "awake";
  });

  // Leader uid → name, for "Leader: <name>". Unknown → just "Leader".
  let leaderNames = $state<Record<string, string>>({});
  $effect(() => {
    api.leaders().then(
      ({ leaders }) => {
        leaderNames = Object.fromEntries(
          leaders.map((l) => [l.uid, l.displayName]),
        );
      },
      () => {},
    );
  });

  const scoutsById = $derived(
    $scouts.state === "ready"
      ? new Map($scouts.docs.map((s) => [s.id, s]))
      : new Map<string, WithId<Scout>>(),
  );
  const checkedIn = $derived(
    $meeting.state === "ready" && $meeting.doc ? $meeting.doc.checkedIn : {},
  );

  // A check-in is shown while the Meeting doc still points at it.
  const present = $derived(
    $attendance.state === "ready"
      ? $attendance.docs
          .filter((a) => !a.undoneAt && checkedIn[a.scoutId] === a.id)
          .sort((a, b) => b.at.toMillis() - a.at.toMillis())
      : [],
  );
  // The newest non-undone check-in, any method; null while nobody is in.
  const lastCheckIn = $derived(present[0]?.at.toDate() ?? null);
  const notYet = $derived(
    [...scoutsById.values()]
      .filter((s) => s.active && !(s.id in checkedIn))
      .sort(byName),
  );

  // "Find a Scout": filters Not checked in by first or last name, any
  // substring, case-insensitive. List order is unchanged.
  let query = $state("");
  const needle = $derived(query.trim().toLowerCase());
  const shown = $derived(
    needle
      ? notYet.filter((s) => fullName(s).toLowerCase().includes(needle))
      : notYet,
  );
  const noMatch = $derived(needle && shown.length === 0);
  const searchResult = $derived(
    !needle
      ? ""
      : noMatch
        ? `No Scout matches “${query.trim()}”.`
        : `${shown.length} ${shown.length === 1 ? "Scout matches" : "Scouts match"}`,
  );

  let spokenResult = $state("");
  $effect(() => {
    const text = searchResult;
    if (!text) {
      spokenResult = "";
      return;
    }
    const timer = setTimeout(() => (spokenResult = text), 400);
    return () => clearTimeout(timer);
  });
  let searchInput: HTMLInputElement | undefined = $state();
  let notCheckedInHeading: HTMLHeadingElement | undefined = $state();
  let checkedInHeading: HTMLHeadingElement | undefined = $state();

  // RSVPs given in each check-in: RSVP docs with an entry carrying its id.
  // Still loading or unreadable counts as none, so the list never waits on it.
  const rsvpsGiven = $derived.by(() => {
    const given: Record<string, number> = {};
    for (const r of docsOf($rsvps)) {
      const ids = new Set(r.history.map((h) => h.attendanceId));
      for (const id of ids) {
        if (id !== null) given[id] = (given[id] ?? 0) + 1;
      }
    }
    return given;
  });

  function nameOf(scoutId: string): string {
    const s = scoutsById.get(scoutId);
    return s ? fullName(s) : "Unknown Scout";
  }

  function methodLabel(a: Attendance): string {
    if (a.method === "nameList") return "Name list";
    if (a.method === "handoff") return "Handoff";
    const name = leaderNames[a.by];
    return name ? `Leader: ${name}` : "Leader";
  }

  let confirming = $state<string | null>(null); // attendance id awaiting Yes/Cancel
  // The answers confirming's Undo rolls back; null when the preview failed.
  let rollback = $state<RSVPRollbackItem[] | null>(null);
  const guard = guardedActions();
  const busy = $derived(guard.busy);
  const message = $derived(guard.message);

  /** Runs a guarded call; a success closes the inline confirm. */
  async function run(
    failure: string,
    action: () => Promise<unknown>,
    done = "",
  ): Promise<boolean> {
    const ok = await guard.run(failure, action);
    if (ok) {
      confirming = null;
      announcement = done;
    }
    return ok;
  }

  // Undo buttons by attendance id, to hand focus back on Cancel.
  const undoButtons: Record<string, HTMLButtonElement | null> = {};
  async function cancelUndo(id: string) {
    confirming = null;
    await tick();
    undoButtons[id]?.focus();
  }

  const wake = () =>
    run("Couldn't wake the Kiosk. Try again.", () => api.wakeKiosk());
  async function checkIn(s: WithId<Scout>) {
    // The row leaves when the snapshot lands: hand focus to the next row.
    const row = focusedRow();
    const next = nextRowButton(row);
    const ok = await run(
      `Couldn't check in ${fullName(s)}. Try again.`,
      () => api.leaderCheckIn(meetingId, s.id),
      `${fullName(s)} checked in`,
    );
    if (ok)
      void focusAfterRemoval(
        row,
        () => next,
        () => searchInput,
        () => notCheckedInHeading,
      );
  }
  /** Previews the RSVP rollback, then shows the inline confirm. */
  async function askUndo(a: WithId<Attendance>) {
    // loadRollback never throws: a failed preview is null.
    const ok = await guard.run("", async () => {
      rollback = await loadRollback(api, meetingId, a.id);
    });
    if (ok) confirming = a.id;
  }
  async function undo(a: WithId<Attendance>) {
    const row = focusedRow();
    const next = nextRowButton(row);
    const ok = await run(
      `Couldn't undo ${nameOf(a.scoutId)}'s check-in. Try again.`,
      () => api.undoCheckIn(meetingId, a.id),
      `${nameOf(a.scoutId)}'s check-in undone`,
    );
    if (ok)
      void focusAfterRemoval(
        row,
        () => next,
        () => checkedInHeading,
      );
  }
</script>

<svelte:head><title>Tonight · Troop 813 HQ</title></svelte:head>

<main>
  <h1>Tonight</h1>
  <Announcer text={announcement} />
  <Announcer text={spokenResult} />

  {#if $meeting.state === "ready" && !$meeting.doc}
    <p>No Meeting today: the Admin starts it on the Kiosk</p>
    <p><a href={resolve("/(leader)/meetings/new")}>Add a past Meeting</a></p>
  {:else if $meeting.state === "error" || $attendance.state === "error" || $scouts.state === "error"}
    <p role="alert">Couldn't load tonight's Meeting. Reload to try again.</p>
  {:else if $meeting.state === "loading" || $attendance.state === "loading" || $scouts.state === "loading"}
    <p role="status">Loading…</p>
  {:else}
    {#if kiosk}
      <p class="kiosk-status banner" class:warn={kiosk.state !== "awake"}>
        {#if kiosk.state === "awake"}
          <strong>Kiosk awake</strong>
          · {lastCheckIn
            ? `last check-in ${clockTime(lastCheckIn)}`
            : "no check-ins yet"}
        {:else if kiosk.state === "asleep"}
          <strong>Kiosk asleep</strong>
          <button type="button" class="primary" disabled={busy} onclick={wake}>
            Wake Kiosk
          </button>
        {:else}
          <strong>Kiosk closed for the night</strong>
        {/if}
      </p>
    {/if}

    <p class="counts stats">
      <strong>{Object.keys(checkedIn).length} checked in</strong>
      <span>{notYet.length} not yet</span>
    </p>

    {#if message}<p role="alert">{message}</p>{/if}

    <h2 bind:this={checkedInHeading} tabindex="-1">Checked in</h2>
    <ul aria-label="Checked in">
      {#each present as a (a.id)}
        {@const who = scoutsById.get(a.scoutId)}
        <li>
          <span class="name"
            >{#if who}<ScoutLink scout={who} />{:else}{nameOf(
                a.scoutId,
              )}{/if}</span
          >
          <span class="time">{clockTime(a.at.toDate())}</span>
          <span class="method">{methodLabel(a)}</span>
          {#if rsvpsGiven[a.id]}
            {@const n = rsvpsGiven[a.id]}
            <span class="rsvps">{n} {n === 1 ? "RSVP" : "RSVPs"}</span>
          {/if}
          {#if confirming === a.id}
            <UndoConfirm
              name={nameOf(a.scoutId)}
              {rollback}
              {busy}
              onConfirm={() => undo(a)}
              onCancel={() => void cancelUndo(a.id)}
            />
          {:else}
            <button
              bind:this={undoButtons[a.id]}
              type="button"
              aria-label="Undo {nameOf(a.scoutId)}"
              disabled={busy}
              onclick={() => askUndo(a)}
            >
              Undo
            </button>
          {/if}
        </li>
      {/each}
    </ul>

    <h2 bind:this={notCheckedInHeading} tabindex="-1">Not checked in</h2>
    {#if notYet.length > 0}
      <label>
        Find a Scout
        <input
          type="search"
          autocomplete="off"
          enterkeyhint="search"
          bind:this={searchInput}
          bind:value={query}
        />
      </label>
    {/if}
    {#if noMatch}
      <p class="no-match" aria-hidden="true">
        No Scout matches “{query.trim()}”.
      </p>
    {/if}
    <ul aria-label="Not checked in">
      {#each shown as s (s.id)}
        <li>
          <span class="name"><ScoutLink scout={s} /></span>
          {#if $meeting.state === "ready" && $meeting.doc?.notFound.includes(s.id)}
            <span class="not-found pill warn">Not found at the Kiosk</span>
          {/if}
          <button
            type="button"
            aria-label="Check in {fullName(s)}"
            disabled={busy}
            onclick={() => checkIn(s)}
          >
            Check in
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</main>
