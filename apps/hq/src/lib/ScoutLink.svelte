<script lang="ts">
  // A Scout's name as a link to their page (spec story 66). Use this wherever
  // a screen shows a Scout's name so every name navigates the same way.
  import { resolve } from "$app/paths";
  import { fullName } from "./scouts";

  let {
    scout,
  }: { scout: { id: string; firstName: string; lastName: string } } = $props();
</script>

<a href={resolve("/(leader)/roster/[id]", { id: scout.id })}
  >{fullName(scout)}</a
>
