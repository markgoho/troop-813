<script lang="ts">
  import HealthStatus from "#lib/HealthStatus.svelte";
  import { httpApi } from "#lib/api.ts";
</script>

<main>
  <h1>Troop 813 HQ</h1>
  <HealthStatus api={httpApi()} />
</main>
