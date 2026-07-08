<!-- @component LusdImportView — Preview-to-Commit Flow für den LUSD-Schuljahreswechsel-Import.
     Go-Handler sind zustandslos (keine Preview-Session), daher wird dieselbe Datei bei
     „Import finalisieren“ erneut an /api/lusd/import gesendet. -->
<script>
  import { apiFetch } from "../../apiFetch.js";
  import { toastStore } from "../../stores/toastStore.svelte.js";

  /** @typedef {{ id: string, vorname: string, nachname: string, alte_klasse?: string, neue_klasse?: string }} StudentDiff */
  /** @typedef {{ new_students: StudentDiff[], class_changes: StudentDiff[], graduates: StudentDiff[], total_csv_records: number }} LusdPreviewResult */

  /** @type {{ onImported?: (result: LusdPreviewResult) => void }} */
  let { onImported = () => {} } = $props();

  /** @type {File | null} */ let selectedFile = $state(null);
  /** @type {'upload' | 'preview' | 'done'} */ let stage = $state("upload");
  let previewLoading = $state(false);
  let importLoading = $state(false);
  /** @type {LusdPreviewResult | null} */ let previewResult = $state(null);
  /** @type {LusdPreviewResult | null} */ let importResult = $state(null);
  /** @type {string | null} */ let errorMessage = $state(null);

  const activeResult = $derived(stage === "done" ? importResult : previewResult);
  const summaryRows = $derived(
    activeResult
      ? [
          { key: "new", label: "Neue Schüler", hint: "Werden neu angelegt", items: activeResult.new_students, valueClass: "text-emerald-600" },
          { key: "changes", label: "Klassenwechsel", hint: "Bestehende Schüler mit geänderter Klasse", items: activeResult.class_changes, valueClass: "text-blue-600" },
          { key: "graduates", label: "Abgänger", hint: "Fehlen in der Datei — werden als Abgänger markiert", items: activeResult.graduates, valueClass: "text-rose-600" },
        ]
      : [],
  );

  const hasRiskyGraduates = $derived(!!previewResult && previewResult.total_csv_records > 0 && previewResult.graduates.length / previewResult.total_csv_records >= 0.3);

  function handleFileChange(/** @type {Event} */ e) {
    const target = /** @type {HTMLInputElement} */ (e.target);
    if (target.files && target.files[0]) {
      selectedFile = target.files[0];
      errorMessage = null;
      previewResult = null;
      stage = "upload";
    }
  }

  function resetFlow() {
    selectedFile = null;
    previewResult = null;
    importResult = null;
    errorMessage = null;
    stage = "upload";
  }

  /** Sendet die aktuell gewählte Datei an einen der beiden LUSD-Endpoints. @param {string} url */
  async function submitLusdFile(url) {
    const formData = new FormData();
    formData.append("csvFile", /** @type {File} */ (selectedFile));

    const res = await apiFetch(url, { method: "POST", body: formData });
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new Error(data?.error || "Fehler bei der Verarbeitung der LUSD-Datei.");
    }
    return res.json();
  }

  async function runPreview() {
    if (!selectedFile || previewLoading) return;
    previewLoading = true;
    errorMessage = null;
    try {
      previewResult = await submitLusdFile("/api/lusd/preview");
      stage = "preview";
    } catch (err) {
      errorMessage = /** @type {any} */ (err).message || String(err);
      toastStore.addToast(errorMessage, "error");
    } finally {
      previewLoading = false;
    }
  }

  async function finalizeImport() {
    if (!selectedFile || importLoading) return;
    importLoading = true;
    errorMessage = null;
    try {
      importResult = await submitLusdFile("/api/lusd/import");
      stage = "done";
      toastStore.addToast("LUSD-Import erfolgreich übernommen.", "success");
      onImported(/** @type {LusdPreviewResult} */ (importResult));
    } catch (err) {
      errorMessage = /** @type {any} */ (err).message || String(err);
      toastStore.addToast(errorMessage, "error");
    } finally {
      importLoading = false;
    }
  }
</script>

