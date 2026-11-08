<script lang="ts">
  // Wraps every Leader screen: the tab bar, and the caller in context (getCaller).
  import type { Snippet } from "svelte";
  import LeaderTabs from "./LeaderTabs.svelte";
  import Logo from "./Logo.svelte";
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
    <Logo />
    <span>Troop 813 HQ</span>
  </header>
  <LeaderTabs role={me.role} />
  {@render children()}
</div>
