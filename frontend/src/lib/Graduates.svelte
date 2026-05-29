<script>
  import { onMount } from "svelte";

  // State Runes
  /** @type {any[]} */
  let graduates = $state([]);
  let loading = $state(true);
  
  let showImportModal = $state(false);
  /** @type {File | null} */
  let selectedFile = $state(null);
  let importLoading = $state(false);
  /** @type {any} */
  let importResult = $state(null);
  /** @type {string | null} */
  let importError = $state(null);

  // Fetch graduates list from backend api
  async function fetchGraduates() {
    try {
      const res = await fetch("/api/abgaenger");
      if (!res.ok) throw new Error("Fehler beim Laden");
      graduates = await res.json();
    } catch (err) {
      console.error("Graduates error:", err);
    } finally {
      loading = false;
    }
  }

  /**
   * @param {Event} e
   */
  function handleFileChange(e) {
    const target = /** @type {HTMLInputElement} */ (e.target);
    if (target.files && target.files[0]) {
      selectedFile = target.files[0];
      importError = null;
    }
  }

  async function startImport() {
    if (!selectedFile || importLoading) return;
    importLoading = true;
    importError = null;

    const formData = new FormData();
    formData.append("file", selectedFile);

    try {
      const res = await fetch("/api/import/lusd", {
        method: "POST",
        body: formData
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Fehler beim LUSD-Import");
      }
      importResult = await res.json();
    } catch (err) {
      const error = /** @type {any} */ (err);
      importError = error.message || String(error);
    } finally {
      importLoading = false;
    }
  }

  function closeImportModal() {
    showImportModal = false;
    selectedFile = null;
    importResult = null;
    importError = null;
    importLoading = false;
    fetchGraduates();
  }

  onMount(() => {
    // Initial fetch
    fetchGraduates();

    // Listen to Go SSE events for instant UI synchronization
    const source = new EventSource("/events");
    
    // When a book is returned or transferred via the Omnibox,
    // refetch the graduates list to verify if the student is cleared.
    source.addEventListener("action", (e) => {
      try {
        const actionData = JSON.parse(e.data);
        if (actionData.event === "rueckgabe" || actionData.event === "fremdrueckgabe") {
          fetchGraduates();
        }
      } catch (err) {
        console.error("Failed to parse SSE payload:", err);
      }
    });

    return () => {
      source.close();
    };
  });
</script>

<div class="w-full space-y-6 text-slate-800">
  
  <!-- Header Info -->
  <div class="flex items-center justify-between border-b border-slate-100 pb-5">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Echtzeit-Überwachung</span>
      <h2 class="text-2xl font-bold text-slate-900">Abgänger-Entlastung</h2>
      <p class="text-xs text-slate-500">Schüler der Abschlussklassen 9h, 10r und 13 mit noch ausstehenden Lehrmitteln.</p>
    </div>
    <div class="flex items-center space-x-4">
      <button onclick={() => showImportModal = true} class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer">
        📥 LUSD-Datei importieren
      </button>
      <div class="h-2.5 w-2.5 rounded-full bg-emerald-500 animate-pulse shrink-0" title="Live-Synchronisation aktiv"></div>
    </div>
  </div>

  {#if loading}
    <div class="py-12 flex justify-center items-center">
      <div class="w-8 h-8 border-2 border-t-blue-600 border-blue-100 rounded-full animate-spin"></div>
    </div>
  {:else if graduates.length === 0}
    <!-- Completed clearing UI state -->
    <div class="py-12 text-center space-y-3 animate-fade-in">
      <div class="w-16 h-16 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-600 mx-auto">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <h3 class="font-bold text-slate-800">Alle Abgänger entlastet!</h3>
      <p class="text-xs text-slate-500 max-w-xs mx-auto">Keine offenen Lehrmittel oder unbezahlten Schadensfälle in den Klassen 9h, 10r und 13.</p>
    </div>
  {:else}
    <!-- Active list of graduates with dues -->
    <div class="overflow-x-auto">
      <table class="w-full text-left text-base border-collapse">
        <thead>
          <tr class="border-b border-slate-100 text-slate-450 text-sm font-mono uppercase">
            <th class="py-3 px-4">Klasse</th>
            <th class="py-3 px-4">Name</th>
            <th class="py-3 px-4">Barcode-ID</th>
            <th class="py-3 px-4">Sperr-Status</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-50">
          {#each graduates as student (student.id)}
            <tr class="hover:bg-slate-50/85 transition-colors animate-slide-up">
              <td class="py-3.5 px-4 font-mono font-bold text-blue-600">{student.klasse}</td>
              <td class="py-3.5 px-4 text-slate-700 font-semibold">{student.vorname} {student.nachname}</td>
              <td class="py-3.5 px-4 text-slate-400 font-mono text-xs">{student.barcode_id}</td>
              <td class="py-3.5 px-4">
                {#if student.ist_gesperrt}
                  <span class="text-[10px] px-2 py-0.5 rounded bg-rose-50 border border-rose-100 text-rose-600 font-semibold">Sperre aktiv</span>
                {:else}
                  <span class="text-[10px] px-2 py-0.5 rounded bg-slate-100 text-slate-400 font-medium">Bereit</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showImportModal}
  <div class="fixed inset-0 bg-slate-900/10 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="w-full max-w-md p-6 rounded-3xl bg-white border border-slate-100 shadow-2xl space-y-5 animate-scale-up">
      <div class="flex justify-between items-center pb-2 border-b border-slate-150/50">
        <h3 class="font-bold text-slate-800 text-base">LUSD-Schülerdatei importieren</h3>
        <button onclick={closeImportModal} class="text-slate-400 hover:text-slate-650 p-1 cursor-pointer" aria-label="Schließen">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      {#if importResult}
        <!-- Success State -->
        <div class="space-y-4 animate-fade-in">
          <div class="p-4 rounded-2xl bg-emerald-50 border border-emerald-100 flex items-center space-x-3 text-emerald-800 text-sm font-semibold">
            <span>🎉</span>
            <span>Import erfolgreich abgeschlossen!</span>
          </div>

          <div class="grid grid-cols-3 gap-3 text-center">
            <div class="p-3 bg-slate-50 border border-slate-100 rounded-2xl">
              <span class="text-[10px] uppercase font-bold text-slate-400 font-mono block">Neu</span>
              <span class="text-xl font-black text-slate-800 font-mono">{importResult.neu}</span>
            </div>
            <div class="p-3 bg-slate-50 border border-slate-100 rounded-2xl">
              <span class="text-[10px] uppercase font-bold text-slate-400 font-mono block">Aktualisiert</span>
              <span class="text-xl font-black text-slate-800 font-mono">{importResult.aktualisiert}</span>
            </div>
            <div class="p-3 bg-slate-50 border border-slate-100 rounded-2xl">
              <span class="text-[10px] uppercase font-bold text-slate-400 font-mono block">Abgänger</span>
              <span class="text-xl font-black text-rose-600 font-mono">{importResult.abgaenger_mit_offenen_buechern}</span>
            </div>
          </div>
          <p class="text-[11px] text-slate-450 text-center font-medium leading-relaxed">
            Schüler, die nicht in der Importdatei enthalten waren, wurden als Abgänger markiert. Davon haben {importResult.abgaenger_mit_offenen_buechern} Schüler noch nicht zurückgegebene Lehrmittel.
          </p>

          <button onclick={closeImportModal} class="w-full py-2.5 bg-slate-900 hover:bg-slate-850 text-white rounded-xl text-xs font-bold transition-all text-center cursor-pointer">
            Fertigstellen
          </button>
        </div>
      {:else}
        <!-- Upload State -->
        <div class="space-y-4">
          <p class="text-xs text-slate-500 font-medium leading-relaxed">
            Lade die LUSD-Exportdatei (.csv) hoch. Bestandsdaten werden aktualisiert, neue Schüler angelegt und Abgänger automatisch markiert.
          </p>

          {#if importError}
            <div class="p-4 rounded-xl bg-rose-50 border border-rose-100 text-rose-650 text-xs font-semibold animate-slide-up flex items-center space-x-2">
              <span>⚠️</span>
              <span>{importError}</span>
            </div>
          {/if}

          {#if importLoading}
            <div class="py-12 flex flex-col items-center justify-center space-y-4">
              <div class="w-10 h-10 border-4 border-t-blue-600 border-slate-200/50 rounded-full animate-spin"></div>
              <p class="text-xs text-slate-500 font-semibold">Schülerdaten werden verarbeitet, bitte warten...</p>
            </div>
          {:else}
            <!-- Drag and drop / file input -->
            <label class="border-2 border-dashed border-slate-200 hover:border-blue-500/80 hover:bg-slate-50/30 transition-all rounded-3xl p-8 flex flex-col items-center justify-center space-y-3 cursor-pointer text-center select-none">
              <input type="file" accept=".csv" class="sr-only" onchange={handleFileChange} />
              <span class="text-4xl">📂</span>
              {#if selectedFile}
                <div class="space-y-1">
                  <p class="text-xs font-bold text-slate-700 max-w-[280px] truncate">{selectedFile.name}</p>
                  <p class="text-[10px] text-slate-400 font-mono">{(selectedFile.size / 1024).toFixed(1)} KB</p>
                </div>
              {:else}
                <div class="space-y-1">
                  <p class="text-xs font-bold text-slate-600">CSV-Datei auswählen oder reinziehen</p>
                  <p class="text-[10px] text-slate-400 font-medium">Unterstützt Komma & Semikolon Trennung</p>
                </div>
              {/if}
            </label>

            <div class="flex justify-end gap-3 pt-2 text-xs">
              <button onclick={closeImportModal} class="px-4 py-2.5 rounded-xl bg-slate-100 text-slate-650 hover:bg-slate-200 transition-colors cursor-pointer font-bold">
                Abbrechen
              </button>
              <button onclick={startImport} disabled={!selectedFile} class="px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold transition-all shadow-xs cursor-pointer">
                Import starten
              </button>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}
