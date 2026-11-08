<script lang="ts">
  import { page } from "$app/state";
  import { getHq } from "#lib/hq.ts";
  import { localDate } from "#lib/dates.ts";
  import ScoutPage from "#lib/ScoutPage.svelte";

  const { api, live } = getHq();
  const today = localDate(new Date());
</script>

{#key page.params.id}
  <ScoutPage {api} {live} scoutId={page.params.id ?? ""} {today} />
{/key}
