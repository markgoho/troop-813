<script lang="ts">
  // Root layout: only the shared design tokens and base styles. Each face (the
  // Kiosk, the Leader screens) brings its own shell.
  import type { Snippet } from "svelte";
  import "../app.css";

  let { children }: { children: Snippet } = $props();
</script>

{@render children()}
