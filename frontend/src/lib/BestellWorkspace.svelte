<script>
  import { onMount } from "svelte";
  let activeTab = $state("bestellungen"), newName = $state(""), newEmail = $state(""), newCustNum = $state("");
  /** @type {any[]} */
  let suppliers = $state([]);
  let selectedSupplierIdx = $state(0), searchQuery = $state(""), showDropdown = $state(false);
  /** @type {any[]} */
  let searchResults = $state([]);
  /** @type {any[]} */
  let orderCart = $state([]);
  let submittingOrder = $state(false);
  /** @type {any} */
  let orderMessage = $state(null);
  /** @type {any[]} */
  let recommendations = $state([]);
  /** @type {any[]} */
  let incomingShipments = $state([]);
  let isReleasing = $state(false), showGreenFade = $state(false);
  /** @type {any | null} */
  let isbnPreview = $state(null);
  let isbnLoading = $state(false);

  onMount(async () => {
    await loadSuppliers();
    await loadIncomingShipments();
    fetchRecommendations();
  });

  async function loadSuppliers() {
    try {
      const res = await fetch("/api/lieferanten");
      if (res.ok) suppliers = (await res.json()) || [];
    } catch (err) { console.error("Fehler beim Laden der Lieferanten:", err); }
  }

  async function loadIncomingShipments() {
    try {
      const res = await fetch("/api/bestellungen/zulauf");
      if (res.ok) incomingShipments = (await res.json()) || [];
    } catch (err) { console.error("Fehler beim Laden des Wareneingangs:", err); }
  }

  async function fetchRecommendations() {
    try {
      const res = await fetch("/api/bestellungen");
      if (res.ok) recommendations = (await res.json()) || [];
    } catch (err) { console.error(err); }
  }

  /** @param {SubmitEvent} e */
  async function addSupplier(e) {
    e.preventDefault(); if (!newName || !newEmail || !newCustNum) return;
    try {
      const res = await fetch("/api/lieferanten", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName, email: newEmail, customerNumber: newCustNum })
      });
      if (res.ok) {
        newName = ""; newEmail = ""; newCustNum = "";
        await loadSuppliers();
      } else {
        const txt = await res.text();
        alert("Fehler beim Erstellen des Lieferanten: " + txt);
      }
    } catch (err) { console.error(err); }
  }

  /** @param {string} id */
  async function removeSupplier(id) {
    try {
      const res = await fetch(`/api/lieferanten/${id}`, {
        method: "DELETE"
      });
      if (res.ok) {
        await loadSuppliers();
        selectedSupplierIdx = Math.max(0, Math.min(selectedSupplierIdx, suppliers.length - 1));
      } else {
        const txt = await res.text();
        alert("Fehler beim Löschen des Lieferanten: " + txt);
      }
    } catch (err) { console.error(err); }
  }

  /** @type {any} */
  let searchTimeout;
  function handleSearchInput() {
    clearTimeout(searchTimeout);
    const raw = searchQuery.trim();
    if (raw.length < 2) { searchResults = []; showDropdown = false; isbnPreview = null; return; }

    const cleanQuery = raw.replace(/[\s-]/g, "");
    const isIsbn = /^\d{10,13}$/.test(cleanQuery);

    if (isIsbn) {
      // ISBN detected: fetch live DNB metadata and upsert catalog entry
      searchResults = []; showDropdown = false; isbnPreview = null; isbnLoading = true;
      (async () => {
        try {
          const res = await fetch("/api/buecher/aus-isbn", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ isbn: cleanQuery })
          });
          if (res.ok) {
            isbnPreview = await res.json();
          } else {
            isbnPreview = { error: true };
          }
        } catch { isbnPreview = { error: true }; }
        finally { isbnLoading = false; }
      })();
    } else {
      isbnPreview = null; isbnLoading = false;
      const performSearch = async () => {
        try {
          const res = await fetch("/api/action", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ query: searchQuery })
          });
          if (res.ok) {
            const data = await res.json();
            if (data.type === "search_results") {
              searchResults = data.search_results || [];
              showDropdown = searchResults.length > 0;
            }
          }
        } catch (err) {
          console.error("Fehler bei der Buchsuche:", err);
        }
      };
      searchTimeout = setTimeout(performSearch, 300);
    }
  }

  /** @param {any} book */
  function addToCart(book) {
    const existing = orderCart.find(item => item.id === book.id || (book.isbn && item.isbn === book.isbn));
    if (existing) { existing.menge += 1; }
    else { orderCart.push({ id: book.titel_id ?? book.id, titel: book.titel, autor: book.autor, isbn: book.isbn ?? book.ISBN, verlag: book.verlag ?? "", cover_url: book.cover_url ?? "", menge: 1 }); }
    searchQuery = ""; searchResults = []; showDropdown = false; isbnPreview = null;
  }

  /** @param {number} idx */
  function removeFromCart(idx) { orderCart.splice(idx, 1); }

  async function submitOrder() {
    if (!orderCart.length || !suppliers.length) return;
    submittingOrder = true; orderMessage = null;
    const supplier = suppliers[selectedSupplierIdx];
    try {
      const res = await fetch("/api/orders", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          supplier_id: supplier.id,
          items: orderCart.map(item => ({ titel_id: item.id, menge: item.menge }))
        })
      });
      const data = await res.json();
      if (res.ok) {
        orderCart = [];
        orderMessage = { type: "success", text: `Bestellung erfolgreich per E-Mail an ${supplier.name} gesendet. ${data.ordered_qty} Barcodes wurden im System reserviert.` };
        await loadIncomingShipments();
        fetchRecommendations();
      } else {
        throw new Error(data.message || "Fehler beim Bestellen");
      }
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      orderMessage = { type: "error", text: "Fehler: " + errMsg };
    } finally { submittingOrder = false; }
  }

  async function releaseIncoming() {
    if (!incomingShipments.length) return;
    isReleasing = true;
    try {
      const res = await fetch("/api/bestellungen/freigeben", { method: "POST" });
      if (res.ok) {
        showGreenFade = true;
        setTimeout(async () => {
          await loadIncomingShipments();
          showGreenFade = false;
          fetchRecommendations();
        }, 1500);
      } else {
        const txt = await res.text();
        alert("Fehler beim Freigeben: " + txt);
      }
    } catch (err) {
      console.error(err);
      showGreenFade = true;
      setTimeout(async () => {
        await loadIncomingShipments();
        showGreenFade = false;
        fetchRecommendations();
      }, 1500);
    } finally { isReleasing = false; }
  }
