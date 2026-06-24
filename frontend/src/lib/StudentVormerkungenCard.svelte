<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { showToast } from "../inventur/lib/store.svelte.js";

  /** @type {{ vormerkungen: any[] }} */
  let { vormerkungen = $bindable() } = $props();

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
</script>

<div class="w-full h-full pt-2">
  <div class="flex items-center justify-between pb-3 border-b border-slate-100 mb-6">
    <h3 class="text-base font-bold text-slate-500 uppercase tracking-wider">Vorgemerkte Bücher ({vormerkungen?.length || 0})</h3>
  </div>

  {#if !vormerkungen || vormerkungen.length === 0}
    <div class="py-16 flex flex-col items-center justify-center text-slate-500 space-y-3">
      <svg class="h-12 w-12 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <span class="text-sm font-semibold text-slate-400">Aktuell keine Bücher vorgemerkt.</span>
    </div>
  {:else}
    <div class="space-y-4">
      {#each vormerkungen as v}
        <div class="border-b border-gray-200 py-4 flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <h4 class="font-bold text-slate-800">{v.titel_name || "Unbekannter Titel"}</h4>
            <div class="flex items-center gap-2 text-xs font-semibold text-slate-500">
              <span class="px-2 py-0.5 rounded-md bg-blue-50 text-blue-700">Wartet seit: {new Date(v.erstellt_am).toLocaleDateString('de-DE')}</span>
            </div>
            {#if v.notiz}
              <p class="text-sm text-slate-600 mt-1 italic">Notiz: {v.notiz}</p>
            {/if}
          </div>
          <button
            onclick={() => deleteVormerkung(v.id)}
            class="text-rose-600 hover:text-rose-700 hover:bg-rose-50 p-2 rounded-lg transition-colors cursor-pointer"
            title="Vormerkung löschen"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>
