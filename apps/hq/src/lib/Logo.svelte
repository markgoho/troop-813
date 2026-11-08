<script lang="ts">
  // The troop logo (static/brand/troop-813-logo.svg), inlined so it takes the
  // surrounding text colour (currentColor) in both themes, and drawn with a
  // fixed on-screen line width so the monoline art stays crisp at any size.
  // Decorative: the text beside it names the troop.
  import svg from "../../static/brand/troop-813-logo.svg?raw";

  let {
    size = "2rem",
    line = 1,
    class: cls = "",
  }: {
    /** Rendered width and height (CSS length). */
    size?: string;
    /** Line width in CSS pixels, whatever the size. */
    line?: number;
    class?: string;
  } = $props();
</script>

<span
  class="logo {cls}"
  style:--size={size}
  style:--line="{line}px"
  aria-hidden="true"
>
  <!-- eslint-disable-next-line svelte/no-at-html-tags -- our own static asset -->
  {@html svg}
</span>

<style>
  .logo {
    display: inline-block;
    flex: none;
    width: var(--size);
    height: var(--size);
  }
  .logo :global(svg) {
    display: block;
    width: 100%;
    height: 100%;
  }
  .logo :global(svg *) {
    vector-effect: non-scaling-stroke;
  }
  .logo :global(svg > g) {
    stroke-width: var(--line);
  }
</style>
