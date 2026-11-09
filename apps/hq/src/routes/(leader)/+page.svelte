<script lang="ts">
  // Tonight: today's Meeting, live. Any Leader; Go and the rules refuse everyone else.
  import Tonight from "#lib/Tonight.svelte";
  import { getHq } from "#lib/hq.ts";

  const { api, live } = getHq();
</script>

<Tonight {api} {live} />
