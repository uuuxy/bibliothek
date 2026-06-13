<script>
  import { apiFetch } from "./apiFetch.js";
  import { authStore } from "./stores/authStore.svelte.js";

  /** @type {any} */
  let summary = $state(null);
  let loading = $state(true);


  async function fetchSummary() {
    try {
      const res = await apiFetch("/api/dashboard/summary");
      if (res.ok) {
        summary = await res.json();
      }
    } catch (err) {
      console.error(err);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchSummary();
  });


</script>

{#if loading}
  <div class="p-6 bg-white border border-rose-200 rounded-2xl shadow-xs flex justify-center h-full items-center">
    <div class="w-6 h-6 border-2 border-t-rose-500 border-rose-500/20 rounded-full animate-spin"></div>
  </div>
{:else if summary}
  <div class="bg-white border-2 border-rose-400 rounded-2xl shadow-xs overflow-hidden flex flex-col h-full">
    <!-- Header -->
    <div class="bg-rose-50 p-4 border-b border-rose-200 flex justify-between items-center">
      <div>
        <h3 class="text-rose-700 font-bold uppercase tracking-wider text-sm">Dringend: Mahnungen</h3>
        <p class="text-rose-600/80 text-xs font-semibold mt-0.5">Überfällige Ausleihen gesamt</p>
      </div>
      <div class="bg-rose-600 text-white font-extrabold text-2xl px-4 py-1.5 rounded-xl shadow-sm">
        {summary.total_overdue}
      </div>
    </div>
    
    <!-- Top 5 List -->
    <div class="p-4 flex-1 pb-6">
      <h4 class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">Am längsten überfällig (Härtefälle)</h4>
      {#if summary.top_overdue && summary.top_overdue.length > 0}
        <ul class="space-y-3">
          {#each summary.top_overdue as item}
            <li class="flex justify-between items-start text-sm">
              <div class="min-w-0 pr-2">
                <span class="block font-bold text-slate-800 truncate">{item.schueler_name} <span class="text-slate-400 font-semibold text-xs ml-1">({item.klasse})</span></span>
                <span class="block text-slate-500 text-xs font-medium truncate">{item.titel}</span>
              </div>
              <div class="shrink-0 text-right">
                <span class="text-rose-600 font-bold bg-rose-50 px-2 py-0.5 rounded text-xs">{item.tage} Tage</span>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-sm text-slate-500 italic py-2">Keine überfälligen Bücher. Alles im Lot!</p>
      {/if}
    </div>
  </div>
{/if}
