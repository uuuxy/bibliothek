<script>
  import { appState, showToast } from "../inventur/lib/store.svelte.js";
  import { apiFetch, apiClient } from "./apiFetch.js";

  /** @type {{ exemplare: any[], book: any, loadAll: (id: string) => void }} */
  let { exemplare = $bindable([]), book, loadAll } = $props();

  let selectedExemplare = $state(new Set());

  /** @type {string | null} */
  let editingExemplarId = $state(null);
  let editBarcodeValue = $state("");
  let barcodeError = $state("");

  /** @type {string | null} */
  let editingStatusId = $state(null);
  let editStatusType = $state("Verfügbar");
  let editStatusNote = $state("");
  let statusError = $state("");

  /** @param {any} ex */
  async function saveBarcode(ex) {
    if (!editBarcodeValue.trim()) return;
    if (editBarcodeValue.trim() === ex.barcode_id) {
      editingExemplarId = null;
      return;
    }
    barcodeError = "";
    try {
      const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/barcode`, { barcode: editBarcodeValue.trim()
      });
      if (res.ok) {
        ex.barcode_id = editBarcodeValue.trim();
        editingExemplarId = null;
      } else {
        const errorData = await res.json().catch(() => ({}));
        barcodeError = errorData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      barcodeError = "Netzwerkfehler";
    }
  }

  /** @param {any} ex */
  function openStatusEdit(ex) {
    editingStatusId = ex.id;
    if (ex.ist_ausleihbar) {
      editStatusType = "Verfügbar";
    } else if (ex.ist_ausgesondert || (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes("verloren"))) {
      editStatusType = "Verloren";
    } else {
      editStatusType = "Gesperrt (Defekt/Reserviert)";
    }
    editStatusNote = ex.zustand_notiz || "";
    statusError = "";
  }

  /** @param {any} ex */
  async function saveStatus(ex) {
    statusError = "";
    try {
      const isAusleihbar = editStatusType === "Verfügbar";
      const isAusgesondert = editStatusType === "Verloren" ? true : false;
      const notiz = isAusleihbar ? "" : editStatusNote.trim();
      const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/status`, { 
          ist_ausleihbar: isAusleihbar,
          ist_ausgesondert: isAusgesondert,
          zustand_notiz: notiz
        });
      if (res.ok) {
        ex.ist_ausleihbar = isAusleihbar;
        ex.ist_ausgesondert = isAusgesondert;
        ex.zustand_notiz = notiz;
        editingStatusId = null;
        showToast("Status erfolgreich gespeichert", "success");
      } else {
        const errData = await res.json().catch(() => ({}));
        statusError = errData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      statusError = "Netzwerkfehler";
    }
  }

  /** @param {any} ex */
  async function deleteCopy(ex) {
    if (!confirm(`Möchtest du das Exemplar ${ex.barcode_id} wirklich unwiderruflich löschen?`)) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, { method: "DELETE", credentials: "include" });
      if (res.ok) {
        exemplare = exemplare.filter((e) => e.id !== ex.id);
        if (book) {
          book.gesamt = Math.max(0, (book.gesamt || 0) - 1);
          if (book.verfuegbar !== undefined && ex.ist_ausleihbar) {
             book.verfuegbar = Math.max(0, (book.verfuegbar || 0) - 1);
          }
        }
        showToast("Exemplar erfolgreich gelöscht", "success");
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Löschen des Exemplars.");
      }
    } catch (e) {
      alert("Netzwerkfehler beim Löschen.");
    }
  }

  async function deleteSelectedCopies() {
    if (selectedExemplare.size === 0) return;
    if (!confirm(`Möchtest du die ${selectedExemplare.size} ausgewählten Exemplare unwiderruflich löschen?`)) return;
    
    let successCount = 0;

    const results = await Promise.allSettled(
      Array.from(selectedExemplare).map(async (id) => {
        const res = await apiFetch(`/api/buecher/exemplare/${id}`, { method: "DELETE", credentials: "include" });
        if (!res.ok) throw new Error("not ok");
        return id;
      })
    );

    for (const result of results) {
      if (result.status === 'fulfilled') {
        const id = result.value;
        exemplare = exemplare.filter((e) => e.id !== id);
        successCount++;
      } else {
        console.error("Fehler beim Löschen:", result.reason);
      }
    }
    selectedExemplare.clear();
    if (successCount > 0) {
      showToast(`${successCount} Exemplare erfolgreich gelöscht`, "success");
      if (book && book.id) loadAll(book.id);
    }
  }
</script>

