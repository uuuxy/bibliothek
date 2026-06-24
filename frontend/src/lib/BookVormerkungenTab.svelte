<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { showToast } from "../inventur/lib/store.svelte.js";

  /** @type {{ vormerkungen: any[], book: any }} */
  let { vormerkungen = $bindable(), book } = $props();

  let isAdding = $state(false);
  let searchVal = $state("");
  let searchResults = $state.raw(/** @type {any[]} */ ([]));
  let isSearching = $state(false);
  let notiz = $state("");

  async function deleteVormerkung(id) {
    if (!confirm("Vormerkung wirklich löschen?")) return;
    try {
      const res = await apiFetch(`/api/vormerkungen/${id}`, { method: "DELETE" });
      if (res.ok) {
        vormerkungen = vormerkungen.filter(v => v.id !== id);
        showToast("Vormerkung gelöscht", "success");
      } else {
        const err = await res.json().catch(() => ({}));
        showToast(err.error || "Fehler beim Löschen", "error");
      }
    } catch (e) {
      showToast("Netzwerkfehler", "error");
    }
  }

  async function searchStudent() {
    if (!searchVal.trim()) {
      searchResults = [];
      return;
    }
    isSearching = true;
    try {
      const res = await apiClient.post("/api/action", { query: searchVal });
      if (res.ok) {
        const data = await res.json();
        searchResults = data.search_results?.filter(r => r.type === "student") || [];
      } else {
        searchResults = [];
      }
    } catch {
      searchResults = [];
    } finally {
      isSearching = false;
    }
  }

  async function addVormerkung(studentId) {
    try {
      const res = await apiClient.post("/api/vormerkungen", { titel_id: book.id, schueler_id: studentId, notiz });
      if (res.ok) {
        showToast("Erfolgreich vorgemerkt", "success");
        isAdding = false;
        searchVal = "";
        searchResults = [];
        notiz = "";
        // Reload list
        const listRes = await apiFetch(`/api/vormerkungen?titel_id=${book.id}`);
        if (listRes.ok) {
          vormerkungen = await listRes.json();
        }
      } else {
        const err = await res.json().catch(() => ({}));
        showToast(err.error || "Fehler beim Hinzufügen", "error");
      }
    } catch (e) {
      showToast("Netzwerkfehler", "error");
    }
  }
</script>

<div class="space-y-6 pt-4">
  <div class="flex items-center justify-between">
    <h3 class="text-lg font-bold text-slate-800">Warteliste / Vormerkungen</h3>
    <button
      onclick={() => isAdding = !isAdding}
      class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold rounded-xl transition-colors shadow-sm"
    >
      {isAdding ? "Abbrechen" : "+ Schüler vormerken"}
    </button>
  </div>

  {#if isAdding}
    <div class="p-5 bg-slate-50 border border-slate-200 rounded-2xl space-y-4 animate-fade-in">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label for="student-search-input" class="block text-sm font-medium text-gray-600 mb-2">Schüler suchen (Name oder Barcode)</label>
          <div class="flex gap-2">
            <input
              id="student-search-input"
              type="text"
              bind:value={searchVal}
              onkeydown={(e) => e.key === 'Enter' && searchStudent()}
              placeholder="z.B. Max Mustermann"
              class="flex-1 bg-white border border-slate-200 rounded-xl px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
            />
            <button
              onclick={searchStudent}
              disabled={isSearching}
              class="px-4 py-2 bg-white border border-slate-200 text-slate-700 hover:bg-slate-100 rounded-xl text-sm font-semibold transition-colors disabled:opacity-50"
            >
              {isSearching ? "..." : "Suchen"}
            </button>
          </div>
        </div>
        <div>
          <label for="notiz-input" class="block text-sm font-medium text-gray-600 mb-2">Interne Notiz (optional)</label>
          <input
            id="notiz-input"
            type="text"
            bind:value={notiz}
            placeholder="z.B. Braucht es dringend für Referat"
            class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
          />
        </div>
      </div>

      {#if searchResults.length > 0}
        <div class="mt-4 border border-slate-200 rounded-xl overflow-hidden bg-white">
          {#each searchResults as r}
            <div class="flex items-center justify-between p-3 border-b border-slate-100 last:border-0 hover:bg-slate-50 transition-colors">
              <div>
                <p class="font-semibold text-slate-800 text-sm">{r.title}</p>
                <p class="text-xs text-slate-500">{r.subtitle}</p>
              </div>
              <button
                onclick={() => addVormerkung(r.id)}
                class="px-3 py-1.5 bg-blue-50 text-blue-700 hover:bg-blue-100 rounded-lg text-xs font-bold transition-colors"
              >
                Auswählen
              </button>
            </div>
          {/each}
        </div>
      {:else if searchVal && !isSearching && searchResults.length === 0}
        <p class="text-sm text-slate-500 mt-2">Keine Schüler gefunden.</p>
      {/if}
    </div>
  {/if}

  {#if vormerkungen.length === 0}
    <div class="py-12 flex flex-col items-center text-slate-400 gap-3 border-2 border-dashed border-slate-200 rounded-2xl bg-slate-50/50">
      <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <p class="font-medium text-sm">Keine ausstehenden Vormerkungen für diesen Titel.</p>
    </div>
  {:else}
    <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <table class="w-full text-left text-sm whitespace-nowrap">
        <thead class="bg-slate-50 text-slate-500 text-xs uppercase tracking-wider font-bold">
          <tr>
            <th class="px-6 py-4">Wartet seit</th>
            <th class="px-6 py-4">Schüler</th>
            <th class="px-6 py-4">Notiz</th>
            <th class="px-6 py-4 text-right">Aktion</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          {#each vormerkungen as v}
            <tr class="hover:bg-slate-50/50 transition-colors">
              <td class="px-6 py-4 font-medium text-slate-800">
                {new Date(v.erstellt_am).toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })}
              </td>
              <td class="px-6 py-4 font-semibold text-blue-600">
                {v.schueler_name || "Unbekannt"}
              </td>
              <td class="px-6 py-4 text-slate-500">
                {v.notiz || "—"}
              </td>
              <td class="px-6 py-4 text-right">
                <button
                  onclick={() => deleteVormerkung(v.id)}
                  class="text-rose-600 hover:text-rose-700 font-semibold p-2 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer"
                  title="Vormerkung löschen"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
