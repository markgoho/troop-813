<script lang="ts">
  // Leader tab bar. Access is the Admin's only (spec story 81); Go refuses
  // its endpoints to anyone else regardless.
  import { resolve } from "$app/paths";
  import type { Role } from "./contract.gen";

  let { role }: { role: Role } = $props();
</script>

<nav aria-label="Leader tabs">
  <a href={resolve("/(leader)")}>Tonight</a>
  <a href={resolve("/(leader)/meetings")}>Meetings</a>
  <a href={resolve("/(leader)/events")}>Events</a>
  <a href={resolve("/(leader)/roster")}>Roster</a>
  {#if role === "admin"}<a href={resolve("/(leader)/access")}>Access</a>{/if}
</nav>
