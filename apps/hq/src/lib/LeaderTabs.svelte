<script lang="ts">
  // Leader tab bar. Access is the Admin's only (spec story 81); Go refuses
  // its endpoints to anyone else regardless.
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import type { Role } from "./contract.gen";

  let { role }: { role: Role } = $props();

  // The tab whose section holds this page (Tonight owns only "/").
  const current = (href: string) =>
    (
      href === resolve("/(leader)")
        ? page.url.pathname === href
        : page.url.pathname.startsWith(href)
    )
      ? "page"
      : undefined;
</script>

<nav aria-label="Leader tabs">
  <a href={resolve("/(leader)")} aria-current={current(resolve("/(leader)"))}
    >Tonight</a
  >
  <a
    href={resolve("/(leader)/meetings")}
    aria-current={current(resolve("/(leader)/meetings"))}>Meetings</a
  >
  <a
    href={resolve("/(leader)/events")}
    aria-current={current(resolve("/(leader)/events"))}>Events</a
  >
  <a
    href={resolve("/(leader)/roster")}
    aria-current={current(resolve("/(leader)/roster"))}>Roster</a
  >
  {#if role === "admin"}<a
      href={resolve("/(leader)/access")}
      aria-current={current(resolve("/(leader)/access"))}>Access</a
    >{/if}
</nav>
