<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { playSuccessBeep, playErrorBeep } from "./audio.js";

  /** @type {any} */
  let selectedTitle = $state(null);
  
  /** @type {any[]} */
  let searchResults = $state([]);
  let searchQuery = $state("");
  let isSearching = $state(false);

  // Inventur State
  /** @type {any[]} */
  let expectedCopies = $state([]);
  /** @type {Set<string>} */
  let scannedIds = $state(new Set());
  let scanInput = $state("");
  let scanFinished = $state(false);
  let isFinalizing = $state(false);

  let missingCopies = $derived(expectedCopies.filter(c => !scannedIds.has(c.barcode_id)));

  async function searchTitles() {
    if (!searchQuery.trim()) {
      searchResults = [];
      return;
    }
    isSearching = true;
    try {
      const res = await apiFetch(`/api/public/opac/suche?q=${encodeURIComponent(searchQuery)}`);
      if (res.ok) {
        searchResults = await res.json();
      }
    } catch (e) {
      console.error("Search failed:", e);
    } finally {
      isSearching = false;
    }
  }

  /** @param {any} title */
  async function selectTitle(title) {
    selectedTitle = title;
    searchQuery = "";
    searchResults = [];
    expectedCopies = [];
    scannedIds.clear();
    scanFinished = false;
    
    // Hole alle Exemplare
    try {
      const res = await apiFetch(`/api/buecher/titel/${title.id}/exemplare`, { credentials: "include" });
      if (res.ok) {
        const copies = await res.json();
        // Wir erwarten im Regal nur Bücher, die verfuegbar sind (nicht ausgeliehen)
        expectedCopies = copies.filter((c) => c.ist_verfuegbar && c.ist_ausleihbar);
      }
    } catch (e) {
      alert("Fehler beim Laden der Exemplare.");
    }
  }

  /** @param {SubmitEvent} e */
  function handleScan(e) {
    e.preventDefault();
    const barcode = scanInput.trim();
    if (!barcode) return;
    
    scanInput = ""; // Reset für nächsten Scan

    // Prüfen, ob dieses Exemplar in der Liste ist
    const copy = expectedCopies.find(c => c.barcode_id === barcode);
    if (copy) {
      if (scannedIds.has(barcode)) {
        // Schon gescannt, Warnung!
        playErrorBeep();
      } else {
        const nextIds = new Set(scannedIds);
        nextIds.add(barcode);
        scannedIds = nextIds;
        playSuccessBeep();
      }
    } else {
      // Falsches Buch gescannt!
      playErrorBeep();
      alert(`Barcode ${barcode} gehört nicht zu diesem Titel oder ist aktuell ausgeliehen/gesperrt!`);
    }

    const inp = document.getElementById("regal-scan-input");
    if (inp) setTimeout(() => inp.focus(), 10);
  }

  function finishScan() {
    scanFinished = true;
  }

  async function markMissingAsLost() {
    if (missingCopies.length === 0) return;
    if (!confirm(`Sicher, dass ${missingCopies.length} Exemplare unwiderruflich als 'Verloren' gebucht werden sollen?`)) return;

    isFinalizing = true;
    const missingIds = missingCopies.map(c => c.id);

    try {
      const res = await apiClient.post(`/api/inventur/titel/${selectedTitle.id}/verlust-batch`, { exemplar_ids: missingIds });
      
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      alert(`${data.updated} Bücher wurden erfolgreich als verloren markiert.`);
      
      // Zurück zur Titelauswahl
      selectedTitle = null;
    } catch (err) {
      const e = /** @type {Error} */ (err);
      alert("Fehler: " + e.message);
    } finally {
      isFinalizing = false;
    }
  }

  $effect(() => {
    // Keep focus active in scan mode
    if (selectedTitle && !scanFinished) {
      const handleFocus = () => {
        const inp = document.getElementById("regal-scan-input");
        if (inp && document.activeElement !== inp) inp.focus();
      };
      document.addEventListener("click", handleFocus);
      handleFocus();
      return () => document.removeEventListener("click", handleFocus);
    }
  });
</script>

