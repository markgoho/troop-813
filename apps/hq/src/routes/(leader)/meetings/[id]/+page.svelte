<script lang="ts">
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import MeetingPage from "#lib/MeetingPage.svelte";
  import { localDate } from "#lib/dates.ts";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

{#key page.params.id}
  <MeetingPage
    {api}
    {live}
    meetingId={page.params.id ?? ""}
    today={localDate(new Date())}
    onDeleted={() => goto(resolve("/(leader)/meetings"))}
  />
{/key}
