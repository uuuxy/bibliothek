<script>
  import { apiPost } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  let { 
    incomingShipments = [],
    onBack,
    onReceived
  } = $props();

  let isSubmitting = $state(false);
  /** @type {string[]} */
  let selectedExemplarIds = $state([]);

  /**
   * @param {Event} e
   * @param {string[]} ids
   */
  function toggleItemSelection(e, ids) {
    const target = /** @type {HTMLInputElement} */ (e.target);
    if (target.checked) {
      selectedExemplarIds = [...selectedExemplarIds, ...ids];
    } else {
      selectedExemplarIds = selectedExemplarIds.filter(id => !ids.includes(id));
    }
  }

  function toggleAll() {
    if (allSelected) {
      selectedExemplarIds = [];
    } else {
      selectedExemplarIds = incomingShipments.flatMap(s => s.items.flatMap(i => i.exemplar_ids || []));
    }
  }

  let totalItems = $derived(incomingShipments.reduce((sum, s) => sum + s.items.reduce((s2, i) => s2 + i.menge, 0), 0));
  let allSelected = $derived(selectedExemplarIds.length > 0 && selectedExemplarIds.length === incomingShipments.flatMap(s => s.items.flatMap(i => i.exemplar_ids || [])).length);

  async function handleBulkReceive() {
    if (selectedExemplarIds.length === 0) return;
    
    isSubmitting = true;
    try {
      const data = await apiPost("/api/orders/bulk-receive", { 
        exemplar_ids: selectedExemplarIds
      });
      toastStore.addToast(`${data.received_count} Exemplare erfolgreich eingebucht!`, "success");
      selectedExemplarIds = [];
      onReceived();
    } catch {
      // API handler shows error toast
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="w-full flex-1 flex flex-col gap-6 bg-white border border-slate-200 rounded-2xl shadow-xs p-6 sm:p-8 animate-fade-in">
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
        <h2 class="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">Wareneingang bearbeiten</h2>
        <p class="mt-1 text-sm text-slate-500">Wähle die Positionen aus, die eingetroffen sind und ins System aufgenommen werden sollen.</p>
      </div>
    </div>
    
    <div class="flex items-center justify-end sm:self-end">
      <button 
        onclick={handleBulkReceive} 
        disabled={isSubmitting || selectedExemplarIds.length === 0} 
        class="px-6 py-3 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 rounded-xl shadow-xs shadow-blue-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer flex items-center gap-2"
      >
        {#if isSubmitting}
          <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
        {/if}
        Ausgewählte Positionen einbuchen
      </button>
    </div>
  </div>

  <!-- Content/Table Section -->
  <div class="flex-1 flex flex-col min-h-0">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-xs font-bold text-slate-400 uppercase tracking-wider">Erwartete Positionen ({totalItems} Exemplare)</h3>
      {#if incomingShipments.length > 0}
        <button onclick={toggleAll} class="text-xs font-bold text-blue-600 hover:text-blue-700 cursor-pointer">
          {allSelected ? 'Auswahl aufheben' : 'Alle auswählen'}
        </button>
      {/if}
    </div>
    
    <div class="flex-1 border border-slate-200 rounded-xl overflow-hidden bg-slate-50/30 flex flex-col">
      <div class="overflow-y-auto max-h-[50vh] sm:max-h-[60vh] custom-scrollbar">
        {#if incomingShipments.length === 0}
          <div class="py-12 text-center text-sm font-medium text-slate-400">Keine Positionen im Zulauf.</div>
        {:else}
          {#each incomingShipments as group}
            <div class="bg-slate-50/80 border-b border-slate-200 px-6 py-3 flex items-center justify-between sticky top-0 z-10 backdrop-blur-sm">
              <div class="font-bold text-slate-800">{group.supplierName}</div>
              <div class="text-xs font-semibold text-slate-500">Bestellt am {group.date}</div>
            </div>
            <table class="w-full text-left text-sm border-collapse">
              <tbody class="divide-y divide-slate-100 bg-white">
                {#each group.items as item}
                  {@const isSelected = item.exemplar_ids.every(id => selectedExemplarIds.includes(id))}
                  <tr class="hover:bg-blue-50/30 transition-colors {isSelected ? 'bg-blue-50/50' : ''}">
                    <td class="pl-6 pr-3 py-4 w-12">
                      <input 
                        type="checkbox" 
                        class="w-4 h-4 text-blue-600 border-slate-300 rounded focus:ring-blue-500 cursor-pointer"
                        checked={isSelected}
                        onchange={(e) => toggleItemSelection(e, item.exemplar_ids || [])}
                      />
                    </td>
                    <td class="px-3 py-4 w-20 shrink-0">
                      {#if item.isbn && item.cover_url}
                        <img src="/api/images/cover?isbn={item.isbn}&url={encodeURIComponent(item.cover_url)}" class="w-16 h-24 object-cover shadow-sm rounded border border-slate-200" alt="Cover" loading="lazy" />
                      {:else}
                        <div class="w-16 h-24 bg-slate-100 rounded border border-slate-200 flex items-center justify-center text-slate-400 text-[10px] text-center p-1 leading-tight">Kein Cover</div>
                      {/if}
                    </td>
                    <td class="px-3 py-4 text-slate-800 font-semibold text-base">{item.titel}</td>
                    <td class="px-6 py-4 text-right">
                      <span class="inline-flex items-center justify-center min-w-14 h-14 px-2 rounded-xl bg-blue-50 text-blue-800 text-3xl font-extrabold shadow-inner border border-blue-200">
                        {item.menge}
                      </span>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/each}
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .custom-scrollbar::-webkit-scrollbar {
    width: 6px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background-color: #cbd5e1;
    border-radius: 6px;
  }
</style>
