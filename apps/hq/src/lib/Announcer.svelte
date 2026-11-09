<script lang="ts">
  // A polite, visually hidden live region: the result of a Leader action
  // ("Avery Quill checked in") spoken without moving focus.
  let { text }: { text: string } = $props();
</script>

<p class="sr-only" role="status" aria-live="polite">{text}</p>
