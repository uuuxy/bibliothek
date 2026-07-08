<script>
  import { showToast } from "../../inventur/lib/store.svelte.js";
  import { apiClient } from "../apiFetch.js";

  /**
   * Status-Editor eines Exemplars (Verfügbar / Gesperrt / Verloren).
   * Initialisiert sich aus ex; speichert in-place und schließt via onDone.
   * @type {{ ex: any, onDone: () => void }}
   */
  let { ex, onDone } = $props();

  let editStatusType = $state("");
  let editStatusNote = $state("");

  $effect(() => {
    editStatusType = ex.ist_ausleihbar
      ? "Verfügbar"
      : (ex.ist_ausgesondert || (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes("verloren")))
        ? "Verloren"
        : "Gesperrt (Defekt/Reserviert)";
    editStatusNote = ex.zustand_notiz || "";
  });
  let statusError = $state("");

  async function saveStatus() {
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
        onDone();
        showToast("Status erfolgreich gespeichert", "success");
      } else {
        const errData = await res.json().catch(() => ({}));
        statusError = errData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      statusError = "Netzwerkfehler";
    }
  }
</script>

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
      <input type="text" bind:value={editStatusNote} placeholder="Notiz (optional)" class="w-full text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30" onkeydown={(e) => { if (e.key === 'Enter') saveStatus(); if (e.key === 'Escape') onDone(); }} />
    </div>
  {/if}
  <div class="flex items-center justify-between">
    <button onclick={onDone} class="text-[10px] text-slate-500 hover:text-slate-700 font-semibold cursor-pointer">Abbrechen</button>
    <button onclick={saveStatus} class="text-[10px] bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded font-semibold cursor-pointer">Speichern</button>
  </div>
  {#if statusError}
    <p class="text-[10px] text-rose-600 mt-1">{statusError}</p>
  {/if}
</div>
