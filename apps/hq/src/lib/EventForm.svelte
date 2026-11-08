<script lang="ts">
  // Add or edit an Event: name, start date, end date, optional location.
  // onSave resolves when the API accepted it; the form then resets (add) or
  // the parent closes it (edit).
  import type { EventInput } from "./contract.gen";
  import { eventFormError } from "./events";

  let {
    initial = { name: "", startDate: "", endDate: "", location: "" },
    submitLabel,
    onSave,
    onCancel,
  }: {
    initial?: EventInput;
    submitLabel: string;
    onSave: (input: EventInput) => Promise<void>;
    onCancel?: () => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const start = { ...initial, location: initial.location ?? "" };
  let name = $state(start.name);
  let startDate = $state(start.startDate);
  let endDate = $state(start.endDate);
  let location = $state(start.location);
  let error = $state<string | null>(null);
  let saving = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = eventFormError({ name, startDate, endDate });
    if (error) return;
    const input: EventInput = { name: name.trim(), startDate, endDate };
    if (location.trim()) input.location = location.trim();
    saving = true;
    try {
      await onSave(input);
      name = start.name;
      startDate = start.startDate;
      endDate = start.endDate;
      location = start.location;
    } catch {
      error = "Couldn't save the Event. Try again.";
    } finally {
      saving = false;
    }
  }
</script>

<form onsubmit={submit}>
  <label>Name <input bind:value={name} required /></label>
  <label>Start date <input type="date" bind:value={startDate} required /></label
  >
  <label>End date <input type="date" bind:value={endDate} required /></label>
  <label>Location (optional) <input bind:value={location} /></label>
  {#if error}<p role="alert">{error}</p>{/if}
  <button type="submit" disabled={saving}>{submitLabel}</button>
  {#if onCancel}
    <button type="button" onclick={onCancel}>Cancel</button>
  {/if}
</form>
