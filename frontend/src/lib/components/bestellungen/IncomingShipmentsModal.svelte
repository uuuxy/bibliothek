<script>
  import { slide, fade } from "svelte/transition";
  import { apiPost } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  let { 
    shipment,
    onClose,
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

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm" transition:fade={{ duration: 150 }} onclick={onClose}>
  <div class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl flex flex-col max-h-[85vh] overflow-hidden" onclick={(e) => e.stopPropagation()} transition:slide={{ duration: 200, axis: 'y' }}>
    
    <div class="px-6 py-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
      <div>
        <h2 class="text-lg font-bold text-slate-800">Wareneingang einbuchen</h2>
        <p class="text-sm text-slate-500 mt-1">Lieferung vom {shipment.date} • {shipment.supplierName}</p>
      </div>
      <button aria-label="Schließen" onclick={onClose} class="p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-100 rounded-lg transition-colors cursor-pointer">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <div class="p-6 overflow-y-auto flex-1">
      <h3 class="text-sm font-bold text-slate-700 mb-4 uppercase tracking-wider">Erwartete Positionen</h3>
      
      <div class="border border-slate-200 rounded-xl overflow-hidden">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 border-b border-slate-200 text-slate-500">
            <tr>
              <th class="px-4 py-3 font-semibold w-full">Titel</th>
              <th class="px-4 py-3 font-semibold text-right whitespace-nowrap">Menge</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            {#each shipment.items as item}
              <tr class="hover:bg-slate-50/50">
                <td class="px-4 py-3 text-slate-800 font-medium">{item.titel}</td>
                <td class="px-4 py-3 text-right">
                  <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-emerald-100 text-emerald-700">
                    {item.menge}x
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="p-6 border-t border-slate-100 bg-slate-50 flex justify-end gap-3 shrink-0">
      <button onclick={onClose} class="px-5 py-2.5 text-sm font-bold text-slate-600 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 transition-colors cursor-pointer">
        Abbrechen
      </button>
      <button onclick={handleBulkReceive} disabled={isSubmitting} class="px-6 py-2.5 text-sm font-bold text-white bg-blue-600 rounded-xl hover:bg-blue-700 shadow-sm shadow-blue-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer flex items-center gap-2">
        {#if isSubmitting}
          <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
        {/if}
        Kompletten Wareneingang einbuchen
      </button>
    </div>

  </div>
</div>
