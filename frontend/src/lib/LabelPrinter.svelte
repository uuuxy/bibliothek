<script>
  import { onMount } from "svelte";
  import { labelStore } from "./stores/labels.svelte.js";
  import LabelSettings from "./components/labels/LabelSettings.svelte";
  import LabelPreview from "./components/labels/LabelPreview.svelte";

  onMount(() => {
    labelStore.loadClassGroups();
  });
</script>

<div class="w-full space-y-6 no-print text-slate-800 animate-fade-in">
  
  <!-- Header Info -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-end gap-4 border-b border-slate-100 pb-5">
    <button onclick={labelStore.triggerPrint} disabled={labelStore.finalLabels.filter(lbl => !lbl.isBlank).length === 0} class="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:bg-slate-200 disabled:text-slate-400 disabled:cursor-not-allowed text-white font-bold transition-all flex items-center gap-2 shadow-xs cursor-pointer">
      <span>🖨️ A4-Bogen drucken</span>
    </button>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
    <LabelSettings />
    <LabelPreview />
  </div>
</div>


