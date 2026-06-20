<script>
  import StudentIdDesigner from "./StudentIdDesigner.svelte";
  import LabelPrinter from "./LabelPrinter.svelte";
  let activeTab = $state("labels");
</script>

<div class="w-full flex flex-col h-full bg-slate-50">
  <div class="px-8 pt-8 pb-4 border-b border-slate-200 bg-white shadow-sm flex-shrink-0">
    <div class="max-w-6xl mx-auto flex items-end justify-between">
      <div>
        <h2 class="text-3xl font-bold text-slate-900 tracking-tight">Druck-Center</h2>
        <p class="text-slate-500 mt-1">Generiere Etiketten und Ausweise für den Schulbetrieb.</p>
      </div>
    </div>
    
    <div class="max-w-6xl mx-auto mt-6 flex gap-6">
      <button 
        onclick={() => activeTab = "labels"}
        class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'labels' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
      >
        Buch-Etiketten
      </button>
      <button 
        onclick={() => activeTab = "ids"}
        class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'ids' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
      >
        Schülerausweise
      </button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if activeTab === "labels"}
      <div class="animate-fade-in h-full">
        <LabelPrinter />
      </div>
    {:else}
      <div class="animate-fade-in h-full">
        <StudentIdDesigner />
      </div>
    {/if}
  </div>
</div>
