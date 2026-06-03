<script>
  let { showVormerkenModal = $bindable(), vormerkenQuery = $bindable(), isSearchingVormerken, vormerkenResults, isSubmittingVormerken, handleVormerkenSearch, handleVormerkenSubmit } = $props();
</script>

{#if showVormerkenModal}
  <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onclick={() => showVormerkenModal = false}></div>
    <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-xl w-full relative z-10 border border-slate-200 flex flex-col max-h-[80vh]">
      <h3 class="text-xl font-bold text-slate-800 mb-4">Titel vormerken</h3>
      <p class="text-sm text-slate-500 mb-4">Suche nach ISBN oder Titel, um das Medium auf die Warteliste zu setzen.</p>
      
      <form onsubmit={(e) => { e.preventDefault(); handleVormerkenSearch(); }} class="flex gap-2 mb-6">
        <input type="text" bind:value={vormerkenQuery} placeholder="Titel oder ISBN eingeben..." class="flex-1 bg-slate-50 border border-slate-200 rounded-xl px-4 py-2 outline-none focus:border-amber-500 focus:ring-2 focus:ring-amber-500/20 transition-all" />
        <button type="submit" disabled={isSearchingVormerken || !vormerkenQuery} class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-xl font-semibold transition-colors disabled:opacity-50 cursor-pointer">Suchen</button>
      </form>

      <div class="flex-1 overflow-y-auto space-y-2 min-h-0">
        {#if isSearchingVormerken}
          <p class="text-center text-slate-500 py-4">Suche läuft...</p>
        {:else if vormerkenResults.length > 0}
          {#each vormerkenResults as res}
            <div class="flex items-center justify-between p-3 bg-slate-50 rounded-xl border border-slate-100">
              <div class="flex-1 min-w-0 pr-4">
                <h4 class="font-bold text-slate-800 truncate">{res.titel}</h4>
                {#if res.isbn}<p class="text-sm text-slate-500">ISBN: {res.isbn}</p>{/if}
              </div>
              <button onclick={() => handleVormerkenSubmit(res.id)} disabled={isSubmittingVormerken} class="shrink-0 px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white text-sm font-bold rounded-lg transition-colors shadow-sm disabled:opacity-50 cursor-pointer">
                Vormerken
              </button>
            </div>
          {/each}
        {:else if vormerkenQuery && !isSearchingVormerken}
          <p class="text-center text-slate-500 py-4">Keine Titel gefunden.</p>
        {/if}
      </div>

      <div class="mt-6 pt-4 border-t border-slate-100 text-right">
        <button onclick={() => showVormerkenModal = false} class="px-4 py-2 text-slate-600 hover:bg-slate-100 font-semibold rounded-xl transition-colors cursor-pointer">Schließen</button>
      </div>
    </div>
  </div>
{/if}
