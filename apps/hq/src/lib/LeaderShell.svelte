<script lang="ts">
  // Wraps every Leader screen: the tab bar, and the caller in context (getCaller).
  import type { Snippet } from "svelte";
  import LeaderTabs from "./LeaderTabs.svelte";
  import type { Me } from "./contract.gen";
  import { provideCaller } from "./hq";
  import "./leader.css";

  let { me, children }: { me: Me; children: Snippet } = $props();

  // RoleGate re-renders this when the caller changes, so reading once is enough.
  // svelte-ignore state_referenced_locally
  provideCaller(me);
</script>

<div class="leader">
  <header class="bar">
    <img src="/brand/troop-813-logo.png" alt="" />
    <span>Troop 813 HQ</span>
  </header>
  <LeaderTabs role={me.role} />
  {@render children()}
</div>
