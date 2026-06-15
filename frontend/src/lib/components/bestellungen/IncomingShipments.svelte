<script>
  let { 
    incomingShipments, 
    showGreenFade, 
    isReleasing, 
    scanningTitelId = $bindable(), 
    scannedBarcode = $bindable(), 
    onReceiveItem, 
    onReleaseAll,
    onReleaseNaacher
  } = $props();
</script>

<div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
  <div class="flex items-center justify-between border-b border-slate-100 pb-3"><h2 class="text-sm font-bold text-slate-800">Wareneingang</h2><span class="text-[10px] bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase">Im Zulauf</span></div>
  {#if !incomingShipments.length}
    <div class="py-8 text-center text-xs text-slate-400">🚚 Keine offenen Bestellungen im Zulauf.</div>
  {:else}
    <div class="space-y-4">
      <div class="max-h-60 overflow-y-auto space-y-2 {showGreenFade ? 'animate-green-fade' : ''}">
        {#each incomingShipments as s}
          <div class="p-3 border border-slate-100 rounded-lg bg-slate-50/50 text-[11px] space-y-2">
            <div class="flex justify-between font-bold text-slate-700"><span>{s.supplierName}</span><span class="text-slate-400 font-medium">{s.date}</span></div>
            {#each s.items as item}
              <div class="flex flex-col gap-2 p-2 bg-white rounded border border-slate-100 shadow-sm">
                <div class="flex justify-between items-center text-slate-600">
                  <span class="truncate font-medium">{item.titel}</span>
                  <div class="flex items-center gap-3">
                    <span class="font-bold text-emerald-600 bg-emerald-50 px-1.5 py-0.5 rounded">{item.menge}x</span>
                    {#if scanningTitelId !== item.titel_id}
                      <button onclick={() => { scanningTitelId = item.titel_id; scannedBarcode = ""; }} class="px-2 py-1 bg-blue-100 hover:bg-blue-200 text-blue-700 font-bold rounded text-xs cursor-pointer">Scannen</button>
                    {/if}
                  </div>
                </div>
                {#if scanningTitelId === item.titel_id}
                  <form onsubmit={(e) => { e.preventDefault(); onReceiveItem(item.titel_id); }} class="flex gap-2 mt-1">
                    <!-- svelte-ignore a11y_autofocus -->
                    <input type="text" bind:value={scannedBarcode} placeholder="Barcode scannen..." autofocus class="flex-1 px-2 py-1 text-xs border border-slate-300 rounded focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
                    <button type="submit" disabled={!scannedBarcode} class="px-2 py-1 bg-emerald-600 hover:bg-emerald-700 disabled:bg-slate-300 text-white font-bold rounded text-xs cursor-pointer">OK</button>
                    <button type="button" onclick={() => scanningTitelId = null} class="px-2 py-1 bg-slate-200 hover:bg-slate-300 text-slate-600 font-bold rounded text-xs cursor-pointer">Abbrechen</button>
                  </form>
                {/if}
              </div>
            {/each}
          </div>
        {/each}
      </div>
      <div class="flex flex-col sm:flex-row gap-2">
        <button onclick={onReleaseNaacher} disabled={isReleasing} class="flex-1 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg text-xs shadow-sm cursor-pointer disabled:bg-blue-300 transition-colors">🚀 Lieferung freigeben (Naacher)</button>
        <button onclick={onReleaseAll} disabled={isReleasing} class="flex-1 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold rounded-lg text-xs shadow-sm cursor-pointer disabled:bg-slate-50 transition-colors">📦 Blind freigeben</button>
      </div>
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
