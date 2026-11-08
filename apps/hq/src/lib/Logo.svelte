<script lang="ts">
  // The troop logo (troop-813-logo.svg, beside this file), inlined so it takes
  // the surrounding text colour (currentColor) in both themes. Its fine
  // detail (trees, stars, lettering) needs room: use it at 10rem or larger,
  // never as a small icon. Decorative: the text beside it names the troop.
  import svg from "./troop-813-logo.svg?raw";

  let {
    size = "12rem",
    class: cls = "",
  }: {
    /** Rendered width and height (CSS length); 10rem or more. */
    size?: string;
    class?: string;
  } = $props();
</script>

<span class="logo {cls}" style:--size={size} aria-hidden="true">
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
</style>
