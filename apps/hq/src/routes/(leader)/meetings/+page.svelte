<script lang="ts">
  import MeetingsTab from "#lib/MeetingsTab.svelte";
  import { localDate } from "#lib/dates.ts";
  import { getHq } from "#lib/hq.ts";

  const { live } = getHq();
</script>

<MeetingsTab {live} today={localDate(new Date())} />
