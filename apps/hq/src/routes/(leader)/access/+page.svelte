<script lang="ts">
  // Access tab: Admin only. Non-Admins never see the tab; Go refuses them anyway.
  import AccessList from "#lib/AccessList.svelte";
  import { getCaller, getHq } from "#lib/hq.ts";

  const { api } = getHq();
  const me = getCaller();
</script>

<svelte:head><title>Access · Troop 813 HQ</title></svelte:head>

{#if me.role === "admin"}
  <AccessList {api} myUid={me.uid} />
{:else}
  <main><p>Only the Admin manages Access.</p></main>
{/if}
