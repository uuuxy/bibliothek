<script>
  /** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void, mode?: "loans" | "scans" }} */
  let { books = [], onReturnClick = undefined, onDamageClick = undefined, mode = "loans" } = $props();
</script>

<div class="space-y-2 max-h-64 overflow-y-auto pr-2 custom-scrollbar">
  {#each books as book (book.id || book.barcode_id || Math.random())}
    {@const isLMF = book.titel?.toLowerCase().startsWith("lmf-")}
    <div class="p-1.5 rounded-xl border border-slate-100 bg-slate-50/50 hover:bg-slate-50 transition-all duration-200 flex flex-row items-center justify-between gap-3">
      <div class="flex items-center space-x-3 flex-1 min-w-0">
        {#if book.cover_url}
          <img src={book.cover_url} class="w-6 h-9 object-cover rounded shadow-sm border border-slate-100/50 shrink-0" alt="Cover" />
        {:else}
          <div class="w-6 h-9 rounded shadow-sm shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-600 text-[10px] border border-indigo-600/10">
            {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
          </div>
        {/if}
        <div class="flex-1 min-w-0 text-left flex flex-col justify-center leading-tight">
          <div class="flex items-center gap-2">
            <h4 class="font-bold text-sm text-slate-900 truncate font-sans">{book.titel}</h4>
            {#if isLMF}
              <span class="shrink-0 px-1 py-0.5 rounded text-[9px] font-bold bg-indigo-50 text-indigo-700 border border-indigo-100 uppercase tracking-wide">
                LMF
              </span>
            {/if}
          </div>
          <div class="flex items-center gap-1.5 text-[11px] text-slate-500 mt-0.5 truncate font-sans">
            {#if mode === "loans"}
              <span class="truncate max-w-[100px] font-medium" title={book.autor}>{book.autor}</span>
              <span class="text-slate-300">•</span>
            {/if}
            <span class="font-bold text-slate-700">{book.barcode_id}</span>
            {#if mode === "loans"}
              <span class="text-slate-300 hidden md:inline">•</span>
              <span class="hidden md:inline">{new Date(book.ausgeliehen_am).toLocaleDateString("de-DE")}</span>
            {/if}
          </div>
        </div>
      </div>

      <div class="text-right shrink-0 flex items-center gap-3">
        {#if mode === "loans"}
          <div class="flex flex-col items-end justify-center">
            <span class="text-[9px] text-slate-400 block font-bold uppercase leading-none mb-0.5 font-sans">Frist</span>
            <span class="{isLMF ? 'text-indigo-600' : 'text-slate-700'} font-black text-xs font-sans">
              {new Date(book.rueckgabe_frist).toLocaleDateString("de-DE")}
            </span>
          </div>
          {#if onDamageClick}
            <button onclick={() => onDamageClick(book)} class="p-1.5 bg-rose-100 hover:bg-rose-200 text-rose-700 rounded-lg transition-colors cursor-pointer" title="Verlust/Schaden melden">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
            </button>
          {/if}
          {#if onReturnClick}
            <button onclick={() => onReturnClick(book.barcode_id)} class="p-1.5 bg-emerald-100 hover:bg-emerald-200 text-emerald-700 rounded-lg transition-colors cursor-pointer" title="Buch zurückgeben">
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
            </button>
          {/if}
        {:else if mode === "scans"}
          <svg class="w-5 h-5 text-emerald-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
        {/if}
      </div>
    </div>
  {/each}
</div>
