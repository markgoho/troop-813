<script lang="ts">
  // Tonight tab (spec stories 44–52): today's Meeting live. Counts, the
  // Checked in list (newest first, with Undo and an inline confirm) and the
  // Not checked in list (Leader check-in). A phone never starts a Meeting.
  import { ApiError, type Api } from "./api";
  import type {
    Attendance,
    RSVPAnswer,
    RSVPRollbackItem,
    Scout,
  } from "./contract.gen";
  import { docsOf } from "./events";
  import { clockTime, localDate } from "./dates";
  import { asleep, meetingOver } from "./kiosk-flow";
  import type { Live, WithId } from "./live";
  import ScoutLink from "./ScoutLink.svelte";

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

  // The Kiosk header: awake (with its last activity), asleep (Wake), or over
  // for the night. null while loading or when no Kiosk session is live.
  const kiosk = $derived.by(() => {
    if ($kioskSession.state !== "ready" || !$kioskSession.doc) return null;
    const lastActivityAt = $kioskSession.doc.lastActivityAt.toDate();
    if (meetingOver(meetingId, at)) return { state: "over" } as const;
    return asleep({ meetingId, lastActivityAt }, at)
      ? ({ state: "asleep" } as const)
      : ({ state: "awake", lastActivityAt } as const);
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

  const fullName = (s: WithId<Scout>) => `${s.firstName} ${s.lastName}`;
  const byName = (a: WithId<Scout>, b: WithId<Scout>) =>
    a.firstName.localeCompare(b.firstName) ||
    a.lastName.localeCompare(b.lastName);

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
  const notYet = $derived(
    [...scoutsById.values()]
      .filter((s) => s.active && !(s.id in checkedIn))
      .sort(byName),
  );

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

  const answerText = (a: RSVPAnswer | null) =>
    a === "going" ? "Going" : a === "notGoing" ? "Not going" : "No answer";

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
  let busy = $state(false);
  let message = $state<string | null>(null);

  async function run(failure: string, action: () => Promise<unknown>) {
    if (busy) return;
    busy = true;
    message = null;
    try {
      await action();
      confirming = null;
    } catch (err) {
      message =
        err instanceof ApiError && err.status === 404
          ? `${failure} It may already have changed.`
          : failure;
    } finally {
      busy = false;
    }
  }

  const wake = () =>
    run("Couldn't wake the Kiosk. Try again.", () => api.wakeKiosk());
  const checkIn = (s: WithId<Scout>) =>
    run(`Couldn't check in ${fullName(s)}. Try again.`, () =>
      api.leaderCheckIn(meetingId, s.id),
    );
  /** Previews the RSVP rollback, then shows the inline confirm. */
  async function askUndo(a: WithId<Attendance>) {
    if (busy) return;
    busy = true;
    message = null;
    try {
      rollback = (await api.rsvpRollbackPreview(meetingId, a.id)).items;
    } catch {
      rollback = null;
    } finally {
      busy = false;
    }
    confirming = a.id;
  }
  const undo = (a: WithId<Attendance>) =>
    run(`Couldn't undo ${nameOf(a.scoutId)}'s check-in. Try again.`, () =>
      api.undoCheckIn(meetingId, a.id),
    );
</script>

<main>
  <h1>Tonight</h1>

  {#if $meeting.state === "ready" && !$meeting.doc}
    <p>No Meeting today: the Admin starts it on the Kiosk</p>
  {:else if $meeting.state === "error" || $attendance.state === "error" || $scouts.state === "error"}
    <p role="alert">Couldn't load tonight's Meeting. Reload to try again.</p>
  {:else if $meeting.state === "loading" || $attendance.state === "loading" || $scouts.state === "loading"}
    <p>Loading…</p>
  {:else}
    {#if kiosk}
      <p class="kiosk-status">
        {#if kiosk.state === "awake"}
          <strong>Kiosk awake</strong>
          · last activity {clockTime(kiosk.lastActivityAt)}
        {:else if kiosk.state === "asleep"}
          <strong>Kiosk asleep</strong>
          <button type="button" disabled={busy} onclick={wake}>
            Wake Kiosk
          </button>
        {:else}
          <strong>Kiosk closed for the night</strong>
        {/if}
      </p>
    {/if}

    <p class="counts">
      <strong>{Object.keys(checkedIn).length} checked in</strong>
      ·
      <span>{notYet.length} not yet</span>
    </p>

    {#if message}<p role="alert">{message}</p>{/if}

    <h2>Checked in</h2>
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
            <span class="confirm">
              <span>Undo {nameOf(a.scoutId)}'s check-in?</span>
              {#if rollback === null}
                <span>
                  Couldn't load which RSVPs this rolls back; Undo still rolls
                  them back.
                </span>
              {:else if rollback.length > 0}
                <span>This rolls back:</span>
                <ul aria-label="RSVPs this rolls back">
                  {#each rollback as item (item.eventId)}
                    <li>
                      {item.eventName}: {answerText(item.from)} → {answerText(
                        item.to,
                      )}
                    </li>
                  {/each}
                </ul>
              {/if}
              <button type="button" disabled={busy} onclick={() => undo(a)}>
                Yes, undo
              </button>
              <button
                type="button"
                disabled={busy}
                onclick={() => (confirming = null)}
              >
                Cancel
              </button>
            </span>
          {:else}
            <button
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

    <h2>Not checked in</h2>
    <ul aria-label="Not checked in">
      {#each notYet as s (s.id)}
        <li>
          <span class="name"><ScoutLink scout={s} /></span>
          {#if $meeting.state === "ready" && $meeting.doc?.notFound.includes(s.id)}
            <span class="not-found">Not found at the Kiosk</span>
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