{#if exemplare.length === 0}
  <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
    <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
    <p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
  </div>
{:else}
  {#if selectedExemplare.size > 0 && appState.adminAuthenticated}
    <div class="mb-4 p-3 bg-rose-50 border border-rose-100 rounded-xl flex items-center justify-between animate-fade-in">
      <span class="text-sm font-semibold text-rose-800">{selectedExemplare.size} Exemplare ausgewählt</span>
      <button class="px-3 py-1.5 bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold rounded-lg shadow-sm transition-colors cursor-pointer" onclick={deleteSelectedCopies}>
        Ausgewählte löschen
      </button>
    </div>
  {/if}
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
    {#each exemplare as ex}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="bg-white rounded-xl border p-4 shadow-sm transition-colors cursor-pointer {selectedExemplare.has(ex.id) ? 'border-blue-500 bg-blue-50/30 ring-1 ring-blue-500' : 'border-slate-200 hover:border-slate-300'}"
           onclick={() => {
             if (!appState.adminAuthenticated) return;
             if (selectedExemplare.has(ex.id)) {
               const newSet = new Set(selectedExemplare);
               newSet.delete(ex.id);
               selectedExemplare = newSet;
             } else {
               selectedExemplare = new Set(selectedExemplare).add(ex.id);
             }
           }}>
        <div class="flex items-start justify-between mb-3">
          {#if editingExemplarId === ex.id}
            <div class="flex-1 mr-2 relative">
              <!-- svelte-ignore a11y_autofocus -->
              <input
                type="text"
                bind:value={editBarcodeValue}
                autofocus
                onfocus={(e) => e.currentTarget.select()}
                class="w-full px-2 py-1 text-xs font-mono border {barcodeError ? 'border-rose-500 bg-rose-50 text-rose-700' : 'border-blue-300'} rounded focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                onkeydown={(e) => {
                  if (e.key === 'Enter') saveBarcode(ex);
                  if (e.key === 'Escape') { editingExemplarId = null; barcodeError = ""; }
                }}
              />
              {#if barcodeError}
                <p class="text-[10px] text-rose-600 mt-1 absolute -bottom-4 left-0 truncate w-full" title={barcodeError}>{barcodeError}</p>
              {/if}
            </div>
          {:else}
            <div class="flex items-center gap-3">
              {#if appState.adminAuthenticated}
                <input type="checkbox"
                       checked={selectedExemplare.has(ex.id)}
                       class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 cursor-pointer pointer-events-none"
                />
              {/if}
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold {ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-') ? 'text-amber-700 bg-amber-50 border-amber-100' : 'text-blue-700 bg-blue-50 border-blue-100'} border px-2 py-0.5 rounded font-mono">
                  {ex.barcode_id}
                </span>
                {#if appState.adminAuthenticated}
                  {#if ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-')}
                    <button
                      class="text-xs px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-800 font-semibold rounded shadow-sm transition-colors cursor-pointer flex items-center gap-1"
                      onclick={(e) => { 
                        e.stopPropagation(); 
                        editingExemplarId = ex.id; 
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
                        editingExemplarId = ex.id; 
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
            {#if editingStatusId !== ex.id}
              <button title="Status ändern" class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer" onclick={() => openStatusEdit(ex)}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
              </button>
              <button title="Exemplar löschen" class="text-slate-400 hover:text-rose-600 transition-colors cursor-pointer" onclick={(e) => { e.stopPropagation(); deleteCopy(ex); }}>
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
              </button>
            {/if}
          </div>
        </div>
        {#if editingStatusId === ex.id}
          <div class="mt-2 bg-slate-50 p-3 rounded-lg border border-slate-200">
            <div class="flex items-center gap-2 mb-2">
              <select bind:value={editStatusType} class="text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30">
                <option value="Verfügbar">Verfügbar</option>
                <option value="Gesperrt (Defekt/Reserviert)">Gesperrt (Defekt/Reserviert)</option>
                <option value="Verloren">Verloren</option>
              </select>
            </div>
            {#if editStatusType !== "Verfügbar"}
              <div class="mb-2">
                <input type="text" bind:value={editStatusNote} placeholder="Notiz (optional)" class="w-full text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30" onkeydown={(e) => { if (e.key === 'Enter') saveStatus(ex); if (e.key === 'Escape') { editingStatusId = null; statusError = ''; } }} />
              </div>
            {/if}
            <div class="flex items-center justify-between">
              <button onclick={() => { editingStatusId = null; statusError = ''; }} class="text-[10px] text-slate-500 hover:text-slate-700 font-semibold cursor-pointer">Abbrechen</button>
              <button onclick={() => saveStatus(ex)} class="text-[10px] bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded font-semibold cursor-pointer">Speichern</button>
            </div>
            {#if statusError}
              <p class="text-[10px] text-rose-600 mt-1">{statusError}</p>
            {/if}
          </div>
        {:else if ex.zustand_notiz}
          <p class="text-xs text-slate-500"><span class="font-semibold text-slate-400">Zustand:</span> {ex.zustand_notiz}</p>
        {/if}
      </div>
    {/each}
  </div>
{/if}
