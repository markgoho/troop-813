<script lang="ts">
  // The Meeting Kiosk, outside the (leader) group: its own gate, its own
  // credential (ADR 0003). Single route, so Back can't leave the flow.
  import Kiosk from "#lib/Kiosk.svelte";
  import { provideHq } from "#lib/hq.ts";

  const { session, api, live } = provideHq();
</script>

<svelte:head><title>Meeting Kiosk · Troop 813 HQ</title></svelte:head>

<Kiosk {session} {api} {live} />