{#snippet diffSection(section)}
  <details class="group py-1">
    <summary class="flex items-center justify-between py-3 cursor-pointer select-none marker:content-none [&::-webkit-details-marker]:hidden">
      <div class="min-w-0 flex items-center gap-2">
        <svg class="w-3 h-3 text-slate-400 shrink-0 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" /></svg>
        <div class="min-w-0"><p class="text-sm font-bold text-slate-800">{section.label}</p><p class="text-xs text-slate-450 mt-0.5">{section.hint}</p></div>
      </div>
      <span class="text-lg font-black tabular-nums shrink-0 ml-4 {section.valueClass}">{section.items.length}</span>
    </summary>
    <ul class="divide-y divide-slate-50 pb-2">
      {#each section.items as item (item.id)}
        <li class="py-2 pl-5 flex items-center justify-between gap-3 text-xs">
          <span class="font-semibold text-slate-700 truncate">{item.vorname} {item.nachname}</span><span class="text-slate-400 font-mono shrink-0">{item.alte_klasse && item.neue_klasse ? `${item.alte_klasse} → ${item.neue_klasse}` : item.neue_klasse || item.alte_klasse || "—"}</span>
        </li>
      {/each}
    </ul>
  </details>
{/snippet}

<div class="w-full max-w-2xl space-y-8">
  <div>
    <h2 class="text-base font-bold text-slate-900">LUSD-Import</h2>
    <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-xl">Lade die LUSD-Exportdatei hoch, um die Änderungen zu prüfen, bevor sie verbindlich in die Datenbank übernommen werden. Kein Datensatz wird ohne Bestätigung überschrieben.</p>
  </div>

  {#if errorMessage}
    <div class="p-4 rounded-xl bg-rose-50 border border-rose-100 text-rose-650 text-xs font-semibold flex items-center gap-2"><span>⚠️</span><span>{errorMessage}</span></div>
  {/if}

  {#if stage === "done"}
    <div class="p-4 rounded-xl bg-emerald-50 border border-emerald-100 text-emerald-800 text-sm font-semibold flex items-center gap-2"><span>🎉</span><span>Import abgeschlossen — der Bestand ist aktuell.</span></div>
    <div class="divide-y divide-slate-100">
      {#each summaryRows as section (section.key)}
        {@render diffSection(section)}
      {/each}
    </div>
    <button onclick={resetFlow} class="px-5 py-2.5 rounded-full bg-slate-900 hover:bg-slate-800 text-white text-xs font-bold transition-colors cursor-pointer">
      Weitere Datei importieren
    </button>
  {:else}
    <label class="border-2 border-dashed border-slate-200 hover:border-blue-500/70 hover:bg-slate-50/40 transition-all rounded-2xl p-8 flex flex-col items-center justify-center gap-3 cursor-pointer text-center select-none">
      <input type="file" accept=".csv" class="sr-only" onchange={handleFileChange} disabled={previewLoading || importLoading} />
      <span class="text-3xl">📂</span>
      {#if selectedFile}
        <div class="space-y-1">
          <p class="text-xs font-bold text-slate-700 max-w-xs truncate">{selectedFile.name}</p>
          <p class="text-[10px] text-slate-400">{(selectedFile.size / 1024).toFixed(1)} KB</p>
        </div>
      {:else}
        <div class="space-y-1">
          <p class="text-xs font-bold text-slate-600">LUSD-CSV auswählen oder reinziehen</p>
          <p class="text-[10px] text-slate-400 font-medium">Unterstützt Komma- &amp; Semikolon-Trennung</p>
        </div>
      {/if}
    </label>

    {#if stage === "upload"}
      <div class="flex justify-end">
        <button onclick={runPreview} disabled={!selectedFile || previewLoading} class="px-5 py-2.5 rounded-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-xs font-bold transition-all cursor-pointer flex items-center gap-2">
          {#if previewLoading}
            <span class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"></span> Vorschau wird geladen…
          {:else}
            Vorschau laden
          {/if}
        </button>
      </div>
    {/if}

    {#if stage === "preview" && previewResult}
      <div class="space-y-4">
        <p class="text-xs text-slate-450">{previewResult.total_csv_records} Datensätze in der Datei gefunden.</p>
        {#if hasRiskyGraduates}
          <div class="p-4 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-semibold flex items-center gap-2"><span>⚠️</span><span>Auffällig viele Abgänger ({previewResult.graduates.length} von {previewResult.total_csv_records}) — Datei vor dem Import genau prüfen, gerade beim Schuljahreswechsel.</span></div>
        {/if}

        <div class="divide-y divide-slate-100">
          {#each summaryRows as section (section.key)}
            {@render diffSection(section)}
          {/each}
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button onclick={resetFlow} class="px-4 py-2.5 rounded-full bg-slate-100 hover:bg-slate-200 text-slate-650 text-xs font-bold transition-colors cursor-pointer">
            Andere Datei wählen
          </button>
          <button onclick={finalizeImport} disabled={importLoading} class="px-5 py-2.5 rounded-full bg-slate-900 hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed text-white text-xs font-bold transition-all cursor-pointer flex items-center gap-2">
            {#if importLoading}
              <span class="w-3.5 h-3.5 border-2 border-white/60 border-t-white rounded-full animate-spin"></span> Import wird übernommen…
            {:else}
              Import finalisieren
            {/if}
          </button>
        </div>
      </div>
    {/if}
  {/if}
</div>