</script>

<div class="w-full h-full p-8 bg-slate-50/50 text-slate-800 font-sans flex flex-col gap-6">
  <!-- Header & Tabs -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-200 pb-5 shrink-0">
    <div class="flex items-center gap-3">
      <div class="flex bg-slate-100 p-0.5 rounded-lg border border-slate-200">
        <button onclick={() => activeTab = "bestellungen"} class="px-4 py-1.5 text-sm font-bold rounded-md cursor-pointer transition-all {activeTab === 'bestellungen' ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-800'}">Bestellungen</button>
        <button onclick={() => activeTab = "lieferanten"} class="px-4 py-1.5 text-sm font-bold rounded-md cursor-pointer transition-all {activeTab === 'lieferanten' ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-800'}">Lieferanten verwalten</button>
      </div>
      <a href="/api/bestellungen/pdf" download class="px-4 py-2 bg-white hover:bg-slate-50 text-slate-700 font-bold border border-slate-200 rounded-lg text-xs transition-all flex items-center gap-1.5 shadow-2xs">🖨️ PDF-Bestellliste</a>
    </div>
  </div>

  {#if orderMessage}
    <div class="p-3 rounded-lg border text-xs font-semibold flex justify-between items-center {orderMessage.type === 'success' ? 'bg-emerald-50 border-emerald-100 text-emerald-800' : 'bg-rose-50 border-rose-100 text-rose-800'}">
      <span>{orderMessage.type === 'success' ? '✅' : '❌'} {orderMessage.text}</span><button onclick={() => orderMessage = null} class="text-slate-400 hover:text-slate-600 text-sm">✕</button>
    </div>
  {/if}

  {#if activeTab === "bestellungen"}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start overflow-y-auto">
      <!-- Order Creation Panel -->
      <div class="lg:col-span-8 bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-5">
        <div class="border-b border-slate-100 pb-3 flex items-center justify-between">
          <h2 class="text-sm font-bold text-slate-800">Neue Buchbestellung erstellen</h2>
          <span class="text-[10px] bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-bold uppercase tracking-wider">Entwurf</span>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="space-y-1"><label for="supplier" class="text-sm font-semibold text-slate-400 uppercase tracking-wide">Lieferant</label><select id="supplier" bind:value={selectedSupplierIdx} class="w-full px-3 py-2 rounded-lg border border-slate-200 text-base bg-slate-50/50">{#each suppliers as s, idx}<option value={idx}>{s.name} ({s.customerNumber})</option>{/each}</select></div>
          <div class="space-y-1 relative">
            <label for="book" class="text-sm font-semibold text-slate-400 uppercase tracking-wide">Buchtitel hinzufügen</label><input id="book" type="text" bind:value={searchQuery} oninput={handleSearchInput} placeholder="Titel, Autor oder ISBN suchen..." class="w-full px-3 py-2 rounded-lg border border-slate-200 text-base bg-slate-50/50" />
            {#if showDropdown && searchResults.length > 0}
              <div class="absolute z-10 w-full mt-1 bg-white border border-slate-200 rounded-lg shadow-lg max-h-56 overflow-y-auto">
                {#each searchResults as b}
                  <button onclick={() => addToCart(b)} class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 border-b border-slate-100 last:border-0 flex items-center gap-3 text-base">
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
                <button onclick={() => addToCart(isbnPreview)} class="shrink-0 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg text-xs cursor-pointer">+ Hinzufügen</button>
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
                    <div class="flex items-center border border-slate-200 bg-white rounded-md overflow-hidden"><button onclick={() => item.menge = Math.max(1, item.menge - 1)} class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">-</button><span class="px-3 font-mono font-bold text-slate-700 min-w-[20px] text-center">{item.menge}</span><button onclick={() => item.menge += 1} class="px-2 py-0.5 hover:bg-slate-50 font-bold text-slate-500">+</button></div>
                    <button onclick={() => removeFromCart(idx)} class="text-slate-400 hover:text-rose-500 cursor-pointer">Löschen</button>
                  </div>
                </div>
              {/each}
            </div>
            <div class="flex justify-end">
              <button onclick={submitOrder} disabled={submittingOrder} class="px-5 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-bold text-base shadow-sm cursor-pointer disabled:bg-slate-200 disabled:text-slate-400 flex items-center gap-2">
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

      <!-- Sidebar Status & Mindestbestellungen -->
      <div class="lg:col-span-4 space-y-6">
        <!-- Wareneingang (Freigabe) -->
        <div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
          <div class="flex items-center justify-between border-b border-slate-100 pb-3"><h2 class="text-sm font-bold text-slate-800">Wareneingang</h2><span class="text-[10px] bg-amber-50 text-amber-700 px-2 py-0.5 rounded font-bold uppercase">Im Zulauf</span></div>
          {#if !incomingShipments.length}
            <div class="py-8 text-center text-xs text-slate-400">🚚 Keine offenen Bestellungen im Zulauf.</div>
          {:else}
            <div class="space-y-4">
              <div class="max-h-60 overflow-y-auto space-y-2 {showGreenFade ? 'animate-green-fade' : ''}">
                {#each incomingShipments as s}
                  <div class="p-3 border border-slate-100 rounded-lg bg-slate-50/50 text-[11px] space-y-1.5">
                    <div class="flex justify-between font-bold text-slate-700"><span>{s.supplierName}</span><span class="text-slate-400 font-mono font-medium">{s.date}</span></div>
                    {#each s.items as item}<div class="flex justify-between text-slate-600"><span class="truncate">{item.titel}</span><span class="font-mono font-bold">{item.menge}x</span></div>{/each}
                  </div>
                {/each}
              </div>
              <button onclick={releaseIncoming} disabled={isReleasing} class="w-full py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold rounded-lg text-xs shadow-sm cursor-pointer disabled:bg-slate-200">📥 Lieferung vollständig freigeben</button>
            </div>
          {/if}
        </div>

        <!-- Mindestbestand recommendations -->
        <div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
          <div class="border-b border-slate-100 pb-3"><h2 class="text-sm font-bold text-slate-800">Bestellbedarf</h2></div>
          {#if !recommendations.length}
            <p class="text-xs text-slate-400 text-center py-4">Bestände ausreichend.</p>
          {:else}
            <div class="max-h-60 overflow-y-auto space-y-2">
              {#each recommendations as r}
                <div class="p-2.5 bg-slate-50 border border-slate-100 rounded-lg flex items-center justify-between gap-3 text-[11px]">
                  <div class="flex items-center gap-2 min-w-0">
                    {#if r.cover_url}
                      <img src={r.cover_url} class="w-7 aspect-3/4 object-cover rounded-sm shrink-0" alt="" />
                    {:else}
                      <div class="w-7 aspect-3/4 rounded bg-slate-200 flex items-center justify-center text-slate-400 shrink-0 text-[9px]">📖</div>
                    {/if}
                    <div class="min-w-0"><h4 class="font-bold text-slate-800 truncate leading-tight">{r.titel}</h4><p class="text-[9px] text-slate-400 mt-0.5">Bestand: {r.verfuegbarer_bestand} / Melde: {r.meldebestand}</p></div>
                  </div>
                  <button onclick={() => addToCart(r)} class="shrink-0 px-2 py-1 bg-blue-50 hover:bg-blue-100 text-blue-700 font-bold rounded-md text-[9px] cursor-pointer">+ Add</button>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  {#if activeTab === "lieferanten"}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-8 items-start overflow-y-auto">
      <div class="bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
        <h2 class="text-sm font-bold text-slate-800 border-b border-slate-100 pb-3">Neuer Lieferant</h2>
        <form onsubmit={addSupplier} class="space-y-4 text-base">
          <div class="space-y-1"><label for="n" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">Name</label><input id="n" type="text" bind:value={newName} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
          <div class="space-y-1"><label for="e" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">E-Mail</label><input id="e" type="email" bind:value={newEmail} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
          <div class="space-y-1"><label for="c" class="font-semibold text-slate-400 uppercase tracking-wide text-sm">Kundennummer</label><input id="c" type="text" bind:value={newCustNum} required class="w-full px-3 py-2 rounded-lg border border-slate-200 bg-slate-50/50 text-base" /></div>
          <button type="submit" class="w-full py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg cursor-pointer text-base">💾 Lieferanten speichern</button>
        </form>
      </div>

      <div class="md:col-span-2 bg-white border border-slate-200/80 rounded-xl p-6 shadow-2xs space-y-4">
        <h2 class="text-sm font-bold text-slate-800 border-b border-slate-100 pb-3">Aktive Lieferanten</h2>
        {#if !suppliers.length}
          <div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
        {:else}
          <table class="w-full text-left border-collapse text-base">
            <thead>
              <tr class="border-b border-slate-100 text-sm font-bold text-slate-400 uppercase tracking-wider"><th class="py-2.5">Name</th><th class="py-2.5">E-Mail</th><th class="py-2.5">Kundennummer</th><th class="py-2.5 text-right">Aktion</th></tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              {#each suppliers as s, idx}
                <tr class="hover:bg-slate-50/40">
                  <td class="py-3 font-bold text-slate-800">{s.name}</td>
                  <td class="py-3 text-slate-650">{s.email}</td>
                  <td class="py-3 font-mono text-slate-650">{s.customerNumber}</td>
                  <td class="py-3 text-right"><button onclick={() => removeSupplier(s.id)} class="text-slate-450 hover:text-rose-600 cursor-pointer">Löschen</button></td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  @keyframes greenGlow {
    0% { background-color: rgba(16, 185, 129, 0.15); border-color: rgba(16, 185, 129, 0.45); transform: scale(1); }
    50% { background-color: rgba(16, 185, 129, 0.35); border-color: rgba(16, 185, 129, 0.9); transform: scale(1.02); }
    100% { background-color: transparent; border-color: rgba(226, 232, 240, 1); opacity: 0; transform: scale(0.95); }
  }
  .animate-green-fade { animation: greenGlow 1.5s cubic-bezier(0.4, 0, 0.2, 1) forwards; }
</style>
