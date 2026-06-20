<script>
  let { 
    incomingShipments, 
    showGreenFade,
    onOpenWareneingang
  } = $props();

  let totalItems = $derived(incomingShipments.reduce((sum, s) => sum + s.items.reduce((s2, i) => s2 + i.menge, 0), 0));
  let totalShipments = $derived(incomingShipments.length);
</script>

<div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4 {showGreenFade ? 'animate-green-fade' : ''}">
  <div class="flex items-center justify-between border-b border-slate-100 pb-3">
    <h2 class="text-sm font-bold text-slate-800">Wareneingang</h2>
    <span class="text-[10px] bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase">Im Zulauf</span>
  </div>
  
  {#if !incomingShipments.length}
    <div class="py-8 text-center text-xs text-slate-400">🚚 Keine offenen Bestellungen im Zulauf.</div>
  {:else}
    <div class="space-y-4 pt-1">
      <div class="flex items-center gap-4">
        <div class="w-12 h-12 rounded-full bg-blue-50 flex items-center justify-center text-blue-600 shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
        </div>
        <div>
          <div class="text-2xl font-black text-slate-800 tracking-tight">{totalItems} <span class="text-base font-bold text-slate-500">Exemplare</span></div>
          <div class="text-xs font-semibold text-slate-400">aus {totalShipments} ausstehenden {totalShipments === 1 ? 'Lieferung' : 'Lieferungen'}</div>
        </div>
      </div>
      
      <button 
        onclick={onOpenWareneingang}
        class="w-full py-2.5 px-4 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-sm rounded-xl transition-colors cursor-pointer flex items-center justify-center gap-2"
      >
        Wareneingang bearbeiten
        <span class="transform transition-transform">→</span>
      </button>
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
</style>
