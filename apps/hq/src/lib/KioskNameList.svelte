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
  // (KioskRsvp; none pending → straight on). Its "Done, pass the tablet on"
  // starts the Handoff.
  // The next Scout comes from kiosk-flow over the live Meeting doc, after the
  // Scout this tablet just checked in (component state, so a reload falls
  // back to the name list).
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
  }: {
    api: Api;
    live: Live;
    kiosk: KioskClaims;
    /** The server says the session is over (22:00 or replaced). */
    onExpired: () => void;
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
  let saving = $state(false);
  let message = $state<string | null>(null);

  // The Scout this tablet last checked in; the Handoff continues after them.
  let latest = $state<string | null>(null);
  // True after a check-in until "No, I'm someone else" breaks the chain.
  let handingOff = $state(false);
  // The Scout the Handoff named when the chain broke ("<Scout> was next").
  let broken = $state<string | null>(null);
  // Not-found marks sent from here, until the live Meeting doc catches up.
  let sentNotFound = $state<string[]>([]);

  const flow = $derived({
    handoffOrder:
      $meeting.state === "ready" && $meeting.doc
        ? $meeting.doc.handoffOrder.filter((id) =>
            names.some((n) => n.id === id),
          )
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
  const offeredId = $derived(handingOff ? nextScout(flow, latest) : null);
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
        // The asleep screen and Wake arrive with #27.
        confirming = null;
        message = "Kiosk asleep, ask a Leader";
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
    latest = entry.id;
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
  <p>Loading…</p>
{:else if $meeting.state === "error" || $scouts.state === "error"}
  <p>Couldn't load the Meeting. Reload to try again.</p>
{:else if !$meeting.doc}
  <p>Today's Meeting hasn't started. Ask the Admin.</p>
{:else if rsvping}
  {#key rsvping.id}
    <KioskRsvp
      {api}
      {live}
      meetingId={kiosk.meetingId}
      scout={rsvping}
      onDone={() => (rsvping = null)}
      {onExpired}
    />
  {/key}
{:else if confirming}
  {@const entry = confirming}
  <h1>Are you {entry.label}?</h1>
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
  {#if message}<p role="alert">{message}</p>{/if}
{:else if offered}
  {@const entry = offered}
  <h1>Pass the tablet to {entry.label}</h1>
  <p>Are you {entry.label}?</p>
  <div class="choices">
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
  {#if message}<p role="alert">{message}</p>{/if}
{:else}
  <h1>{anyoneIn ? "Tap your name" : "Who's first?"}</h1>
  {#if due}<p>{due.label} was next</p>{/if}
  {#if message}<p role="alert">{message}</p>{/if}
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
          {done ? `✓ ${entry.label}` : entry.label}
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .names {
    list-style: none;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 0.75rem;
  }
  .names button {
    width: 100%;
  }
  .done {
    opacity: 0.45;
  }
  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
</style>
