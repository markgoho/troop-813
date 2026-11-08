<script lang="ts">
  // The Kiosk name list (spec stories 9–12): "Who's first?", every active
  // Scout by name-list label, checked-in Scouts greyed with a check mark.
  // Tap a name → "Are you <Scout>?" → Yes records Attendance (method nameList)
  // at once, then back to the list. Greyed comes from the live Meeting doc's
  // checkedIn, never from the API response, so an undo on a Leader's phone
  // or a reload shows up here by itself.
  //
  // After a check-in the Handoff (stories 13–16) names the next Scout:
  // Yes → handoff check-in; "No, I'm someone else" → the name list, which
  // says "<Scout> was next"; "I can't find <Scout>" → marked not found, a
  // different Scout is named. Nobody left to name → the name list.
  // Between a check-in and the Handoff, the Scout answers their pending RSVPs
  // (KioskRsvp; no upcoming Events → straight on). Its "Done, pass the tablet
  // on" starts the Handoff.
  // The next Scout comes from kiosk-flow over the live Meeting doc, after the
  // Scout last checked in.
  //
  // A reload, Back press or crash resumes from the server (story 22): when
  // the Meeting doc first loads, its lastCheckIn Scout, if still checked in
  // and with Events left to answer, gets their RSVP cards back; otherwise the
  // Handoff continues after them. No lastCheckIn means the name list.
  import { ApiError, type Api } from "./api";
  import KioskRsvp from "./KioskRsvp.svelte";
  import {
    type NameListEntry,
    nameList,
    nextScout,
    wasNext,
  } from "./kiosk-flow";
  import type { Live } from "./live";
  import type { KioskClaims } from "./session";

  let {
    api,
    live,
    kiosk,
    onExpired,
    onAsleep,
  }: {
    api: Api;
    live: Live;
    kiosk: KioskClaims;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
    /** The server says the session is asleep: the parent shows the asleep screen. */
    onAsleep: () => void;
  } = $props();

  // The parent re-creates this component for a new session.
  // svelte-ignore state_referenced_locally
  const meeting = live.meeting(kiosk.meetingId);
  // svelte-ignore state_referenced_locally
  const scouts = live.scouts();

  const names = $derived(
    $scouts.state === "ready" ? nameList($scouts.docs) : [],
  );
  const checkedIn = $derived(
    $meeting.state === "ready" && $meeting.doc ? $meeting.doc.checkedIn : {},
  );
  const anyoneIn = $derived(Object.keys(checkedIn).length > 0);

  let confirming = $state<NameListEntry | null>(null);
  // The Scout who just checked in and is answering RSVPs.
  let rsvping = $state<NameListEntry | null>(null);
  // True when rsvping was restored by a reload: no pending Events means no stage.
  let resumedRsvp = $state(false);
  let saving = $state(false);
  let message = $state<string | null>(null);

  // The Scout this tablet just checked in, until the live Meeting doc shows them.
  let latest = $state<string | null>(null);
  // The Scout the Handoff continues after: the last one checked in, here or
  // (after a reload) per the Meeting doc's lastCheckIn.
  let after = $state<string | null>(null);
  // True after a check-in until "No, I'm someone else" breaks the chain.
  let handingOff = $state(false);
  // Whether the stage was derived from the Meeting doc yet (once, on load).
  let resumed = $state(false);
  // The Scout the Handoff named when the chain broke ("<Scout> was next").
  let broken = $state<string | null>(null);
  // Not-found marks sent from here, until the live Meeting doc catches up.
  let sentNotFound = $state<string[]>([]);

  $effect(() => {
    if (resumed || $meeting.state !== "ready" || !$meeting.doc) return;
    if ($scouts.state !== "ready") return;
    resumed = true;
    const last = $meeting.doc.lastCheckIn;
    if (!last) return; // null, or a Meeting from before the field existed
    after = last;
    handingOff = true;
    const entry = names.find((n) => n.id === last);
    if (entry && last in checkedIn) {
      rsvping = entry;
      resumedRsvp = true;
    }
  });

  const activeIds = $derived(new Set(names.map((n) => n.id)));
  const flow = $derived({
    handoffOrder:
      $meeting.state === "ready" && $meeting.doc
        ? $meeting.doc.handoffOrder
        : [],
    // The Scout just checked in counts as in even before the snapshot lands.
    checkedIn: latest ? { ...checkedIn, [latest]: "" } : checkedIn,
    notFound: [
      ...($meeting.state === "ready" && $meeting.doc
        ? $meeting.doc.notFound
        : []),
      ...sentNotFound,
    ],
  });
  const offeredId = $derived(
    handingOff ? nextScout(flow, after, activeIds) : null,
  );
  const offered = $derived(names.find((n) => n.id === offeredId) ?? null);
  const dueId = $derived(wasNext(broken, flow));
  const due = $derived(names.find((n) => n.id === dueId) ?? null);

  function pick(entry: NameListEntry) {
    message = null;
    confirming = entry;
  }

  /** Runs one Kiosk call; maps a failure to the screen's message. */
  async function run(fail: string, call: () => Promise<unknown>) {
    saving = true;
    message = null;
    try {
      await call();
      return true;
    } catch (err) {
      const code = err instanceof ApiError ? err.body?.code : undefined;
      if (code === "session_expired") {
        onExpired();
      } else if (code === "session_asleep") {
        // The asleep screen covers this one; whatever the Scout was doing
        // is still here when a Leader wakes the Kiosk.
        onAsleep();
      } else {
        message = fail;
      }
      return false;
    } finally {
      saving = false;
    }
  }

  async function checkIn(entry: NameListEntry, method: "nameList" | "handoff") {
    const ok = await run("Couldn't check you in. Try again.", () =>
      api.kioskCheckIn({ scoutId: entry.id, method }),
    );
    if (!ok) return;
    confirming = null;
    rsvping = entry;
    resumedRsvp = false;
    latest = entry.id;
    after = entry.id;
    handingOff = true;
    broken = null;
  }

  async function cantFind(entry: NameListEntry) {
    const ok = await run("Couldn't save that. Try again.", () =>
      api.kioskNotFound({ scoutId: entry.id }),
    );
    if (ok) sentNotFound = [...sentNotFound, entry.id];
  }

  function someoneElse(entry: NameListEntry) {
    message = null;
    broken = entry.id;
    handingOff = false;
  }
