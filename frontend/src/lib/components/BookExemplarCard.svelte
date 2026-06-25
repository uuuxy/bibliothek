<script>
  import { appState } from "../../inventur/lib/store.svelte.js";
  import { apiFetch, apiClient } from "../apiFetch.js";
  import BookExemplarStatusEditor from "./BookExemplarStatusEditor.svelte";

  /**
   * Einzelne Exemplar-Karte. Verwaltet ihren eigenen Bearbeitungsmodus
   * (Barcode/Status) lokal; Auswahl & Löschen laufen über Callbacks zum Eltern-Tab.
   * @type {{ ex: any, selected: boolean, onToggleSelect: () => void, onDelete: () => void }}
   */
  let { ex, selected, onToggleSelect, onDelete } = $props();

  let editingBarcode = $state(false);
  let editBarcodeValue = $state("");
  let barcodeError = $state("");

  let editingStatus = $state(false);

  async function saveBarcode() {
    if (!editBarcodeValue.trim()) return;
    if (editBarcodeValue.trim() === ex.barcode_id) {
      editingBarcode = false;
      return;
    }
    barcodeError = "";
    try {
      const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/barcode`, { barcode: editBarcodeValue.trim()
      });
      if (res.ok) {
        ex.barcode_id = editBarcodeValue.trim();
        editingBarcode = false;
      } else {
        const errorData = await res.json().catch(() => ({}));
        barcodeError = errorData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      barcodeError = "Netzwerkfehler";
    }
  }

  async function generateInternalId() {
    try {
      const res = await apiFetch("/api/barcode/next");
      if (res.ok) {
        const data = await res.json();
        editBarcodeValue = data.next_barcode;
      } else {
        barcodeError = "Fehler beim Generieren der ID";
      }
    } catch(e) {
      barcodeError = "Netzwerkfehler";
    }
  }

</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="bg-white rounded-xl border p-4 shadow-sm transition-colors cursor-pointer {selected ? 'border-blue-500 bg-blue-50/30 ring-1 ring-blue-500' : 'border-slate-200 hover:border-slate-300'}"
     onclick={() => { if (appState.adminAuthenticated) onToggleSelect(); }}>
  <div class="flex items-start justify-between mb-3">
    {#if editingBarcode}
      <div class="flex-1 mr-2 relative">
        <!-- svelte-ignore a11y_autofocus -->
        <input
          type="text"
          bind:value={editBarcodeValue}
          autofocus
          onfocus={(e) => e.currentTarget.select()}
          class="w-full px-2 py-1 text-xs font-mono border {barcodeError ? 'border-rose-500 bg-rose-50 text-rose-700' : 'border-blue-300'} rounded focus:outline-none focus:ring-2 focus:ring-blue-500/30"
          onkeydown={(e) => {
            if (e.key === 'Enter') saveBarcode();
            if (e.key === 'Escape') { editingBarcode = false; barcodeError = ""; }
          }}
        />
        <div class="mt-1 flex gap-2">
          <button onclick={generateInternalId} class="text-[10px] bg-slate-100 hover:bg-slate-200 text-slate-700 px-2 py-0.5 rounded font-semibold cursor-pointer">Interne ID generieren</button>
          <button onclick={saveBarcode} class="text-[10px] bg-blue-600 hover:bg-blue-700 text-white px-2 py-0.5 rounded font-semibold cursor-pointer">Speichern</button>
        </div>
        {#if barcodeError}
          <p class="text-[10px] text-rose-600 mt-1 absolute -bottom-4 left-0 truncate w-full" title={barcodeError}>{barcodeError}</p>
        {/if}
      </div>
    {:else}
      <div class="flex items-center gap-3">
        {#if appState.adminAuthenticated}
          <input type="checkbox"
                 checked={selected}
                 class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 cursor-pointer pointer-events-none"
          />
        {/if}
        <div class="flex items-center gap-2">
          <span class="text-xs font-bold {ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-') ? 'text-amber-700 bg-amber-50 border-amber-100' : 'text-blue-700 bg-blue-50 border-blue-100'} border px-2 py-0.5 rounded font-mono">
            {ex.barcode_id}
          </span>
          {#if appState.adminAuthenticated}
            {#if ex.barcode_id.startsWith('B-')}
              <a href={`/api/print/etikett/${ex.id}`} target="_blank" title="Ersatz-Etikett drucken" class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer flex items-center gap-1" onclick={(e) => e.stopPropagation()}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" /></svg>
              </a>
            {/if}
            {#if ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-')}
              <button
                class="text-xs px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-800 font-semibold rounded shadow-sm transition-colors cursor-pointer flex items-center gap-1"
                onclick={(e) => {
                  e.stopPropagation();
                  editingBarcode = true;
                  editBarcodeValue = ""; // Leer lassen für den Scanner
                  barcodeError = "";
                }}
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
                Barcode scannen
              </button>
            {:else}
              <button
                title="Barcode zuweisen/ändern"
                class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer flex items-center gap-1"
                onclick={(e) => {
                  e.stopPropagation();
                  editingBarcode = true;
                  editBarcodeValue = ex.barcode_id;
                  barcodeError = "";
                }}
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
              </button>
            {/if}
          {/if}
        </div>
      </div>
    {/if}
    <div class="flex items-center gap-2">
      <span class="text-[10px] font-bold px-2 py-0.5 rounded-full {!ex.ist_ausleihbar ? 'bg-rose-50 text-rose-700 border border-rose-100' : !ex.ist_verfuegbar ? 'bg-amber-50 text-amber-700 border border-amber-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
        {!ex.ist_ausleihbar ? "Gesperrt" : !ex.ist_verfuegbar ? "Ausgeliehen" : "Verfügbar"}
      </span>
      {#if !editingStatus}
        <button title="Status ändern" class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer" onclick={(e) => { e.stopPropagation(); editingStatus = true; }}>
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
        </button>
        <button title="Exemplar löschen" class="text-slate-400 hover:text-rose-600 transition-colors cursor-pointer" onclick={(e) => { e.stopPropagation(); onDelete(); }}>
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
        </button>
      {/if}
    </div>
  </div>
  {#if editingStatus}
    <BookExemplarStatusEditor {ex} onDone={() => editingStatus = false} />
  {:else if ex.zustand_notiz}
    <p class="text-xs text-slate-500"><span class="font-semibold text-slate-400">Zustand:</span> {ex.zustand_notiz}</p>
  {/if}
</div>
