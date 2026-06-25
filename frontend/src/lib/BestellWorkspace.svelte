<script>
  import { onMount } from "svelte";
  import { appState } from "../inventur/lib/store.svelte.js";
  import { apiGet, apiPost, apiPut, apiDelete } from "./apiFetch.js";
  import { toastStore } from "./stores/toastStore.svelte.js";
  import { playSuccessBeep, playErrorBeep } from "./audio.js";

  import OrderCreationPanel from "./components/bestellungen/OrderCreationPanel.svelte";
  import IncomingShipments from "./components/bestellungen/IncomingShipments.svelte";
  import WareneingangView from "./components/bestellungen/WareneingangView.svelte";
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
  let generateBarcodes = $state(true);
  /** @type {any[]} */
  let recommendations = $state([]);
  /** @type {any[]} */
  let incomingShipments = $state([]);
  let isReleasing = $state(false);
  let showGreenFade = $state(false);
  let showWareneingang = $state(false);
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
      suppliers = await apiGet("/api/lieferanten") || [];
    } catch { /* toast handles error */ }
  }

  async function loadIncomingShipments() {
    try {
      incomingShipments = await apiGet("/api/bestellungen/zulauf") || [];
    } catch { /* toast handles error */ }
  }

  async function fetchRecommendations() {
    try {
      recommendations = await apiGet("/api/bestellungen") || [];
    } catch { /* toast handles error */ }
  }

  /**
   * @param {string} name
   * @param {string} email
   * @param {string} customerNumber
   */
  async function addSupplier(name, email, customerNumber) {
    if (!name || !email || !customerNumber) return;
    try {
      await apiPost("/api/lieferanten", { name, email, customerNumber });
      await loadSuppliers();
    } catch { /* toast handles error */ }
  }

  /** @param {string} id */
  async function editSupplier(id, name, email, customerNumber) {
    try {
      await apiPut(`/api/lieferanten/${id}`, { name, email, customerNumber });
      await loadSuppliers();
      toastStore.addToast("Lieferant aktualisiert.", "success");
    } catch { /* apiPut already shows error toast */ }
  }

  async function removeSupplier(id) {
    try {
      await apiDelete(`/api/lieferanten/${id}`);
      await loadSuppliers();
      selectedSupplierIdx = Math.max(0, Math.min(selectedSupplierIdx, suppliers.length - 1));
    } catch { /* toast handles error */ }
  }

  /** @type {any} */
  let searchTimeout;
  function handleSearchInput() {
    clearTimeout(searchTimeout);
    const raw = searchQuery.trim();
    if (raw.length < 2) { searchResults = []; showDropdown = false; isbnPreview = null; return; }

    isbnPreview = null;
    const performSearch = async () => {
      isbnLoading = true;
      try {
        const data = await apiPost("/api/bestellungen/suche", { query: raw });
        searchResults = data || [];
        showDropdown = searchResults.length > 0;
      } catch {
        searchResults = [];
        showDropdown = false;
      } finally {
        isbnLoading = false;
      }
    };
    searchTimeout = setTimeout(performSearch, 300);
  }

  /** @param {any} book */
  function addToCart(book, menge = 1, generateBarcodes = false) {
    const existing = orderCart.find(item => item.id === book.id || (book.isbn && item.isbn === book.isbn));
    if (existing) { 
        existing.menge += menge;
        if (generateBarcodes) existing.generate_barcodes = true;
    }
    else { orderCart.push({ id: book.titel_id ?? book.id, titel: book.titel, autor: book.autor, isbn: book.isbn ?? book.ISBN, verlag: book.verlag ?? "", cover_url: book.cover_url ?? "", menge: menge, preis: 0.00, generate_barcodes: generateBarcodes }); }
    searchQuery = ""; searchResults = []; showDropdown = false; isbnPreview = null;
  }

  /** @param {number} idx */
  function removeFromCart(idx) { orderCart.splice(idx, 1); }

  async function submitOrder() {
    if (!orderCart.length || !suppliers.length) return;
    submittingOrder = true;
    const supplier = suppliers[selectedSupplierIdx];
    try {
      const data = await apiPost("/api/orders", {
          supplier_id: supplier.id,
          items: orderCart.map(item => ({ 
              titel_id: item.id, 
              menge: item.menge, 
              preis: Number(item.preis) || 0,
              generate_barcodes: generateBarcodes ? item.generate_barcodes : false
          }))
      });
      orderCart = [];
      const toastType = data.status === "warning" ? "error" : "success";
      toastStore.addToast(data.message + ` (${data.ordered_qty} Barcodes reserviert.)`, toastType);
      await loadIncomingShipments();
      fetchRecommendations();
    } catch {
      // toast handles error
    } finally { submittingOrder = false; }
  }

  function handleOpenWareneingang() {
    showWareneingang = true;
  }

  async function handleShipmentReceived() {
    showWareneingang = false;
    showGreenFade = true;
    await loadIncomingShipments();
    fetchRecommendations();
    setTimeout(() => { showGreenFade = false; }, 1500);
  }

  async function releaseIncoming() {
    if (!incomingShipments.length) return;
    isReleasing = true;
    printSuggestion = null;
    try {
      const data = await apiPost("/api/bestellungen/freigeben");
      showGreenFade = true;
      if (data && data.released_items && data.released_items.length > 0) {
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
    } catch {
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
      const data = await apiPost("/api/orders/release");
      showGreenFade = true;
      toastStore.addToast(`Lieferung freigegeben. ${data.released_count} Exemplare sind nun im aktiven Bestand.`, "success");
      await loadIncomingShipments();
      setTimeout(() => { showGreenFade = false; fetchRecommendations(); }, 1500);
    } catch {
      // error toast shown
    } finally { isReleasing = false; }
  }

  function handlePrintSuggestion() {
    appState.pendingPrintCopies = printSuggestion;
    printSuggestion = null;
  }
</script>

<div class="w-full h-full text-slate-800 font-sans flex flex-col gap-6">
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

  {#if activeTab === "bestellungen"}
    {#if showWareneingang}
      <WareneingangView 
        {incomingShipments}
        onBack={() => showWareneingang = false}
        onReceived={handleShipmentReceived}
      />
    {:else}
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
          bind:globalGenerateBarcodes={generateBarcodes}
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
            onOpenWareneingang={handleOpenWareneingang}
          />

          <OrderRecommendations 
            {recommendations}
            onAddToCart={addToCart}
          />
        </div>
      </div>
    {/if}
  {/if}

  {#if activeTab === "lieferanten"}
    <SupplierManager
      {suppliers}
      onAddSupplier={addSupplier}
      onEditSupplier={editSupplier}
      onRemoveSupplier={removeSupplier}
    />
  {/if}
</div>
