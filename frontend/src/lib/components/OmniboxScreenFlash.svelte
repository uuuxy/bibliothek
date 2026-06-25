<script>
  import { omniboxStore } from "../stores/omnibox.svelte.js";
</script>

<!-- ── Screen-edge flash overlay (Apple-style full-border glow) ── -->
{#if omniboxStore.screenFlash}
  <div class="screen-flash screen-flash--{omniboxStore.screenFlash}" aria-hidden="true"></div>
{/if}

<style>
  /* ── Screen-edge flash overlay ─────────────────────────────── */
  .screen-flash {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 9999;
    border-radius: 0;
    animation: screen-flash-fade 300ms ease-out forwards;
  }
  .screen-flash--success {
    box-shadow:
      inset 0 0 0 6px rgba(16, 185, 129, 0.85),  /* emerald-500 */
      inset 0 0 60px 10px rgba(16, 185, 129, 0.18);
  }
  .screen-flash--error {
    box-shadow:
      inset 0 0 0 6px rgba(239, 68, 68, 0.85),   /* red-500 */
      inset 0 0 60px 10px rgba(239, 68, 68, 0.18);
  }
  .screen-flash--warning {
    box-shadow:
      inset 0 0 0 6px rgba(245, 158, 11, 0.85),  /* amber-500 */
      inset 0 0 60px 10px rgba(245, 158, 11, 0.18);
  }
  @keyframes screen-flash-fade {
    0%   { opacity: 1; }
    60%  { opacity: 0.9; }
    100% { opacity: 0; }
  }
</style>
