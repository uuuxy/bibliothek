<script>
  import { apiFetch } from "./apiFetch.js";

  /** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void, mode?: "loans" | "scans" }} */
  let { books = [], onReturnClick = undefined, onDamageClick = undefined, mode = "loans" } = $props();

  let extendingIds = $state(new Set());

  async function handleExtend(book) {
    const id = book.ausleihe_id || book.id;
    if (!id || extendingIds.has(id)) return;
    
    const next = new Set(extendingIds);
    next.add(id);
    extendingIds = next;
    
    try {
      const response = await apiFetch(`/api/ausleihen/${id}/verlaengern`, { method: "POST" });
      if (response.ok) {
        const data = await response.json();
        book.rueckgabe_frist = data.neues_rueckgabe_datum;
      } else {
        alert("Fehler bei der Verlängerung");
      }
    } catch (e) {
      console.error(e);
      alert("Netzwerkfehler");
    } finally {
      const reset = new Set(extendingIds);
      reset.delete(id);
      extendingIds = reset;
    }
  }
</script>

<div class="max-h-64 overflow-y-auto pr-2 custom-scrollbar">
  <table class="w-full text-left border-collapse">
    <thead>
      <tr class="border-b border-slate-200 text-xs font-bold text-slate-600 uppercase tracking-wider">
        <th class="py-3 px-4">Titel & Autor</th>
        <th class="py-3 px-4">Barcode</th>
        {#if mode === "loans"}
          <th class="py-3 px-4">Rückgabedatum</th>
          <th class="py-3 px-4">Status</th>
        {/if}
        <th class="py-3 px-4 text-right">Aktion</th>
      </tr>
    </thead>
    <tbody class="divide-y divide-slate-100">
      {#each books as book (book.id || book.barcode_id || Math.random())}
        {@const isLMF = book.titel?.toLowerCase().startsWith("lmf-")}
        {@const isOverdue = mode === "loans" && new Date(book.rueckgabe_frist) < new Date()}
        <tr class="hover:bg-slate-50 transition-colors">
          <td class="py-3 px-4">
            <div class="flex items-center space-x-3">
              {#if book.cover_url}
                <img src={book.cover_url} class="w-8 h-12 object-cover rounded shadow-sm border border-slate-100" alt="Cover" />
              {:else}
                <div class="w-8 h-12 rounded shadow-sm flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-600 text-xs border border-indigo-600/10">
                  {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
                </div>
              {/if}
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <h4 class="font-bold text-sm text-slate-900 truncate">{book.titel}</h4>
                  {#if isLMF}
                    <span class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-indigo-50 text-indigo-700 border border-indigo-100 uppercase">LMF</span>
                  {/if}
                </div>
                {#if mode === "loans"}
                  <div class="text-xs text-slate-600 truncate mt-0.5">{book.autor}</div>
                {/if}
              </div>
            </div>
          </td>
          <td class="py-3 px-4 text-sm font-semibold text-slate-700">{book.barcode_id}</td>
          {#if mode === "loans"}
            <td class="py-3 px-4 text-sm font-semibold text-slate-700">
              {new Date(book.rueckgabe_frist).toLocaleDateString("de-DE")}
              <div class="text-xs font-normal text-slate-600 mt-0.5">Geliehen: {new Date(book.ausgeliehen_am).toLocaleDateString("de-DE")}</div>
            </td>
            <td class="py-3 px-4">
              {#if isOverdue}
                <span class="px-2 py-1 bg-rose-50 text-rose-600 text-xs font-bold rounded-full border border-rose-100">Überfällig</span>
              {:else}
                <span class="px-2 py-1 bg-emerald-50 text-emerald-600 text-xs font-bold rounded-full border border-emerald-100">In Frist</span>
              {/if}
            </td>
          {/if}
          <td class="py-3 px-4 text-right">
            <div class="flex items-center justify-end gap-2">
              {#if mode === "loans"}
                <button onclick={() => handleExtend(book)} disabled={extendingIds.has(book.ausleihe_id || book.id)} class="p-2 bg-blue-50 hover:bg-blue-100 text-blue-600 disabled:opacity-50 rounded-full transition-colors cursor-pointer" title="Verlängern">
                  {#if extendingIds.has(book.ausleihe_id || book.id)}
                    <svg class="w-4 h-4 animate-spin text-blue-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path></svg>
                  {:else}
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                  {/if}
                </button>
                {#if onDamageClick}
                  <button onclick={() => onDamageClick(book)} class="p-2 bg-rose-100 hover:bg-rose-200 text-rose-700 rounded-full transition-colors cursor-pointer" title="Verlust/Schaden melden">
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                  </button>
                {/if}
                {#if onReturnClick}
                  <button onclick={() => onReturnClick(book.barcode_id)} class="p-2 bg-emerald-100 hover:bg-emerald-200 text-emerald-700 rounded-full transition-colors cursor-pointer" title="Buch zurückgeben">
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
                  </button>
                {/if}
              {:else if mode === "scans"}
                <svg class="w-5 h-5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
