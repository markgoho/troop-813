<script lang="ts">
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import EventPage from "#lib/EventPage.svelte";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

{#key page.params.id}
  <EventPage
    {api}
    {live}
    eventId={page.params.id ?? ""}
    onDeleted={() => goto(resolve("/(leader)/events"))}
  />
{/key}
