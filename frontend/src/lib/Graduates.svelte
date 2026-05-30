<script>
  import { onMount } from "svelte";

  // State Runes
  /** @type {any[]} */
  let graduates = $state([]);
  let loading = $state(true);

  // Laufzettel print state
  /** @type {any[]} */
  let detailStudents = $state([]);
  let loadingLaufzettel = $state(false);
  let printDate = $state("");

  async function printLaufzettel() {
    loadingLaufzettel = true;
    printDate = new Date().toLocaleDateString("de-DE");
    try {
      const res = await fetch("/api/abgaenger?details=true");
      if (res.ok) detailStudents = await res.json();
    } catch (err) {
      console.error("Laufzettel load error:", err);
    } finally {
      loadingLaufzettel = false;
    }
    // Allow Svelte to flush the reactive update before printing
    await new Promise(r => setTimeout(r, 150));
    window.print();
  }
  
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
  <div class="flex items-center justify-end border-b border-slate-100 pb-5">
    <div class="flex items-center space-x-4">
      <button
        onclick={printLaufzettel}
        disabled={loadingLaufzettel || graduates.length === 0}
        class="no-print px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer">
        {#if loadingLaufzettel}
          <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
          Lade Daten…
        {:else}
          🖨️ Laufzettel drucken
        {/if}
      </button>
      <button onclick={() => showImportModal = true} class="no-print px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer">
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

<!-- ═══════════════════════════════════════════════════════════════
     PRINT OUTPUT — Laufzettel (one A4 page per graduate, hidden on screen)
     ═══════════════════════════════════════════════════════════════ -->
<div class="hidden print:block">
  {#each detailStudents as student (student.id)}
    <div class="laufzettel-page">

      <!-- Page header ──────────────────────────────────────── -->
      <div style="display:flex; justify-content:space-between; align-items:flex-end; border-bottom:2pt solid #111; padding-bottom:6mm; margin-bottom:7mm;">
        <div>
          <div style="font-size:18pt; font-weight:900; letter-spacing:-0.3mm; line-height:1; color:#111;">Entlassungslaufzettel</div>
          <div style="font-size:9pt; color:#555; margin-top:2mm; font-weight:600;">Schulbibliothek</div>
        </div>
        <div style="text-align:right; font-size:8.5pt; color:#777;">
          <div>Erstellt am: {printDate}</div>
        </div>
      </div>

      <!-- Student info box ──────────────────────────────────── -->
      <div style="background:#f3f4f6; border:1pt solid #d1d5db; border-radius:4pt; padding:5mm 7mm; margin-bottom:7mm; display:flex; justify-content:space-between; align-items:center;">
        <div>
          <div style="font-size:15pt; font-weight:900; color:#111; line-height:1.2;">{student.vorname} {student.nachname}</div>
          <div style="font-size:9.5pt; color:#444; margin-top:1.5mm; font-weight:600;">Klasse: <span style="color:#1d4ed8;">{student.klasse}</span></div>
        </div>
        <div style="text-align:right;">
          <div style="font-size:7.5pt; color:#777; text-transform:uppercase; letter-spacing:0.4mm; font-weight:700;">Schüler-ID</div>
          <div style="font-size:10pt; font-family:monospace; font-weight:900; color:#111; margin-top:1mm;">{student.barcode_id}</div>
        </div>
      </div>

      <!-- Unreturned media table ────────────────────────────── -->
      <div style="font-size:8.5pt; font-weight:800; text-transform:uppercase; letter-spacing:0.5mm; color:#374151; margin-bottom:3mm;">
        Ausstehende Medien ({(student.ausleihen ?? []).length})
      </div>
      <table style="width:100%; border-collapse:collapse; margin-bottom:10mm; font-size:8pt;">
        <thead>
          <tr style="background:#e5e7eb; color:#374151;">
            <th style="width:14mm; padding:1.5mm 2mm;"></th>
            <th style="text-align:left; padding:2mm 3mm; font-weight:700; font-size:7.5pt; text-transform:uppercase; letter-spacing:0.3mm;">Titel</th>
            <th style="text-align:left; padding:2mm 3mm; font-weight:700; font-size:7.5pt; text-transform:uppercase; letter-spacing:0.3mm;">Autor</th>
            <th style="text-align:left; padding:2mm 3mm; font-weight:700; font-size:7.5pt; text-transform:uppercase; letter-spacing:0.3mm;">Barcode</th>
            <th style="text-align:left; padding:2mm 3mm; font-weight:700; font-size:7.5pt; text-transform:uppercase; letter-spacing:0.3mm;">Fällig am</th>
          </tr>
        </thead>
        <tbody>
          {#each (student.ausleihen ?? []) as loan, i}
            <tr style="border-bottom:0.5pt solid #e5e7eb; background:{i % 2 === 0 ? 'white' : '#f9fafb'};">
              <td style="padding:2mm 3mm; vertical-align:middle;">
                {#if loan.cover_url}
                  <img src={loan.cover_url} style="width:10mm; height:14mm; object-fit:cover; border-radius:1pt; border:0.5pt solid #e5e7eb; display:block;" alt="" />
                {:else}
                  <div style="width:10mm; height:14mm; background:#e5e7eb; border-radius:1pt; display:flex; align-items:center; justify-content:center;">
                    <span style="font-size:6pt; color:#9ca3af;">–</span>
                  </div>
                {/if}
              </td>
              <td style="padding:2mm 3mm; font-weight:600; color:#111; vertical-align:middle;">{loan.titel}</td>
              <td style="padding:2mm 3mm; color:#555; vertical-align:middle;">{loan.autor || '—'}</td>
              <td style="padding:2mm 3mm; font-family:monospace; font-size:7.5pt; color:#374151; vertical-align:middle;">{loan.barcode_id}</td>
              <td style="padding:2mm 3mm; color:{loan.frist ? '#111' : '#9ca3af'}; vertical-align:middle;">{loan.frist || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>

      <!-- Signature section ────────────────────────────────── -->
      <div style="border-top:1pt solid #d1d5db; padding-top:6mm; margin-top:auto;">
        <div style="font-size:8.5pt; color:#374151; margin-bottom:8mm; line-height:1.5;">
          Ich bestätige die vollständige und ordnungsgemäße Rückgabe aller ausgeliehenen Bibliotheksmedien.
        </div>
        <div style="display:flex; gap:8mm;">
          <div style="flex:1; min-width:0;">
            <div style="border-bottom:1pt solid #111; height:9mm; margin-bottom:2mm;"></div>
            <div style="font-size:7pt; color:#6b7280; font-weight:600;">Datum</div>
          </div>
          <div style="flex:2; min-width:0;">
            <div style="border-bottom:1pt solid #111; height:9mm; margin-bottom:2mm;"></div>
            <div style="font-size:7pt; color:#6b7280; font-weight:600;">Unterschrift Schüler/in</div>
          </div>
          <div style="flex:2; min-width:0;">
            <div style="border-bottom:1pt solid #111; height:9mm; margin-bottom:2mm;"></div>
            <div style="font-size:7pt; color:#6b7280; font-weight:600;">Unterschrift Schulbibliothek</div>
          </div>
        </div>
      </div>

    </div>
  {/each}
</div>
