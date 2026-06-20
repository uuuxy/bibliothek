<script>
  import { apiPost } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  let { 
    shipment,
    onBack,
    onReceived
  } = $props();

  let isSubmitting = $state(false);

  async function handleBulkReceive() {
    isSubmitting = true;
    try {
      const data = await apiPost("/api/orders/bulk-receive", { 
        supplier_name: shipment.supplierName,
        date: shipment.date
      });
      toastStore.addToast(`${data.received_count} Exemplare erfolgreich eingebucht!`, "success");
      onReceived();
    } catch {
      // API handler shows error toast
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="w-full flex-1 flex flex-col gap-6 bg-white border border-slate-200 rounded-2xl shadow-xs p-6 sm:p-8">
  <!-- Back Button & Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-100">
    <div class="space-y-3">
      <button 
        onclick={onBack} 
        class="inline-flex items-center gap-1 text-xs font-bold text-slate-500 hover:text-slate-800 transition-colors cursor-pointer group"
      >
        <span class="transform group-hover:-translate-x-0.5 transition-transform">←</span> Zurück zur Übersicht
      </button>
      <div>
        <h2 class="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">Lieferung im Detail</h2>
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5 mt-1 text-sm text-slate-500">
          <span class="font-bold text-slate-700 bg-slate-100 px-2.5 py-0.5 rounded-md">{shipment.supplierName}</span>
          <span>•</span>
          <span>Bestelldatum: <strong class="text-slate-700">{shipment.date}</strong></span>
          <span>•</span>
          <span class="inline-flex items-center px-2 py-0.5 rounded-md bg-amber-50 text-amber-800 font-semibold text-xs border border-amber-200">im Zulauf</span>
        </div>
      </div>
    </div>
    
    <div class="flex items-center justify-end sm:self-end">
      <button 
        onclick={handleBulkReceive} 
        disabled={isSubmitting} 
        class="px-6 py-3 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 rounded-xl shadow-xs shadow-blue-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer flex items-center gap-2"
      >
        {#if isSubmitting}
          <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
        {/if}
        Kompletten Wareneingang einbuchen
      </button>
    </div>
  </div>

  <!-- Content/Table Section -->
  <div class="flex-1 flex flex-col min-h-0">
    <h3 class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3">Erwartete Positionen ({shipment.items.length})</h3>
    <div class="flex-1 border border-slate-200 rounded-xl overflow-hidden bg-slate-50/30 flex flex-col">
      <div class="overflow-y-auto max-h-[50vh] sm:max-h-[60vh]">
        <table class="w-full text-left text-sm border-collapse">
          <thead class="bg-slate-50 border-b border-slate-200 text-slate-500 sticky top-0 z-10">
            <tr>
              <th class="px-6 py-4 font-bold text-xs uppercase tracking-wider w-full">Titel</th>
              <th class="px-6 py-4 font-bold text-xs uppercase tracking-wider text-right whitespace-nowrap">Erwartete Menge</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 bg-white">
            {#each shipment.items as item}
              <tr class="hover:bg-slate-50/50 transition-colors">
                <td class="px-6 py-4 text-slate-800 font-semibold">{item.titel}</td>
                <td class="px-6 py-4 text-right">
                  <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                    {item.menge}x
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>
