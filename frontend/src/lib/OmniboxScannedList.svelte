<script>
  let { scannedBooks, activeTeacher, undoReturn, markDefekt } = $props();
</script>

{#if scannedBooks.length > 0}
  <div class="w-full max-w-xl rounded-2xl border border-slate-100 bg-white overflow-hidden animate-slide-up shadow-sm">
    <div class="px-5 py-3 border-b border-slate-100 text-xs text-slate-400 uppercase tracking-wider font-mono">Scans in dieser Sitzung</div>
    <div class="divide-y divide-slate-100 max-h-60 overflow-y-auto">
      {#each scannedBooks as entry, idx}
        <div class="p-4 flex items-center justify-between hover:bg-slate-50 transition-colors duration-200 {entry.defekt ? 'bg-rose-50/60' : ''}">
          <div class="flex items-center space-x-4">
            {#if entry.book.cover_url}
              <img src={entry.book.cover_url} class="w-12 h-16 object-cover rounded-md shadow-sm border border-slate-100" alt="Cover" />
            {:else}
              <div class="w-12 h-16 rounded-md shadow-sm flex-none flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-600 text-sm border border-indigo-600/10">
                {entry.book.titel ? entry.book.titel.charAt(0).toUpperCase() : '?'}
              </div>
            {/if}
            <div>
              <div class="flex items-center space-x-2 mb-1">
                <span class="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full font-bold border {entry.action === 'ausleihe' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : entry.defekt ? 'bg-rose-100 border-rose-200 text-rose-700' : 'bg-blue-50 border-blue-100 text-blue-700'}">
                  {entry.defekt ? 'Defekt' : entry.action === 'ausleihe' ? 'Ausleihe' : 'Rückgabe'}
                </span>
              </div>
              <h4 class="font-semibold text-sm text-slate-800">{entry.book.titel}</h4>
              <p class="text-xs text-slate-400">{entry.book.autor} · Barcode: {entry.book.barcode_id}</p>
            </div>
          </div>
          <div class="flex items-center space-x-2">
            {#if entry.dueDate}
              <div class="text-right mr-2">
                <span class="text-[10px] text-slate-400">Frist:</span>
                <p class="text-xs font-mono text-emerald-600 font-bold">
                  {activeTeacher ? 'Dauerhaft (Handapparat)' : new Date(entry.dueDate).toLocaleDateString("de-DE")}
                </p>
              </div>
            {/if}
            {#if entry.action === 'rueckgabe' && entry.loanId && !entry.defekt}
              <button onclick={() => undoReturn(entry.loanId, idx)} title="Rückgabe rückgängig machen"
                class="px-2 py-1 text-xs font-semibold rounded-lg bg-amber-50 border border-amber-200 text-amber-700 hover:bg-amber-100 transition-colors cursor-pointer flex items-center gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
                Undo
              </button>
              <button onclick={() => markDefekt(entry, idx)} title="Defekt/Schaden melden und Mahngebühr erheben"
                class="px-2 py-1 text-xs font-semibold rounded-lg bg-rose-50 border border-rose-200 text-rose-700 hover:bg-rose-100 transition-colors cursor-pointer flex items-center gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
                Defekt
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}
