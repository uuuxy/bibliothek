<script>
  import { appState } from "../inventur/lib/store.svelte.js";

  /**
   * @typedef {Object} Props
   * @property {any[]} borrowers
   */
  /** @type {Props} */
  let { borrowers } = $props();

  let filterKlasse = $state("Alle");
  let filterName = $state("");

  let availableKlassen = $derived(
    ["Alle", ...Array.from(new Set(borrowers.map((/** @type {any} */ b) => b.klasse || 'Unbekannt'))).sort()]
  );

  let filteredBorrowers = $derived(
    borrowers.filter((/** @type {any} */ b) => {
      const matchKlasse = filterKlasse === "Alle" || (b.klasse || 'Unbekannt') === filterKlasse;
      const matchName = filterName === "" || `${b.schueler_name} ${b.schueler_nachname}`.toLowerCase().includes(filterName.toLowerCase());
      return matchKlasse && matchName;
    })
  );
</script>

{#if borrowers && borrowers.length > 0}
  <div class="mt-8 border-t border-slate-100 pt-6">
    <h3 class="text-sm font-bold text-slate-700 mb-4 flex items-center gap-2 shrink-0">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>
      Aktuell entliehen von:
    </h3>
    
    <!-- Filter Controls (Sticky) -->
    <div class="sticky -top-6 pt-4 pb-3 z-10 bg-white flex flex-col gap-3 border-b border-slate-100 mb-3 -mx-2 px-2">
      <div class="flex gap-2">
        <select 
          bind:value={filterKlasse} 
          class="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-indigo-500/50 cursor-pointer min-w-[100px]"
        >
          {#each availableKlassen as k}
            <option value={k}>{k}</option>
          {/each}
        </select>
        <div class="relative flex-1">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
          <input 
            type="text" 
            bind:value={filterName} 
            placeholder="Name filtern..." 
            class="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 placeholder:text-slate-400"
          />
        </div>
      </div>
    </div>

    <!-- Scrollable List -->
    <div class="pb-8">
      {#if filteredBorrowers.length === 0}
        <div class="text-center py-6 text-sm text-slate-400 bg-slate-50 rounded-xl">Keine Ausleihen entsprechen dem Filter.</div>
      {:else}
        <div class="bg-white rounded-2xl border border-slate-100 overflow-hidden shadow-xs">
          <ul class="divide-y divide-slate-50">
            {#each filteredBorrowers as b}
              <li class="p-3 hover:bg-slate-50 transition-colors flex items-center justify-between group">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-8 h-8 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold text-xs shrink-0">
                    {b.schueler_name ? b.schueler_name.charAt(0) : ''}{b.schueler_nachname ? b.schueler_nachname.charAt(0) : ''}
                  </div>
                  <div class="min-w-0">
                    <button onclick={() => appState.triggerStudentScan = b.schueler_barcode} class="text-sm font-semibold text-slate-800 hover:text-indigo-600 text-left transition-colors cursor-pointer block truncate">
                      {b.schueler_name} {b.schueler_nachname} <span class="text-xs font-normal text-slate-500">({b.klasse || 'Unbekannt'})</span>
                    </button>
                    <div class="text-xs text-slate-400 mt-0.5 truncate">Exemplar: <span class="font-mono text-slate-500">{b.exemplar_barcode}</span></div>
                  </div>
                </div>
                <div class="text-right shrink-0 ml-2">
                  <div class="text-[10px] font-medium text-slate-400 uppercase tracking-wider mb-0.5">Rückgabe bis</div>
                  <div class="text-xs font-bold {new Date(b.rueckgabe_frist) < new Date() ? 'text-rose-600' : 'text-slate-700'}">
                    {new Date(b.rueckgabe_frist).toLocaleDateString('de-DE')}
                  </div>
                </div>
              </li>
            {/each}
          </ul>
        </div>
      {/if}
    </div>
  </div>
{/if}
