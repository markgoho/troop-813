<script lang="ts">
  // Every Leader screen sits in this group, behind the role gate. The Kiosk
  // (/kiosk) lives outside it with its own credential; a Kiosk session that
  // lands here is sent there.
  import type { Snippet } from "svelte";
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import RoleGate from "#lib/RoleGate.svelte";
  import LeaderShell from "#lib/LeaderShell.svelte";
  import { provideHq } from "#lib/hq.ts";

  let { children: page }: { children: Snippet } = $props();

  const { session, api } = provideHq();
</script>

<RoleGate {session} {api} onKiosk={() => void goto(resolve("/kiosk"))}>
  {#snippet children(me)}
    <LeaderShell {me}>{@render page()}</LeaderShell>
  {/snippet}
</RoleGate>
