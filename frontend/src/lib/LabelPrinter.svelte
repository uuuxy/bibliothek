<script>
  import { onMount } from "svelte";

  // State Runes (Svelte 5)
  let searchVal = $state("");
  /** @type {any[]} */
  let searchResults = $state([]);
  let isSearching = $state(false);

  /** @type {any[]} */
  let classGroups = $state([]);
  let selectedClass = $state("");
  /** @type {any[]} */
  let classBooks = $state([]);

  /** @type {any} */
  let selectedTitle = $state(null);
  let barcodeType = $state("code39"); // "code39" | "qr"
  let labelBorder = $state(true);
  let startPosition = $state(1); // 1 to 21

  // Generation mode: "existing" | "new"
  let generationMode = $state("existing");
  /** @type {any[]} */
  let existingCopies = $state([]);
  let loadingCopies = $state(false);
  let newQuantity = $state(9);
  let newStartNum = $state(20060);

  // Derived printable labels list
  /** @type {any[]} */
  let finalLabels = $derived.by(() => {
    if (!selectedTitle) return [];

    let rawList = [];
    if (generationMode === "existing") {
      rawList = existingCopies
        .filter(c => c.checked)
        .map(c => ({
          barcode_id: c.barcode_id,
          titel: selectedTitle.titel,
          autor: selectedTitle.autor || ""
        }));
    } else {
      rawList = Array.from({ length: Math.max(1, newQuantity) }, (_, i) => ({
        barcode_id: `B-${newStartNum + i}`,
        titel: selectedTitle.titel,
        autor: selectedTitle.autor || ""
      }));
    }

    // Insert empty/blank labels to offset the starting position on the A4 sheet
    const offsetCount = Math.max(0, startPosition - 1);
    const offsetLabels = Array.from({ length: offsetCount }, () => ({
      isBlank: true
    }));

    return [...offsetLabels, ...rawList];
  });

  // Load class groups on mount
  async function loadClassGroups() {
    try {
      const res = await fetch("/api/class-books");
      if (res.ok) {
        const body = await res.json();
        if (body && body.data) {
          classGroups = body.data;
        }
      }
    } catch (err) {
      console.error("Fehler beim Laden der Klassengruppen:", err);
    }
  }

  // Handle class selection
  function handleClassChange() {
    const group = classGroups.find(g => g.className === selectedClass);
    if (group) {
      classBooks = group.books || [];
    } else {
      classBooks = [];
    }
    selectedTitle = null;
    existingCopies = [];
  }

  // Handle title search in catalog buecher_titel
  /** @type {any} */
  let searchTimeout = null;
  function handleSearchInput() {
    if (searchTimeout) clearTimeout(searchTimeout);
    if (!searchVal.trim()) {
      searchResults = [];
      return;
    }
    isSearching = true;
    searchTimeout = setTimeout(async () => {
      try {
        const res = await fetch("/api/action", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ query: searchVal.trim() })
        });
        if (res.ok) {
          const body = await res.json();
          if (body.type === "search_results") {
            searchResults = body.search_results || [];
          }
        }
      } catch (err) {
        console.error("Fehler bei Buchtitelsuche:", err);
      } finally {
        isSearching = false;
      }
    }, 300);
  }

  // Load copies for a selected book title
  /** @param {any} titleObj */
  async function selectBookTitle(titleObj) {
    selectedTitle = titleObj;
    searchResults = [];
    searchVal = titleObj.titel;
    selectedClass = "";
    classBooks = [];
    await loadExistingCopies();
  }

  // Load existing copies from buecher_exemplare
  async function loadExistingCopies() {
    if (!selectedTitle) return;
    loadingCopies = true;
    try {
      const res = await fetch(`/api/buecher/titel/${selectedTitle.id}/exemplare`);
      if (res.ok) {
        const data = await res.json();
        existingCopies = (data || []).map((/** @type {any} */ c) => ({
          ...c,
          checked: true
        }));
      } else {
        existingCopies = [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Exemplare:", err);
      existingCopies = [];
    } finally {
      loadingCopies = false;
    }
  }

  onMount(() => {
    loadClassGroups();
  });

  function triggerPrint() {
    window.print();
  }
</script>

<div class="w-full space-y-6 no-print text-slate-800 animate-fade-in">
  
  <!-- Header Info -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-100 pb-5">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Massen-Etikettendruck</span>
      <h2 class="text-2xl font-bold text-slate-900">Buch-Barcodes drucken</h2>
      <p class="text-xs text-slate-500 font-medium">Wähle einen Buchtitel, lege die Barcodes fest und drucke auf A4-Etikettenbögen.</p>
    </div>
    
    <button onclick={triggerPrint} disabled={!selectedTitle || finalLabels.length === 0} class="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:bg-slate-200 disabled:text-slate-400 disabled:cursor-not-allowed text-white font-bold transition-all flex items-center gap-2 shadow-xs cursor-pointer">
      <span>🖨️ A4-Bogen drucken</span>
    </button>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
    
    <!-- Left Configuration Panel (5 cols) -->
    <div class="lg:col-span-5 space-y-6 text-left">
      
      <!-- Step 1: Selection -->
      <div class="p-5 rounded-2xl bg-white border border-slate-100 shadow-sm space-y-4">
        <h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-mono font-bold">1. Titel / Klassensatz wählen</h3>
        
        <!-- Tab selector for search vs class set -->
        <div class="space-y-3">
          <!-- Autocomplete search -->
          <div class="space-y-1.5">
            <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Buchtitel im Katalog suchen</span>
            <div class="relative">
              <input type="text" bind:value={searchVal} oninput={handleSearchInput} placeholder="Titel, Autor oder ISBN eingeben..." class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500" />
              {#if isSearching}
                <div class="absolute right-3 top-1/2 -translate-y-1/2">
                  <div class="w-3.5 h-3.5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                </div>
              {/if}
            </div>
            
            {#if searchResults.length > 0}
              <div class="relative">
                <div class="absolute left-0 right-0 mt-1 bg-white border border-slate-100 rounded-xl shadow-xl z-20 max-h-48 overflow-y-auto divide-y divide-slate-50">
                  {#each searchResults as r}
                    <button onclick={() => selectBookTitle(r)} class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 transition-colors flex flex-col gap-0.5 cursor-pointer">
                      <span class="text-xs font-bold text-slate-900">{r.titel}</span>
                      <span class="text-[10px] text-slate-450">{r.autor || 'Unbekannt'} · {r.verlag || 'Kein Verlag'}</span>
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>

          <!-- Divider -->
          <div class="relative flex py-1 items-center">
            <div class="grow border-t border-slate-100"></div>
            <span class="shrink mx-3 text-[9px] uppercase tracking-wider font-mono text-slate-400 font-bold">ODER</span>
            <div class="grow border-t border-slate-100"></div>
          </div>

          <!-- Class Selection -->
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Aus Klasse laden</span>
              <select bind:value={selectedClass} onchange={handleClassChange} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550">
                <option value="">-- Klasse wählen --</option>
                {#each classGroups as group}
                  <option value={group.className}>{group.className}</option>
                {/each}
              </select>
            </div>

            <div class="space-y-1.5">
              <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Buch aus Klasse</span>
              <select disabled={!selectedClass} onchange={(e) => {
                const bookId = /** @type {any} */ (e.target).value;
                const book = classBooks.find(b => String(b.id) === bookId);
                if (book) {
                  selectBookTitle({
                    id: String(book.id),
                    titel: book.title,
                    autor: book.author
                  });
                }
              }} class="w-full bg-slate-50 border border-slate-200 disabled:opacity-50 disabled:cursor-not-allowed rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none">
                <option value="">-- Buch wählen --</option>
                {#each classBooks as book}
                  <option value={String(book.id)}>{book.title}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>
      </div>

      <!-- Step 2: Barcodes & Mode -->
      {#if selectedTitle}
        <div class="p-5 rounded-2xl bg-white border border-slate-100 shadow-sm space-y-4">
          <h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-mono font-bold">2. Barcodes generieren</h3>

          <!-- Selection mode -->
          <div class="flex bg-slate-100 p-0.5 rounded-lg border border-slate-200/40 text-xs">
            <button onclick={() => generationMode = "existing"} class="flex-1 text-center py-1 rounded-md font-bold transition-all cursor-pointer {generationMode === 'existing' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-700'}">Vorhandene Exemplare</button>
            <button onclick={() => generationMode = "new"} class="flex-1 text-center py-1 rounded-md font-bold transition-all cursor-pointer {generationMode === 'new' ? 'bg-white text-slate-800 shadow-xs' : 'text-slate-500 hover:text-slate-700'}">Neue Barcodes</button>
          </div>

          {#if generationMode === "existing"}
            <div class="space-y-2">
              <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Exemplare auswählen ({existingCopies.length} gefunden)</span>
              {#if loadingCopies}
                <div class="flex items-center justify-center py-4">
                  <div class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                </div>
              {:else if existingCopies.length === 0}
                <p class="text-[11px] text-slate-450">Keine physischen Exemplare in der Datenbank vorhanden.</p>
              {:else}
                <div class="max-h-40 overflow-y-auto border border-slate-100 rounded-xl divide-y divide-slate-50 p-2 space-y-1 bg-slate-50/50">
                  {#each existingCopies as copy}
                    <label class="flex items-center space-x-3 text-xs text-slate-700 cursor-pointer p-1.5 hover:bg-slate-50 rounded-lg">
                      <input type="checkbox" bind:checked={copy.checked} class="accent-blue-600 w-4 h-4 rounded border-slate-200 bg-white" />
                      <span class="font-mono font-bold text-slate-800">{copy.barcode_id}</span>
                      <span class="text-[10px] text-slate-450 font-sans">({copy.zustand_notiz || 'Neuwertig'})</span>
                    </label>
                  {/each}
                </div>
              {/if}
            </div>
          {:else}
            <!-- Generating new sequential labels -->
            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-1.5">
                <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Menge</span>
                <input type="number" min="1" max="100" bind:value={newQuantity} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550" />
              </div>
              <div class="space-y-1.5">
                <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Start-Ziffer (B-)</span>
                <input type="number" min="1" bind:value={newStartNum} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550" />
              </div>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Step 3: Print Layout settings -->
      <div class="p-5 rounded-2xl bg-white border border-slate-100 shadow-sm space-y-4">
        <h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-mono font-bold">3. Layout-Optionen</h3>

        <div class="space-y-3.5">
          <div class="space-y-1.5">
            <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Startposition auf dem A4-Bogen</span>
            <select bind:value={startPosition} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550">
              {#each Array.from({ length: 21 }, (_, i) => i + 1) as pos}
                <option value={pos}>Etikett Position {pos} {pos === 1 ? '(Ganz oben links)' : ''}</option>
              {/each}
            </select>
            <p class="text-[10px] text-slate-400">Verhindert Verschnitt bei bereits genutzten Etikettenbögen.</p>
          </div>

          <div class="space-y-1.5">
            <span class="text-[10px] uppercase font-bold text-slate-450 font-mono block">Barcode-Ausgabe</span>
            <select bind:value={barcodeType} class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none">
              <option value="code39">Code39 (1D Standard)</option>
              <option value="qr">QR-Code (2D)</option>
            </select>
          </div>

          <label class="flex items-center space-x-3 text-xs text-slate-705 cursor-pointer select-none">
            <input type="checkbox" bind:checked={labelBorder} class="accent-blue-600 w-4 h-4 rounded border-slate-200 bg-white" />
            <span>Hilfsrahmen / Begrenzungslinien auf Etikett zeichnen</span>
          </label>
        </div>
      </div>

    </div>

    <!-- Right Preview Panel (7 cols) -->
    <div class="lg:col-span-7 flex flex-col items-center justify-start p-6 bg-slate-50 border border-dashed border-slate-200 rounded-3xl min-h-[500px]">
      <span class="text-[10px] uppercase tracking-widest text-slate-400 font-bold font-mono mb-4">A4 Etiketten-Vorschau (3 Spalten x 7 Zeilen)</span>
      
      {#if !selectedTitle}
        <div class="grow flex flex-col items-center justify-center text-slate-400 py-12">
          <span>Kein Buch ausgewählt</span>
          <span class="text-[10px] mt-1 text-slate-450">Suche einen Titel links, um die Live-Vorschau zu aktivieren.</span>
        </div>
      {:else if finalLabels.length === 0}
        <div class="grow flex flex-col items-center justify-center text-slate-400 py-12">
          <span>Keine Etiketten gewählt</span>
          <span class="text-[10px] mt-1 text-slate-450">Wähle mindestens ein Exemplar oder erhöhe die Menge.</span>
        </div>
      {:else}
        <!-- A4 Page Mockup container -->
        <div class="bg-white border border-slate-300 shadow-2xl p-6 relative flex flex-col items-center select-none" style="width: 140mm; min-height: 198mm; font-size: 5px;">
          <div class="grid grid-cols-3 gap-x-2 gap-y-1.5 w-full justify-center">
            {#each finalLabels as lbl}
              {#if lbl.isBlank}
                <!-- Blank Label placeholder representation -->
                <div class="border border-dashed border-slate-200 bg-slate-50 flex items-center justify-center" style="width: 40mm; height: 23mm;">
                  <span class="text-[6px] text-slate-350 tracking-wider font-mono font-bold">LEER</span>
                </div>
              {:else}
                <div class="bg-white p-1 text-slate-800 text-left overflow-hidden flex flex-col justify-between {labelBorder ? 'border border-slate-300' : ''}" style="width: 40mm; height: 23mm; font-size: 5px;">
                  <div class="font-extrabold text-slate-900 truncate tracking-tight">{lbl.titel}</div>
                  <div class="text-slate-550 truncate -mt-0.5">{lbl.autor || 'Unbekannt'}</div>
                  <div class="flex flex-col items-center justify-center grow pt-1">
                    <img src="/api/barcode?content={lbl.barcode_id}&qr={barcodeType === 'qr'}&width=150&height=50" class="{barcodeType === 'qr' ? 'h-6 w-6' : 'h-4 w-full'} object-contain" alt="Barcode" />
                    <span class="font-mono text-[5px] mt-0.5 font-bold tracking-widest text-slate-600">{lbl.barcode_id}</span>
                  </div>
                </div>
              {/if}
            {/each}
          </div>
        </div>
      {/if}
    </div>

  </div>
</div>

<!-- Print Output (Invisible on screen, visible on print) -->
<div class="print-rendered-output a4_grid">
  <div class="print-labels-grid">
    {#each finalLabels as lbl}
      {#if lbl.isBlank}
        <!-- Hidden blank box keeping grid positions correct -->
        <div class="print-label-box border-none opacity-0"></div>
      {:else}
        <div class="print-label-box {labelBorder ? 'border border-black' : ''} p-2 text-black bg-white flex flex-col justify-between">
          <div class="font-extrabold text-zinc-950 truncate leading-none text-[8pt]" style="width: 50mm;">{lbl.titel}</div>
          <div class="text-zinc-700 font-medium leading-none text-[7pt] mt-0.5">{lbl.autor}</div>
          <div class="flex flex-col items-center justify-center grow pt-1">
            <img src="/api/barcode?content={lbl.barcode_id}&qr={barcodeType === 'qr'}&width=220&height=70" class="{barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[7mm]'} object-contain" alt="Barcode" />
            <span class="font-mono font-bold mt-1 text-[6.5pt] tracking-widest text-zinc-800">{lbl.barcode_id}</span>
          </div>
        </div>
      {/if}
    {/each}
  </div>
</div>
