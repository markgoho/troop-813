<script lang="ts">
  // Tonight: a placeholder until the Tonight ticket lands.
  import HealthStatus from "#lib/HealthStatus.svelte";
  import { getHq } from "#lib/hq.ts";

  const { api } = getHq();
</script>

<main>
  <h1>Tonight</h1>
  <p>Tonight's Meeting will show here.</p>
  <HealthStatus {api} />
</main>