</script>

{#if $meeting.state === "loading" || $scouts.state === "loading"}
  <p class="status">Loading…</p>
{:else if $meeting.state === "error" || $scouts.state === "error"}
  <p class="status">Couldn't load the Meeting. Reload to try again.</p>
{:else if !$meeting.doc}
  <p class="status">Today's Meeting hasn't started. Ask the Admin.</p>
{:else if !resumed}
  <p class="status">Loading…</p>
{:else if rsvping}
  {#key rsvping.id}
    <KioskRsvp
      {api}
      {live}
      meetingId={kiosk.meetingId}
      scout={rsvping}
      resume={resumedRsvp}
      onDone={() => (rsvping = null)}
      {onExpired}
      {onAsleep}
    />
  {/key}
{:else if confirming}
  {@const entry = confirming}
  <h1 class="hero">
    <span class="lead">Are you</span>
    <span class="bigname" style:--len={entry.label.length + 1}
      >{entry.label}?</span
    >
  </h1>
  <div class="choices">
    <button
      type="button"
      class="yes"
      disabled={saving}
      onclick={() => void checkIn(entry, "nameList")}
    >
      Yes, I'm {entry.label}
    </button>
    <button type="button" disabled={saving} onclick={() => (confirming = null)}>
      No
    </button>
  </div>
  {#if message}<p role="alert" class="alert">{message}</p>{/if}
{:else if offered}
  {@const entry = offered}
  <h1 class="hero">
    <span class="lead">Pass the tablet to</span>
    <span class="bigname" style:--len={entry.label.length}>{entry.label}</span>
  </h1>
  <p class="ask">Are you {entry.label}?</p>
  <div class="choices handoff">
    <button
      type="button"
      class="yes"
      disabled={saving}
      onclick={() => void checkIn(entry, "handoff")}
    >
      Yes
    </button>
    <button type="button" disabled={saving} onclick={() => someoneElse(entry)}>
      No, I'm someone else
    </button>
    <button
      type="button"
      disabled={saving}
      onclick={() => void cantFind(entry)}
    >
      I can't find {entry.label}
    </button>
  </div>
  {#if message}<p role="alert" class="alert">{message}</p>{/if}
{:else}
  <div class="listhead">
    <h1>{anyoneIn ? "Tap your name" : "Who's first?"}</h1>
    {#if due}<p class="sub">{due.label} was next</p>{/if}
  </div>
  {#if message}<p role="alert" class="alert">{message}</p>{/if}
  <ul aria-label="Scouts" class="names">
    {#each names as entry (entry.id)}
      {@const done = entry.id in checkedIn}
      <li>
        <button
          type="button"
          class:done
          disabled={done}
          onclick={() => pick(entry)}
        >
          {#if done}<span class="sr-only">✓</span>{/if}{entry.label}
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .hero {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .lead {
    font-family: var(--body);
    font-weight: 400;
    font-size: 1.5rem;
    letter-spacing: 0;
    color: var(--muted);
  }
  /* The Handoff's hero: as big as the stage allows, and shrunk to fit so a
     long name (--len characters) stays on one line instead of breaking. */
  .bigname {
    font-size: clamp(
      3rem,
      min(18cqi, 22vh, 11rem, calc(190cqi / var(--len, 8))),
      11rem
    );
    line-height: 0.95;
    color: var(--accent);
    white-space: nowrap;
  }
  @media (prefers-reduced-motion: no-preference) {
    .bigname {
      animation: arrive 420ms var(--ease) both;
    }
  }
  /* Starts visible: a short rise into place, never a fade from nothing. */
  @keyframes arrive {
    from {
      transform: translateY(0.12em);
      opacity: 0.6;
    }
  }
  .ask {
    font-size: 1.75rem;
    font-weight: 700;
  }
  .handoff button {
    flex: 1 1 12rem;
  }
  .handoff .yes {
    flex-basis: 100%;
    min-height: 5rem;
    font-size: 1.75rem;
  }
  @media (orientation: portrait) {
    .handoff button,
    .handoff .yes {
      flex-basis: auto;
    }
  }
  /* The name list may use more of a landscape tablet than the one-question
     screens, so ~30 Scouts fit without scrolling. */
  .listhead,
  .names {
    align-self: center;
    width: min(76rem, 100vw - 3rem);
  }
  .listhead {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0.5rem 1.25rem;
  }
  .names {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(11rem, 1fr));
    gap: 0.75rem;
  }
  .names button {
    width: 100%;
    min-height: 4rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    text-align: left;
    font-size: 1.5rem;
  }
  .names .done {
    background: transparent;
    border-color: transparent;
    color: var(--muted);
    font-weight: 400;
    opacity: 1;
  }
  /* The check mark (spec story 11), drawn; the button text carries a hidden
     "✓" for screen readers. */
  .names .done::before {
    content: "";
    flex: none;
    width: 1.5rem;
    height: 1.5rem;
    background: var(--ok);
    mask: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='3' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M5 12.5l4.5 4.5L19 7.5'/%3E%3C/svg%3E")
      center / contain no-repeat;
  }
</style>
