<script>
  import { onMount } from "svelte";
  import { appState } from "../inventur/lib/store.svelte.js";
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { playSuccessBeep, playErrorBeep } from "./audio.js";

  import OrderCreationPanel from "./components/bestellungen/OrderCreationPanel.svelte";
  import IncomingShipments from "./components/bestellungen/IncomingShipments.svelte";
  import OrderRecommendations from "./components/bestellungen/OrderRecommendations.svelte";
  import SupplierManager from "./components/bestellungen/SupplierManager.svelte";
  import PrintSuggestion from "./components/bestellungen/PrintSuggestion.svelte";

  let activeTab = $state("bestellungen");
  
  /** @type {any[]} */
  let suppliers = $state([]);
  let selectedSupplierIdx = $state(0), searchQuery = $state(""), showDropdown = $state(false);
  /** @type {any[]} */
  let searchResults = $state([]);
  /** @type {any[]} */
  let orderCart = $state([]);
  let orderTotal = $derived(orderCart.reduce((sum, item) => sum + (item.menge * (Number(item.preis) || 0)), 0));
  let submittingOrder = $state(false);
  let generateBarcodes = $state(false);
  /** @type {any} */
  let orderMessage = $state(null);
  /** @type {any[]} */
  let recommendations = $state([]);
  /** @type {any[]} */
  let incomingShipments = $state([]);
  let isReleasing = $state(false);
  let showGreenFade = $state(false);
  let scanningTitelId = $state(null);
  let scannedBarcode = $state("");
  /** @type {any[] | null} */
  let printSuggestion = $state(null);
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
      const res = await apiFetch("/api/lieferanten");
      if (res.ok) suppliers = (await res.json()) || [];
    } catch (err) { console.error("Fehler beim Laden der Lieferanten:", err); }
  }

  async function loadIncomingShipments() {
    try {
      const res = await apiFetch("/api/bestellungen/zulauf");
      if (res.ok) incomingShipments = (await res.json()) || [];
    } catch (err) { console.error("Fehler beim Laden des Wareneingangs:", err); }
  }

  async function fetchRecommendations() {
    try {
      const res = await apiFetch("/api/bestellungen");
      if (res.ok) recommendations = (await res.json()) || [];
    } catch (err) { console.error(err); }
  }

  /**
   * @param {string} name
   * @param {string} email
   * @param {string} customerNumber
   */
  async function addSupplier(name, email, customerNumber) {
    if (!name || !email || !customerNumber) return;
    try {
      const res = await apiClient.post("/api/lieferanten", { name, email, customerNumber });
      if (res.ok) {
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
      const res = await apiFetch(`/api/lieferanten/${id}`, { method: "DELETE" });
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
      searchResults = []; showDropdown = false; isbnPreview = null; isbnLoading = true;
      (async () => {
        try {
          const res = await apiClient.post("/api/buecher/aus-isbn", { isbn: cleanQuery });
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
          const res = await apiClient.post("/api/action", { query: searchQuery });
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
    else { orderCart.push({ id: book.titel_id ?? book.id, titel: book.titel, autor: book.autor, isbn: book.isbn ?? book.ISBN, verlag: book.verlag ?? "", cover_url: book.cover_url ?? "", menge: 1, preis: 0.00 }); }
    searchQuery = ""; searchResults = []; showDropdown = false; isbnPreview = null;
  }

  /** @param {number} idx */
  function removeFromCart(idx) { orderCart.splice(idx, 1); }

  async function submitOrder() {
    if (!orderCart.length || !suppliers.length) return;
    submittingOrder = true; orderMessage = null;
    const supplier = suppliers[selectedSupplierIdx];
    try {
      const res = await apiClient.post("/api/orders", {
          supplier_id: supplier.id,
          items: orderCart.map(item => ({ titel_id: item.id, menge: item.menge, preis: Number(item.preis) || 0 })),
          generate_barcodes: generateBarcodes
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

  /** @param {string} titelId */
  async function receiveItem(titelId) {
    if (!scannedBarcode) return;
    try {
      const res = await apiClient.post("/api/orders/receive", { titel_id: titelId, barcode: scannedBarcode });
      if (res.ok) {
        playSuccessBeep();
        showGreenFade = true;
        scannedBarcode = "";
        scanningTitelId = null;
        await loadIncomingShipments();
        setTimeout(() => { showGreenFade = false; fetchRecommendations(); }, 1500);
      } else {
        const txt = await res.text();
        throw new Error(txt);
      }
    } catch (err) {
      playErrorBeep();
      const msg = err instanceof Error ? err.message : String(err);
      alert("Fehler beim Scannen: " + msg);
      const currentId = scanningTitelId;
      scanningTitelId = null;
      setTimeout(() => { scanningTitelId = currentId; scannedBarcode = ""; }, 10);
    }
  }

  async function releaseIncoming() {
    if (!incomingShipments.length) return;
    isReleasing = true;
    printSuggestion = null;
    try {
      const res = await apiFetch("/api/bestellungen/freigeben", { method: "POST" });
      if (res.ok) {
        const data = await res.json();
        showGreenFade = true;
        if (data.released_items && data.released_items.length > 0) {
          const needsPrinting = data.released_items.filter(/** @type {any} */ item => !item.etikett_gedruckt);
          if (needsPrinting.length > 0) {
            printSuggestion = needsPrinting;
          }
        }
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

  async function releaseNaacher() {
    if (!incomingShipments.length) return;
    isReleasing = true;
    try {
      const res = await apiClient.post("/api/orders/release");
      if (res.ok) {
        const data = await res.json();
        showGreenFade = true;
        orderMessage = { type: "success", text: `Lieferung freigegeben. ${data.released_count} Exemplare sind nun im aktiven Bestand.` };
        await loadIncomingShipments();
        setTimeout(() => { showGreenFade = false; fetchRecommendations(); }, 1500);
      } else {
        const txt = await res.text();
        alert("Fehler beim Freigeben (Naacher): " + txt);
      }
    } catch (err) {
      console.error(err);
    } finally { isReleasing = false; }
  }

  function handlePrintSuggestion() {
    appState.pendingPrintCopies = printSuggestion;
    printSuggestion = null;
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
      <OrderCreationPanel
        {suppliers}
        {orderCart}
        {orderTotal}
        {submittingOrder}
        bind:selectedSupplierIdx
        bind:searchQuery
        bind:searchResults
        bind:showDropdown
        bind:isbnPreview
        bind:isbnLoading
        bind:generateBarcodes
        onSearchInput={handleSearchInput}
        onAddToCart={addToCart}
        onRemoveFromCart={removeFromCart}
        onSubmitOrder={submitOrder}
      />

      <div class="lg:col-span-4 space-y-6">
        <PrintSuggestion 
          {printSuggestion} 
          onPrint={handlePrintSuggestion} 
        />

        <IncomingShipments 
          {incomingShipments}
          {showGreenFade}
          {isReleasing}
          bind:scanningTitelId
          bind:scannedBarcode
          onReceiveItem={receiveItem}
          onReleaseAll={releaseIncoming}
          onReleaseNaacher={releaseNaacher}
        />

        <OrderRecommendations 
          {recommendations}
          onAddToCart={addToCart}
        />
      </div>
    </div>
  {/if}

  {#if activeTab === "lieferanten"}
    <SupplierManager 
      {suppliers}
      onAddSupplier={addSupplier}
      onRemoveSupplier={removeSupplier}
    />
  {/if}
</div>
