<script lang="ts">
  import EventsTab from "#lib/EventsTab.svelte";
  import { localDate } from "#lib/dates.ts";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

<EventsTab {api} {live} today={localDate(new Date())} />
