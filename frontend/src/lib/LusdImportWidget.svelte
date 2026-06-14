<script>
  import { apiFetch, apiClient } from "./apiFetch.js";

  let isDragging = $state(false);
  let isUploading = $state(false);
  let fileToUpload = $state(/** @type {File|null} */ (null));
  let result = $state(/** @type {{inserted: number, updated: number, skipped: number}|null} */ (null));
  let errorMsg = $state("");

  /** @type {HTMLInputElement|null} */
  let fileInput;

  /** @param {DragEvent} e */
  function onDragOver(e) {
    e.preventDefault();
    isDragging = true;
  }

  function onDragLeave() {
    isDragging = false;
  }

  /** @param {DragEvent} e */
  function onDrop(e) {
    e.preventDefault();
    isDragging = false;
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFile(e.dataTransfer.files[0]);
    }
  }

  /** @param {Event} e */
  function onFileSelect(e) {
    const target = /** @type {HTMLInputElement} */ (e.target);
    if (target.files && target.files.length > 0) {
      handleFile(target.files[0]);
    }
  }

  /** @param {File} file */
  function handleFile(file) {
    errorMsg = "";
    result = null;
    if (!file.name.toLowerCase().endsWith(".csv")) {
      errorMsg = "Bitte lade eine gültige CSV-Datei (.csv) hoch.";
      return;
    }
    fileToUpload = file;
    uploadFile();
  }

  async function uploadFile() {
    if (!fileToUpload) return;
    
    isUploading = true;
    errorMsg = "";
    result = null;

    const formData = new FormData();
    formData.append("file", fileToUpload);

    try {
      const res = await apiFetch("/api/schueler/import-lusd", {
        method: "POST",
        body: formData // DO NOT set Content-Type header manually for FormData
      });
      
      if (!res.ok) {
        throw new Error(await res.text() || "Fehler beim Import.");
      }
      
      result = await res.json();
    } catch (e) {
      errorMsg = String(e);
    } finally {
      isUploading = false;
      fileToUpload = null;
      if (fileInput) fileInput.value = "";
    }
  }
</script>

<div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden flex flex-col font-sans max-w-xl mx-auto">
  <div class="px-5 py-4 border-b border-slate-100 bg-slate-50 flex justify-between items-center">
    <div>
      <h3 class="text-sm font-bold text-slate-800">LUSD Schnell-Import</h3>
      <p class="text-xs text-slate-500 mt-0.5">Schülerdaten (Klasse, Adresse) aktualisieren</p>
    </div>
  </div>

  <div class="p-5">
    {#if result}
      <div class="mb-5 p-4 bg-emerald-50 border border-emerald-200 rounded-xl flex items-start gap-4">
        <div class="text-2xl mt-0.5">✅</div>
        <div>
          <p class="text-sm font-bold text-emerald-800">Import erfolgreich abgeschlossen</p>
          <div class="text-xs text-emerald-700 mt-1.5 space-y-0.5">
            <p><strong>{result.inserted}</strong> Schüler neu angelegt</p>
            <p><strong>{result.updated}</strong> Schüler aktualisiert</p>
            <p><strong>{result.skipped}</strong> Einträge übersprungen</p>
          </div>
        </div>
      </div>
    {/if}

    {#if errorMsg}
      <div class="mb-5 p-3 bg-rose-50 border border-rose-200 rounded-xl text-xs font-semibold text-rose-700 flex items-center gap-2">
        <span class="text-base">⚠️</span> {errorMsg}
      </div>
    {/if}

    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div 
      class="border-2 border-dashed rounded-xl p-10 text-center transition-all cursor-pointer flex flex-col items-center justify-center min-h-[200px]
             {isDragging ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:border-slate-300 hover:bg-slate-50'}
             {isUploading ? 'opacity-50 pointer-events-none' : ''}"
      ondragover={onDragOver}
      ondragleave={onDragLeave}
      ondrop={onDrop}
      onclick={() => fileInput?.click()}
    >
      <input 
        type="file" 
        accept=".csv" 
        class="hidden" 
        bind:this={fileInput} 
        onchange={onFileSelect} 
        aria-label="LUSD CSV-Datei auswählen"
      />
      
      {#if isUploading}
        <div class="w-10 h-10 border-4 border-t-blue-600 border-blue-200 rounded-full animate-spin mb-4"></div>
        <p class="text-sm font-bold text-blue-800">Verarbeite CSV-Datei...</p>
        <p class="text-xs text-blue-600 mt-1">Die Daten werden mit der Datenbank abgeglichen.</p>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-slate-300 mb-4 transition-colors {isDragging ? 'text-blue-400' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
        </svg>
        <p class="text-sm font-bold text-slate-700">Klicken oder LUSD CSV-Datei hier ablegen</p>
        <p class="text-xs text-slate-500 mt-1.5 font-medium">Unterstützt .csv Exporte mit Trennzeichen (Semikolon)</p>
      {/if}
    </div>
  </div>
</div>
