<script>
  let { 
    incomingShipments, 
    showGreenFade,
    onSelectShipment
  } = $props();
</script>

<div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
  <div class="flex items-center justify-between border-b border-slate-100 pb-3">
    <h2 class="text-sm font-bold text-slate-800">Wareneingang</h2>
    <span class="text-[10px] bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase">Im Zulauf</span>
  </div>
  
  {#if !incomingShipments.length}
    <div class="py-8 text-center text-xs text-slate-400">🚚 Keine offenen Bestellungen im Zulauf.</div>
  {:else}
    <div class="max-h-60 overflow-y-auto space-y-2 {showGreenFade ? 'animate-green-fade' : ''} pr-1 custom-scrollbar">
      {#each incomingShipments as s}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div 
          class="p-3 border border-slate-200 rounded-lg bg-white hover:bg-blue-50/50 hover:border-blue-200 text-[11px] space-y-2 cursor-pointer transition-all group"
          onclick={() => onSelectShipment(s)}
        >
          <div class="flex justify-between items-center font-bold text-slate-700 group-hover:text-blue-700">
            <span class="truncate">{s.supplierName}</span>
            <span class="text-slate-400 font-medium shrink-0">{s.date}</span>
          </div>
          <div class="text-xs text-slate-500 font-medium">
            {s.items.reduce((sum, item) => sum + item.menge, 0)} Exemplare erwartet
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  @keyframes greenGlow {
    0% { background-color: rgba(16, 185, 129, 0.15); border-color: rgba(16, 185, 129, 0.45); transform: scale(1); }
    50% { background-color: rgba(16, 185, 129, 0.35); border-color: rgba(16, 185, 129, 0.9); transform: scale(1.02); }
    100% { background-color: transparent; border-color: rgba(226, 232, 240, 1); opacity: 0; transform: scale(0.95); }
  }
  .animate-green-fade { animation: greenGlow 1.5s cubic-bezier(0.4, 0, 0.2, 1) forwards; }
  
  .custom-scrollbar::-webkit-scrollbar {
    width: 4px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background-color: #cbd5e1;
    border-radius: 4px;
  }
</style>
