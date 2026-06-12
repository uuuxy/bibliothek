<script>
  import { undoToasts, removeUndoToast } from "./undoToastStore.svelte.js";
  import { slide } from "svelte/transition";
  import { backOut } from "svelte/easing";
</script>

<div class="fixed top-24 left-1/2 -translate-x-1/2 z-60 flex flex-col gap-3 items-center pointer-events-none">
  {#each undoToasts as toast (toast.id)}
    <div 
      transition:slide={{ duration: 300, easing: backOut }} 
      class="pointer-events-auto flex items-center gap-6 bg-slate-800 text-white pl-5 pr-2 py-2 rounded-full shadow-lg border border-slate-700/50"
    >
      <span class="text-sm font-medium tracking-wide">{toast.message}</span>
      <button 
        onclick={() => {
          toast.onUndo();
          removeUndoToast(toast.id);
        }}
        class="text-blue-400 font-bold text-xs tracking-wider px-3 py-1.5 hover:bg-slate-700 hover:text-blue-300 rounded-full transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/50"
      >
        RÜCKGÄNGIG
      </button>
    </div>
  {/each}
</div>
