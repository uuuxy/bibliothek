<script>
  let { showChecklistModal = $bindable(), pendingGeraet = $bindable(), checklistItems, checkedItems = $bindable(), isSubmittingChecklist, handleChecklistSubmit } = $props();
</script>

{#if showChecklistModal && pendingGeraet}
  <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
    <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm"></div>
    <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-lg w-full relative z-10 border border-slate-200">
      <div class="mb-4 text-center">
        <div class="w-16 h-16 bg-amber-100 text-amber-600 rounded-full flex items-center justify-center mx-auto mb-3">
          <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/></svg>
        </div>
        <h3 class="text-xl font-bold text-slate-800">Hardware-Checkliste</h3>
        <p class="text-slate-500 font-medium">{pendingGeraet.modellname}</p>
      </div>

      <div class="space-y-3 mb-6 bg-slate-50 p-4 rounded-xl border border-slate-100">
        <p class="text-sm font-bold text-slate-700 uppercase tracking-wide mb-2">Bitte auf Vollständigkeit prüfen:</p>
        {#each checklistItems as item}
          <label class="flex items-center space-x-3 cursor-pointer group">
            <input type="checkbox" 
              checked={checkedItems.has(item)}
              onchange={(e) => {
                const target = /** @type {HTMLInputElement} */ (e.target);
                if (target && target.checked) {
                  checkedItems.add(item);
                } else {
                  checkedItems.delete(item);
                }
                checkedItems = new Set(checkedItems);
              }}
              class="w-5 h-5 rounded text-amber-600 focus:ring-amber-500 border-slate-300 cursor-pointer" />
            <span class="text-slate-700 font-medium group-hover:text-amber-700 transition-colors">{item}</span>
          </label>
        {/each}
      </div>

      <div class="flex space-x-3">
        <button onclick={() => { showChecklistModal = false; pendingGeraet = null; }}
          class="flex-1 py-3 px-4 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold rounded-xl transition-colors cursor-pointer">
          Abbrechen
        </button>
        <button onclick={handleChecklistSubmit} disabled={isSubmittingChecklist || checklistItems.length !== checkedItems.size}
          class="flex-1 py-3 px-4 bg-amber-500 hover:bg-amber-600 text-white font-bold rounded-xl transition-colors disabled:opacity-50 shadow-sm cursor-pointer">
          {isSubmittingChecklist ? "Speichere..." : "Bestätigen & Buchen"}
        </button>
      </div>
    </div>
  </div>
{/if}
