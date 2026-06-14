<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import Modal from "./Modal.svelte";

  let { open = false, file = null, onclose, onsuccess } = $props();

  let loading = $state(false);
  let error = $state(/** @type {string|null} */ (null));
  let stats = $state(/** @type {any} */ (null));

  // When opened with a file, immediately fetch the preview
  $effect(() => {
    if (open && file && !stats && !loading && !error) {
      loadPreview(file);
    }
  });

  async function loadPreview(/** @type {File} */ f) {
    loading = true;
    error = null;
    stats = null;
    try {
      const fd = new FormData();
      fd.append("csvFile", f);
      const res = await apiFetch("/api/lusd/preview", {
        method: "POST",
        body: fd
      });
      if (!res.ok) throw new Error(await res.text() || "Vorschau fehlgeschlagen.");
      stats = await res.json();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  async function confirmImport() {
    if (!file) return;
    loading = true;
    error = null;
    try {
      const fd = new FormData();
      fd.append("csvFile", file);
      const res = await apiFetch("/api/lusd/import", {
        method: "POST",
        body: fd
      });
      if (!res.ok) throw new Error(await res.text() || "Import fehlgeschlagen.");
      const data = await res.json();
      onsuccess(data);
    } catch (e) {
      error = String(e);
      loading = false;
    }
  }

  function handleClose() {
    stats = null;
    error = null;
    onclose();
  }
</script>

<Modal {open} onclose={handleClose} size="md">
  {#snippet header()}
    <h3 class="text-sm font-bold text-slate-800">LUSD-Import: Vorschau & Bestätigung</h3>
  {/snippet}
  {#snippet children()}
    <div class="p-6 space-y-6">
      {#if loading}
        <div class="py-12 flex flex-col items-center justify-center space-y-4">
          <div class="w-8 h-8 border-4 border-t-blue-600 border-slate-200 rounded-full animate-spin"></div>
          <p class="text-sm font-semibold text-slate-500">CSV wird analysiert... Bitte warten.</p>
        </div>
      {:else if error}
        <div class="p-4 bg-rose-50 border border-rose-100 rounded-2xl text-rose-700 text-sm font-semibold">
          <p class="mb-2 font-bold flex items-center gap-2">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
            Import-Fehler
          </p>
          <p>{error}</p>
        </div>
      {:else if stats}
        <div class="space-y-4">
          <p class="text-sm text-slate-600 font-medium">Die CSV-Datei ({file?.name}) wurde analysiert. Bevor Änderungen an der Datenbank durchgeführt werden, überprüfe bitte die folgenden Auswirkungen:</p>
          
          <div class="grid grid-cols-2 gap-3">
            <div class="p-4 bg-slate-50 border border-slate-100 rounded-2xl text-center">
              <p class="text-2xl font-bold text-slate-800">{stats.total_csv_records}</p>
              <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mt-1">Gültige Zeilen in CSV</p>
            </div>
            <div class="p-4 bg-blue-50 border border-blue-100 rounded-2xl text-center">
              <p class="text-2xl font-bold text-blue-700">{stats.new_students}</p>
              <p class="text-xs font-semibold text-blue-500 uppercase tracking-wider mt-1">Neue Schüler (Anlegen)</p>
            </div>
            <div class="p-4 bg-amber-50 border border-amber-100 rounded-2xl text-center">
              <p class="text-2xl font-bold text-amber-700">{stats.class_changes}</p>
              <p class="text-xs font-semibold text-amber-600 uppercase tracking-wider mt-1">Klassenwechsel (Update)</p>
            </div>
            <div class="p-4 bg-rose-50 border border-rose-100 rounded-2xl text-center">
              <p class="text-2xl font-bold text-rose-700">{stats.graduates}</p>
              <p class="text-xs font-semibold text-rose-600 uppercase tracking-wider mt-1">Abgänger (Sperren/Löschen)</p>
            </div>
          </div>

          <div class="p-4 bg-blue-50/50 border border-blue-100 rounded-2xl text-xs font-semibold text-blue-800 leading-relaxed">
            <strong>Hinweis zu Abgängern:</strong> Schüler, die nicht mehr in der CSV-Datei aufgeführt sind, werden als Abgänger markiert. Wenn sie noch Bücher ausgeliehen haben, wird ihr Profil nur gesperrt. Andernfalls wird ihr Profil DSGVO-konform anonymisiert.
          </div>
        </div>
      {/if}

      <div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
        <button onclick={handleClose} disabled={loading} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer">Abbrechen</button>
        {#if stats && !error && !loading}
          <button onclick={confirmImport} class="rounded-xl bg-blue-600 px-4 py-2 text-sm font-bold text-white hover:bg-blue-750 transition-colors cursor-pointer flex items-center gap-2 shadow-sm">
            Bestätigen & Importieren
          </button>
        {/if}
      </div>
    </div>
  {/snippet}
</Modal>
