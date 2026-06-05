<script>
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
    onSubmitOrder
  } = $props();
</script>

<div class="lg:col-span-8 bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-5">
  <div class="border-b border-slate-100 pb-3 flex items-center justify-between">
    <h2 class="text-sm font-bold text-slate-800">Neue Buchbestellung erstellen</h2>
    <span class="text-[10px] bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-bold uppercase tracking-wider">Entwurf</span>
  </div>
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div class="space-y-1"><label for="supplier" class="text-sm font-semibold text-slate-400 uppercase tracking-wide">Lieferant</label><select id="supplier" bind:value={selectedSupplierIdx} class="w-full px-3 py-2 rounded-lg border border-slate-200 text-base bg-slate-50/50">{#each suppliers as s, idx}<option value={idx}>{s.name} ({s.customerNumber})</option>{/each}</select></div>
    <div class="space-y-1 relative">
      <label for="book" class="text-sm font-semibold text-slate-400 uppercase tracking-wide">Buchtitel hinzufügen</label><input id="book" type="text" bind:value={searchQuery} oninput={onSearchInput} placeholder="Titel, Autor oder ISBN suchen..." class="w-full px-3 py-2 rounded-lg border border-slate-200 text-base bg-slate-50/50" />
      {#if showDropdown && searchResults.length > 0}
        <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-56 overflow-y-auto">
          {#each searchResults as b}
            <button onclick={() => onAddToCart(b)} class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 border-b border-slate-100 last:border-0 flex items-center gap-3 text-base">
              {#if b.cover_url}<img src={b.cover_url} class="w-7 aspect-3/4 object-cover rounded-sm" alt="" />{:else}<div class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase">{b.titel.charAt(0)}</div>{/if}
              <div class="min-w-0"><div class="font-bold text-slate-800 truncate">{b.titel}</div><div class="text-sm text-slate-400 truncate">{b.autor} · {b.isbn}</div></div>
            </button>
          {/each}
        </div>
      {/if}
      {#if isbnLoading}
        <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg px-4 py-3 flex items-center gap-2 text-sm text-slate-500">
          <div class="w-4 h-4 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin shrink-0"></div>
          ISBN wird bei DNB abgerufen...
        </div>
      {:else if isbnPreview && !isbnPreview.error}
        <div class="absolute z-10 w-full mt-1 bg-white border border-blue-200 rounded-lg shadow-lg p-3 flex items-center gap-3">
          {#if isbnPreview.cover_url}
            <img src={isbnPreview.cover_url} class="w-10 aspect-3/4 object-cover rounded shadow-sm border border-slate-100 shrink-0" alt="" />
          {:else}
            <div class="w-10 aspect-3/4 rounded bg-slate-100 flex items-center justify-center text-slate-400 text-xs shrink-0">📖</div>
          {/if}
          <div class="min-w-0 flex-1">
            <div class="font-bold text-slate-800 truncate text-sm">{isbnPreview.titel}</div>
            <div class="text-xs text-slate-500 truncate">{isbnPreview.autor} · ISBN {isbnPreview.isbn}</div>
            {#if !isbnPreview.exists}<span class="text-[10px] bg-amber-50 text-amber-700 px-1.5 py-0.5 rounded font-bold">Neu im Katalog</span>{/if}
          </div>
          <button onclick={() => onAddToCart(isbnPreview)} class="shrink-0 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg text-xs cursor-pointer">+ Hinzufügen</button>
        </div>
      {:else if isbnPreview && isbnPreview.error}
        <div class="absolute z-10 w-full mt-1 bg-white border border-rose-200 rounded-lg shadow-lg px-4 py-3 text-sm text-rose-600 font-semibold">
          ISBN nicht gefunden (DNB, Google Books, OpenLibrary)
        </div>
      {/if}
    </div>
  </div>

  <div class="space-y-3">
    <span class="text-sm font-semibold text-slate-400 uppercase tracking-wide">Warenkorb</span>
    {#if !orderCart.length}
      <div class="py-10 border border-dashed border-slate-200 rounded-lg text-center text-base text-slate-400">Der Warenkorb ist leer. Suche nach Büchern zum Hinzufügen.</div>
    {:else}
      <div class="border border-slate-100 rounded-lg overflow-hidden divide-y divide-slate-100">
        {#each orderCart as item, idx}
          <div class="p-3 bg-slate-50/30 flex items-center justify-between gap-4 text-base">
            <div class="flex items-center gap-3 min-w-0">
              {#if item.cover_url}<img src={item.cover_url} class="w-8 aspect-3/4 object-cover rounded-sm" alt="" />{:else}<div class="w-8 aspect-3/4 rounded bg-slate-200 flex items-center justify-center font-bold text-sm uppercase">{item.titel.charAt(0)}</div>{/if}
              <div class="min-w-0"><h4 class="font-bold text-slate-800 truncate">{item.titel}</h4><p class="text-sm text-slate-400 truncate">ISBN: {item.isbn}</p></div>
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
      <div class="flex items-center justify-between mt-4">
        <div class="text-lg font-bold text-slate-800">
          Gesamtsumme: {orderTotal.toFixed(2).replace('.', ',')} €
        </div>
        <button onclick={onSubmitOrder} disabled={submittingOrder} class="px-5 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-bold text-base shadow-sm cursor-pointer disabled:bg-slate-200 disabled:text-slate-400 flex items-center gap-2">
          {#if submittingOrder}
            <div class="w-4 h-4 border-2 border-t-white border-white/20 rounded-full animate-spin"></div>
            Bestellung wird gesendet...
          {:else}
            📤 Bestellung auslösen ({orderCart.reduce((a, c) => a + c.menge, 0)} Expl.)
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>
