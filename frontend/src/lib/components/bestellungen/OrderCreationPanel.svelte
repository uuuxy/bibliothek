<script>
  import { apiPost } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  let {
    suppliers,
    orderCart,
    orderTotal,
    submittingOrder,
    selectedSupplierIdx = $bindable(),
    searchQuery = $bindable(),
    searchResults = $bindable(),
    showDropdown = $bindable(),
    isbnPreview = $bindable(),
    isbnLoading = $bindable(),
    onSearchInput,
    onAddToCart,
    onRemoveFromCart,
    onSubmitOrder,
    globalGenerateBarcodes = $bindable(true)
  } = $props();

  /** @type {any} */
  let stagedBook = $state(null);
  let stagedMenge = $state(1);
  let stagedGenerateBarcodes = $state(true);
  let resolvingDnb = $state(false);

  let localResults = $derived(searchResults.filter(r => r.source === 'local'));
  let dnbResults = $derived(searchResults.filter(r => r.source === 'dnb'));

  async function openStaging(book) {
    if (book.source === 'dnb') {
      resolvingDnb = true;
      try {
        const localBook = await apiPost("/api/buecher/aus-isbn", { isbn: book.isbn });
        if (localBook && localBook.titel_id) {
          stagedBook = {
            id: localBook.titel_id,
            titel: localBook.titel,
            autor: localBook.autor,
            isbn: localBook.isbn,
            verlag: localBook.verlag,
            cover_url: localBook.cover_url
          };
          stagedMenge = 1;
          stagedGenerateBarcodes = true;
          showDropdown = false;
          searchQuery = "";
        } else {
          toastStore.addToast("Fehler beim Anlegen des DNB-Buchs", "error");
        }
      } catch {
        toastStore.addToast("Fehler beim Anlegen des DNB-Buchs", "error");
      } finally {
        resolvingDnb = false;
      }
    } else {
      stagedBook = book;
      stagedMenge = 1;
      stagedGenerateBarcodes = true;
      showDropdown = false;
      searchQuery = "";
    }
  }

  function confirmAddToCart() {
    onAddToCart(stagedBook, stagedMenge, stagedGenerateBarcodes);
    stagedBook = null;
  }

  function cancelAddToCart() {
    stagedBook = null;
  }
</script>

