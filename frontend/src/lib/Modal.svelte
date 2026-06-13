<script>
  /**
   * Modal — generic overlay container that accepts snippet render-props.
   *
   * Usage:
   *   <Modal open={showModal} onclose={() => showModal = false} size="md">
   *     {#snippet header()}<h3>Titel</h3>{/snippet}
   *     {#snippet children()}<p>Inhalt</p>{/snippet}
   *   </Modal>
   *
   * Props:
   *   open     — controls visibility
   *   onclose  — optional; if provided, an × button is rendered in the header bar
   *   size     — "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" (default: "md")
   *   header   — optional snippet: rendered inside the top bar (title area)
   *   children — required snippet: the modal body content
   */

  /** @type {{ open: boolean, onclose?: () => void, size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl', header?: import('svelte').Snippet, children: import('svelte').Snippet }} */
  let { open, onclose, size = "md", header, children } = $props();

  const sizeClass = $derived({
    sm: "max-w-sm",
    md: "max-w-md",
    lg: "max-w-lg",
    xl: "max-w-xl",
    "2xl": "max-w-2xl",
    "3xl": "max-w-3xl",
    "4xl": "max-w-4xl",
  }[size] ?? "max-w-md");
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-fade-in"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) onclose?.(); }}
  >
    <div class="bg-white border border-slate-200 w-full {sizeClass} rounded-3xl shadow-2xl overflow-hidden animate-scale-up">
      {#if header}
        <div class="p-6 border-b border-slate-100 bg-slate-50/50 flex items-center justify-between">
          {@render header()}
          {#if onclose}
            <button onclick={onclose} class="text-slate-400 hover:text-slate-600 font-bold text-lg leading-none cursor-pointer" aria-label="Schließen">×</button>
          {/if}
        </div>
      {/if}
      {@render children()}
    </div>
  </div>
{/if}
