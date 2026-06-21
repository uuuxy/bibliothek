<script>
  /**
   * @component StudentDirectoryToolbar
   * Toolbar für das Schülerverzeichnis mit Suche, "Neuer Schüler", LUSD-Import und Druck-Optionen.
   *
   * @prop {string} searchQuery - Der aktuelle Suchbegriff (bindable).
   * @prop {string} role - Die Rolle des aktuellen Nutzers (z.B. 'admin', 'mitarbeiter').
   * @prop {number} totalCount - Gesamtanzahl der Schüler.
   * @prop {number} filteredCount - Anzahl der gefilterten Schüler.
   * @prop {() => void} oncreate - Callback wenn "Neuer Schüler" geklickt wird.
   * @prop {(file: File) => void} onimport - Callback wenn eine LUSD-Datei ausgewählt wird.
   * @prop {() => void} onprint - Callback wenn "Klassensatz drucken" geklickt wird.
   */
  let {
    searchQuery = $bindable(""),
    role = "",
    totalCount = 0,
    filteredCount = 0,
    oncreate,
    onimport,
    onprint
  } = $props();

  /** @type {HTMLInputElement | null} */
  let fileInputEl = $state(null);

  function triggerImportPicker() {
    fileInputEl?.click();
  }

  /** @param {Event} event */
  function handleFileChange(event) {
    const target = /** @type {HTMLInputElement} */ (event.target);
    const file = target.files?.[0];
    if (file && onimport) {
      onimport(file);
    }
    target.value = ""; // Reset für erneute Auswahl
  }
</script>

<div class="flex items-center gap-4 bg-white p-4 rounded-2xl border border-slate-100 shadow-xs justify-between">
  <div class="flex flex-1 items-center gap-4">
    <div class="relative w-full max-w-md">
      <svg class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input 
        type="text" 
        aria-label="Schüler suchen" 
        placeholder="Nach Name, Klasse oder Barcode filtern..." 
        bind:value={searchQuery} 
        class="w-full pl-10 pr-4 py-2 bg-slate-55 border border-slate-200 rounded-xl text-base text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" 
      />
    </div>

    {#if role === 'admin' || role === 'mitarbeiter'}
      <button onclick={oncreate} aria-label="Neuen Schüler anlegen" class="inline-flex items-center gap-2 bg-blue-600 hover:bg-blue-750 text-white font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        <span>+ Neuer Schüler</span>
      </button>
    {/if}

    {#if role === 'admin'}
      <input type="file" accept=".csv" bind:this={fileInputEl} onchange={handleFileChange} class="hidden" aria-label="LUSD Datei auswählen" />
      <button onclick={triggerImportPicker} aria-label="LUSD-Import starten" class="inline-flex items-center gap-2 bg-slate-100 hover:bg-slate-200/80 text-slate-700 font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0 border border-slate-200">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
        </svg>
        <span>LUSD Import (CSV)</span>
      </button>
    {/if}

    {#if role === 'admin' || role === 'mitarbeiter'}
      <button onclick={onprint} aria-label="Klassensatz drucken" class="inline-flex items-center gap-2 bg-slate-100 hover:bg-slate-200/80 text-slate-700 font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0 border border-slate-200">
        <span aria-hidden="true">🖨️</span>
        <span>Klassensatz drucken</span>
      </button>
    {/if}
  </div>
  
  <div class="text-xs font-semibold text-slate-500">
    Einträge: {filteredCount} / {totalCount}
  </div>
</div>
