<script lang="ts">
  // Every Leader screen sits in this group, behind the role gate. The Kiosk
  // (/kiosk, a later ticket) lives outside it with its own credential.
  import type { Snippet } from "svelte";
  import RoleGate from "#lib/RoleGate.svelte";
  import LeaderShell from "#lib/LeaderShell.svelte";
  import { provideHq } from "#lib/hq.ts";

  let { children: page }: { children: Snippet } = $props();

  const { session, api } = provideHq();
</script>

<RoleGate {session} {api}>
  {#snippet children(me)}
    <LeaderShell {me}>{@render page()}</LeaderShell>
  {/snippet}
</RoleGate>