<div class="w-full text-slate-800">
  {#if !selectedTitle}
    <!-- Titelauswahl -->
    <div class="max-w-xl mx-auto space-y-6 mt-10 border border-slate-100 bg-white rounded-3xl p-8 shadow-sm">
      <div>
        <h2 class="text-2xl font-bold text-slate-900 mb-2">Titel auswählen</h2>
        <p class="text-sm text-slate-500">Welchen Titel möchtest du im Regal auf Fehlbestände prüfen?</p>
      </div>

      <div class="relative">
        <input 
          type="text" 
          bind:value={searchQuery} 
          oninput={searchTitles}
          placeholder="Buchtitel oder ISBN eingeben..." 
          class="w-full px-4 py-3 bg-slate-50 border border-slate-200 focus:border-blue-500 rounded-xl outline-none"
        />
        {#if searchResults.length > 0}
          <div class="absolute top-full left-0 right-0 mt-2 bg-white border border-slate-200 rounded-xl shadow-lg z-50 max-h-64 overflow-y-auto">
            {#each searchResults as item}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div 
                class="px-4 py-3 hover:bg-slate-50 cursor-pointer border-b border-slate-100 last:border-0 flex items-center gap-3"
                onclick={() => selectTitle(item)}
              >
                <div class="w-10 h-14 bg-slate-100 rounded shrink-0 flex items-center justify-center text-slate-400 font-bold text-xs overflow-hidden">
                  {#if item.coverUrl}<img src={item.coverUrl} alt="Cover" class="w-full h-full object-cover" />{:else}?{/if}
                </div>
                <div>
                  <h4 class="font-bold text-slate-800 text-sm truncate">{item.title}</h4>
                  <p class="text-xs text-slate-500 truncate">{item.author}</p>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <!-- Aktiver Scan-Modus -->
    <div class="flex flex-col lg:flex-row gap-6">
      
      <!-- Linke Seite: Meta & Scanner -->
      <div class="lg:w-1/3 flex flex-col gap-4">
        <div class="bg-white rounded-2xl p-6 border border-slate-200 shadow-sm flex items-start gap-4">
          {#if selectedTitle.coverUrl}
            <img src={selectedTitle.coverUrl} alt="Cover" class="w-16 rounded shadow-sm border border-slate-100" />
          {/if}
          <div>
            <h3 class="font-bold text-lg text-slate-900 leading-tight mb-1">{selectedTitle.title}</h3>
            <p class="text-xs text-slate-500 mb-2">Im Regal erwartet: <strong class="text-slate-800">{expectedCopies.length}</strong> Exemplare</p>
            <button onclick={() => selectedTitle = null} class="text-xs font-semibold text-blue-600 hover:text-blue-800 cursor-pointer">Anderen Titel wählen</button>
          </div>
        </div>

        {#if !scanFinished}
          <div class="bg-indigo-50 border border-indigo-100 rounded-2xl p-6 shadow-sm">
            <h4 class="font-bold text-indigo-900 mb-3 text-sm">Jetzt scannen!</h4>
            <form onsubmit={handleScan} class="relative">
              <input 
                id="regal-scan-input" 
                type="text" 
                bind:value={scanInput}
                autocomplete="off"
                placeholder="Barcode einscannen..."
                class="w-full px-4 py-3 bg-white border border-indigo-200 focus:border-indigo-500 rounded-xl outline-none font-mono text-center tracking-widest shadow-inner"
              />
            </form>
            <div class="mt-4 flex justify-end">
              <button onclick={finishScan} class="text-sm font-bold bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-xl shadow-sm transition-colors cursor-pointer">
                Scan abschließen
              </button>
            </div>
          </div>
        {/if}
      </div>

      <!-- Rechte Seite: Listen -->
      <div class="lg:w-2/3 bg-white border border-slate-200 rounded-3xl p-6 shadow-sm flex flex-col">
        <div class="flex items-center justify-between border-b border-slate-100 pb-4 mb-4">
          <h3 class="font-bold text-slate-800">Status der Exemplare</h3>
          <div class="text-sm font-semibold">
            <span class="text-emerald-600 bg-emerald-50 px-2 py-1 rounded-md">{scannedIds.size} gescannt</span>
            <span class="text-slate-300 mx-2">/</span>
            <span class="text-rose-600 bg-rose-50 px-2 py-1 rounded-md">{missingCopies.length} fehlen</span>
          </div>
        </div>

        <div class="grow overflow-y-auto max-h-[600px] space-y-2">
          {#each expectedCopies as c}
            <div class="flex items-center justify-between p-3 border rounded-xl transition-colors
              {scannedIds.has(c.barcode_id) ? 'bg-emerald-50/50 border-emerald-100' : (scanFinished ? 'bg-rose-50 border-rose-200' : 'bg-slate-50 border-slate-100')}">
              <div class="flex items-center gap-3">
                <span class="text-xl">{scannedIds.has(c.barcode_id) ? '🟢' : (scanFinished ? '🔴' : '⚪')}</span>
                <span class="font-mono text-sm font-semibold {scannedIds.has(c.barcode_id) ? 'text-emerald-800' : 'text-slate-700'}">{c.barcode_id}</span>
              </div>
              <span class="text-xs font-bold {scannedIds.has(c.barcode_id) ? 'text-emerald-600' : (scanFinished ? 'text-rose-600' : 'text-slate-400')}">
                {scannedIds.has(c.barcode_id) ? 'Vorhanden' : (scanFinished ? 'Fehlt' : 'Ausstehend')}
              </span>
            </div>
          {/each}
        </div>

        {#if scanFinished}
          <div class="mt-6 pt-6 border-t border-slate-100 flex items-center justify-between bg-rose-50/30 p-4 rounded-xl border-dashed">
            <div>
              <p class="font-bold text-rose-800 text-sm">Inventur-Abgleich beendet</p>
              <p class="text-xs text-rose-600/80 mt-1">Es fehlen <strong>{missingCopies.length}</strong> Exemplare im Regal.</p>
            </div>
            {#if missingCopies.length > 0}
              <button 
                onclick={markMissingAsLost}
                disabled={isFinalizing}
                class="bg-rose-600 hover:bg-rose-700 text-white font-bold text-sm px-5 py-2.5 rounded-xl shadow-sm transition-colors cursor-pointer disabled:opacity-50"
              >
                {isFinalizing ? 'Wird gebucht...' : 'Fehlende als "Verloren" markieren'}
              </button>
            {/if}
          </div>
        {/if}

      </div>
    </div>
  {/if}
</div>
