<script lang="ts">
  import type { Api } from "./api";

  let { api }: { api: Api } = $props();

  const health = $derived(api.health());
</script>

{#await health}
  <p>Checking the API…</p>
{:then h}
  <p>API {h.status}</p>
  <p>Today is {h.today}</p>
{:catch}
  <p>API unreachable</p>
{/await}
