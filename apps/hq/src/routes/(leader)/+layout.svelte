<script lang="ts">
  // Every Leader screen sits in this group, behind the role gate. The Kiosk
  // (/kiosk, a later ticket) lives outside it with its own credential.
  import type { Snippet } from "svelte";
  import RoleGate from "#lib/RoleGate.svelte";
  import { provideHq } from "#lib/hq.ts";

  let { children }: { children: Snippet } = $props();

  const { session, api } = provideHq();
</script>

<RoleGate {session} {api}>
  {@render children()}
</RoleGate>