<div class="lg:col-span-8 space-y-6">
  <div class="border-b border-gray-200 pb-3 flex items-center justify-between">
    <h2 class="text-lg font-bold text-slate-800">Neue Buchbestellung erstellen</h2>
    <span class="text-[10px] bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-bold uppercase tracking-wider">Entwurf</span>
  </div>
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div class="space-y-1.5"><label for="supplier" class="block text-sm font-medium text-gray-600">Lieferant</label><select id="supplier" bind:value={selectedSupplierIdx} class="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-base bg-white">{#each suppliers as s, idx}<option value={idx}>{s.name} ({s.customerNumber})</option>{/each}</select></div>
    <div class="space-y-1.5 relative">
      <label for="book" class="block text-sm font-medium text-gray-600">Buchtitel hinzufügen</label><input id="book" type="text" bind:value={searchQuery} oninput={onSearchInput} placeholder="Titel, Autor oder ISBN suchen..." class="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-base bg-white" />
      {#if showDropdown && (localResults.length > 0 || dnbResults.length > 0)}
        <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-72 overflow-y-auto divide-y divide-slate-100">
          {#if localResults.length > 0}
            <div class="bg-slate-50/80 px-3.5 py-2 text-[10px] font-bold text-slate-500 uppercase tracking-wider sticky top-0 backdrop-blur-xs z-5">
              Im lokalen Bestand
            </div>
            {#each localResults as b}
              <button onclick={() => openStaging(b)} class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 border-b border-slate-100 last:border-0 flex items-center gap-3 text-base">
                {#if b.cover_url}<img src="/api/images/cover?isbn={b.isbn || ''}&url={encodeURIComponent(b.cover_url)}" class="w-7 aspect-3/4 object-cover rounded-sm" alt="" />{:else}<div class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase">{b.titel.charAt(0)}</div>{/if}
                <div class="min-w-0 flex-1">
                  <div class="font-bold text-slate-800 truncate">{b.titel}</div>
                  <div class="text-sm text-slate-400 truncate">{b.autor} · {b.isbn}</div>
                </div>
                <span class="shrink-0 text-xs bg-emerald-50 text-emerald-700 px-2 py-0.5 rounded-full font-bold">
                  Bestand: {b.current_stock || 0}
                </span>
              </button>
            {/each}
          {/if}

          {#if dnbResults.length > 0}
            <div class="bg-slate-50/80 px-3.5 py-2 text-[10px] font-bold text-slate-500 uppercase tracking-wider sticky top-0 backdrop-blur-xs z-5">
              Neu aus DNB (Externe Suche)
            </div>
            {#each dnbResults as b}
              {@const isDuplicate = b.is_duplicate || localResults.some(l => (l.isbn || '').replace(/-/g, '') === (b.isbn || '').replace(/-/g, ''))}
              <button 
                onclick={() => !isDuplicate && openStaging(b)} 
                disabled={isDuplicate}
                class="w-full text-left px-3.5 py-2.5 flex items-center gap-3 text-base border-b border-slate-100 last:border-0 {isDuplicate ? 'opacity-50 cursor-not-allowed bg-slate-50/30' : 'hover:bg-slate-50'}"
              >
                {#if b.cover_url}<img src="/api/images/cover?isbn={b.isbn || ''}&url={encodeURIComponent(b.cover_url)}" class="w-7 aspect-3/4 object-cover rounded-sm" alt="" />{:else}<div class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase">{b.titel.charAt(0)}</div>{/if}
                <div class="min-w-0 flex-1">
                  <div class="font-bold text-slate-800 truncate">{b.titel}</div>
                  <div class="text-sm text-slate-400 truncate">{b.autor} · {b.isbn}</div>
                </div>
                {#if isDuplicate}
                  <span class="shrink-0 text-[10px] bg-slate-100 text-slate-500 px-2 py-0.5 rounded font-bold uppercase">
                    Vorhanden
                  </span>
                {:else}
                  <span class="shrink-0 text-[10px] bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase">
                    NEU
                  </span>
                {/if}
              </button>
            {/each}
          {/if}
        </div>
      {/if}
      {#if isbnLoading}
        <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg px-4 py-3 flex items-center gap-2 text-sm text-slate-500">
          <div class="w-4 h-4 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin shrink-0"></div>
          Suche läuft...
        </div>
      {:else if resolvingDnb}
        <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg px-4 py-3 flex items-center gap-2 text-sm text-slate-500">
          <div class="w-4 h-4 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin shrink-0"></div>
          Titel wird im Katalog angelegt...
        </div>
      {/if}
    </div>
  </div>

  {#if stagedBook}
    <div class="p-4 border-l-2 border-blue-400 bg-blue-50/50 flex flex-col md:flex-row items-center justify-between gap-4 animate-fade-in">
      <div class="flex items-center gap-3 min-w-0">
        {#if stagedBook.cover_url}
          <img src="/api/images/cover?isbn={stagedBook.isbn || ''}&url={encodeURIComponent(stagedBook.cover_url)}" class="w-10 aspect-3/4 object-cover rounded shadow-sm border border-slate-100 shrink-0" alt="" />
        {:else}
          <div class="w-10 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase shrink-0">{stagedBook.titel.charAt(0)}</div>
        {/if}
        <div class="min-w-0">
          <div class="font-bold text-slate-800 truncate">{stagedBook.titel}</div>
          <div class="text-xs text-slate-500 truncate">{stagedBook.autor}</div>
        </div>
      </div>
      
      <div class="flex flex-wrap items-center gap-4 shrink-0">
        <div class="flex items-center gap-2">
          <label for="stagedMengeInput" class="text-xs font-bold text-slate-500 uppercase">Menge:</label>
          <input id="stagedMengeInput" type="number" min="1" bind:value={stagedMenge} class="w-16 px-2 py-1.5 border border-slate-200 bg-white rounded-md text-center font-bold text-slate-700 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
        </div>
        
        <label class="flex items-center gap-2 cursor-pointer bg-white px-3 py-1.5 border border-slate-200 rounded-md">
          <input type="checkbox" bind:checked={stagedGenerateBarcodes} class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500" />
          <span class="text-sm font-semibold text-slate-700">Barcodes generieren</span>
        </label>

        <div class="flex items-center gap-2 ml-auto">
          <button onclick={cancelAddToCart} class="px-3 py-1.5 text-sm font-bold text-slate-500 hover:text-slate-700 cursor-pointer">Abbrechen</button>
          <button onclick={confirmAddToCart} class="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg text-sm shadow-sm cursor-pointer whitespace-nowrap">In den Warenkorb</button>
        </div>
      </div>
    </div>
  {/if}

  <div class="space-y-3">
    <span class="text-sm font-medium text-gray-600">Warenkorb</span>
    {#if !orderCart.length}
      <div class="py-10 border border-dashed border-slate-200 rounded-lg text-center text-base text-slate-400">Der Warenkorb ist leer. Suche nach Büchern zum Hinzufügen.</div>
    {:else}
      <div class="border border-slate-100 rounded-lg overflow-hidden divide-y divide-slate-100">
        {#each orderCart as item, idx}
          <div class="p-3 bg-slate-50/30 flex items-center justify-between gap-4 text-base">
            <div class="flex items-center gap-3 min-w-0">
              {#if item.cover_url}<img src="/api/images/cover?isbn={item.isbn || ''}&url={encodeURIComponent(item.cover_url)}" class="w-8 aspect-3/4 object-cover rounded-sm" alt="" />{:else}<div class="w-8 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase">{item.titel.charAt(0)}</div>{/if}
              <div class="min-w-0">
                <h4 class="font-bold text-slate-800 truncate">{item.titel}</h4>
                <p class="text-sm text-slate-400 truncate">ISBN: {item.isbn}</p>
                {#if item.generate_barcodes}
                  <div class="text-[10px] font-bold text-blue-600 mt-1 flex items-center gap-1 bg-blue-50 w-fit px-1.5 py-0.5 rounded-md">
                    🔖 {item.menge} {item.menge === 1 ? 'Barcode' : 'Barcodes'} reserviert
                  </div>
                {/if}
              </div>
            </div>
            <div class="flex items-center gap-4">
              <div class="flex items-center gap-2">
                <span class="text-sm font-semibold text-slate-400">€</span>
                <input type="number" step="0.01" bind:value={item.preis} class="w-20 px-2 py-1 border border-slate-200 rounded-md text-right font-semibold text-slate-700 focus:outline-none focus:border-blue-400 focus:ring-1 focus:ring-blue-400" />
              </div>
              <div class="flex items-center border border-slate-200 bg-white rounded-md overflow-hidden"><button onclick={() => item.menge = Math.max(1, item.menge - 1)} class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">-</button><span class="px-3 font-bold text-slate-700 min-w-[20px] text-center">{item.menge}</span><button onclick={() => item.menge += 1} class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">+</button></div>
              <button onclick={() => onRemoveFromCart(idx)} class="text-slate-400 hover:text-rose-500 cursor-pointer">Löschen</button>
            </div>
          </div>
        {/each}
      </div>
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mt-4">
        <div class="text-lg font-bold text-slate-800">
          Gesamtsumme: {orderTotal.toFixed(2).replace('.', ',')} €
        </div>
        <div class="flex flex-col sm:flex-row items-end sm:items-center gap-4">
          <label class="flex items-center gap-2 cursor-pointer bg-white px-3 py-2 border border-slate-200 rounded-lg shadow-sm">
            <input type="checkbox" bind:checked={globalGenerateBarcodes} class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500" />
            <span class="text-sm font-bold text-slate-700">Barcodes mitschicken</span>
          </label>
          <button onclick={onSubmitOrder} disabled={submittingOrder} class="px-5 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-bold text-base shadow-sm cursor-pointer disabled:bg-slate-200 disabled:text-slate-400 flex items-center gap-2">
            {#if submittingOrder}
              <div class="w-4 h-4 border-2 border-t-white border-white/20 rounded-full animate-spin"></div>
              Bestellung wird gesendet...
            {:else}
              📤 Bestellung auslösen ({orderCart.reduce((a, c) => a + c.menge, 0)} Expl.)
            {/if}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
