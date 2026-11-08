<script lang="ts">
  // Roster tab: any Leader. Go and the rules refuse everyone else.
  import Roster from "#lib/Roster.svelte";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

<Roster {api} {live} />
