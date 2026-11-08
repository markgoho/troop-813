<script lang="ts">
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import AddPastMeeting from "#lib/AddPastMeeting.svelte";
  import { localDate } from "#lib/dates.ts";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

<AddPastMeeting
  {api}
  {live}
  today={localDate(new Date())}
  onCreated={(id) => goto(resolve("/(leader)/meetings/[id]", { id }))}
/>
