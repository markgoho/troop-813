<script lang="ts">
  // Leader tab bar. Access is the Admin's only (spec story 81); Go refuses
  // its endpoints to anyone else regardless.
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import CalendarCheck from "phosphor-svelte/lib/CalendarCheck";
  import Campfire from "phosphor-svelte/lib/Campfire";
  import Key from "phosphor-svelte/lib/Key";
  import Tent from "phosphor-svelte/lib/Tent";
  import UsersThree from "phosphor-svelte/lib/UsersThree";
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
    ><Campfire weight="thin" aria-hidden="true" />Tonight</a
  >
  <a
    href={resolve("/(leader)/meetings")}
    aria-current={current(resolve("/(leader)/meetings"))}
    ><CalendarCheck weight="thin" aria-hidden="true" />Meetings</a
  >
  <a
    href={resolve("/(leader)/events")}
    aria-current={current(resolve("/(leader)/events"))}
    ><Tent weight="thin" aria-hidden="true" />Events</a
  >
  <a
    href={resolve("/(leader)/roster")}
    aria-current={current(resolve("/(leader)/roster"))}
    ><UsersThree weight="thin" aria-hidden="true" />Roster</a
  >
  {#if role === "admin"}<a
      href={resolve("/(leader)/access")}
      aria-current={current(resolve("/(leader)/access"))}
      ><Key weight="thin" aria-hidden="true" />Access</a
    >{/if}
</nav>
