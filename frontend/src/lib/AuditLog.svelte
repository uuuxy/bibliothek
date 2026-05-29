<script>
  import { onMount } from "svelte";

  // State Runes
  /** @type {any[]} */
  let logs = $state([]);
  /** @type {string|null} */
  let error = $state(null);
  let loading = $state(true);

  async function fetchLogs() {
    loading = true;
    error = null;
    try {
      const res = await fetch("/api/audit");
      if (!res.ok) {
        if (res.status === 403) {
          throw new Error("Zugriff verweigert: Nur für System-Administratoren.");
        }
        const text = await res.text();
        throw new Error(text || "Fehler beim Laden des Logbuchs");
      }
      logs = await res.json();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchLogs();
  });
</script>

<div class="w-full space-y-6 animate-fade-in no-print">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-xl font-bold text-slate-850 tracking-tight">Enterprise Audit-Logbuch</h2>
      <p class="text-xs text-slate-500">Unveränderliches Register aller sicherheitsrelevanten Löschungen von Buchtiteln und Benutzern.</p>
    </div>
    <button onclick={fetchLogs} class="px-4 py-2 text-xs font-semibold rounded-xl bg-white border border-slate-200 text-slate-600 hover:bg-slate-50 transition-all cursor-pointer">
      🔄 Aktualisieren
    </button>
  </div>

  {#if loading}
    <div class="p-12 text-center text-slate-400 font-medium animate-pulse">Lade Logbuch-Einträge...</div>
  {:else if error}
    <div class="p-6 rounded-2xl bg-rose-50 border border-rose-100 text-rose-600 text-sm font-medium">{error}</div>
  {:else if logs.length === 0}
    <div class="p-12 rounded-2xl border border-dashed border-slate-200 bg-white text-center text-slate-400">
      <span class="text-2xl block mb-2">📜</span>
      Keine Audit-Einträge vorhanden.
    </div>
  {:else}
    <div class="border border-slate-100 bg-white rounded-2xl overflow-hidden shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-100 text-base font-semibold text-slate-500 uppercase font-mono tracking-wider">
              <th class="p-4.5">Zeitstempel</th>
              <th class="p-4.5">Aktion</th>
              <th class="p-4.5">Tabelle</th>
              <th class="p-4.5">Datensatz-ID</th>
              <th class="p-4.5">Bearbeiter (Operator)</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 text-base text-slate-600">
            {#each logs as log}
              <tr class="hover:bg-slate-50/50 transition-colors">
                <td class="p-4.5 font-mono text-xs text-slate-500">
                  {new Date(log.timestamp).toLocaleString("de-DE")}
                </td>
                <td class="p-4.5">
                  <span class="inline-flex px-2 py-0.5 rounded-md font-mono text-xs font-bold bg-rose-50 border border-rose-100 text-rose-600">
                    {log.aktion}
                  </span>
                </td>
                <td class="p-4.5 font-mono text-xs text-emerald-600">
                  {log.tabelle}
                </td>
                <td class="p-4.5 font-mono text-xs text-slate-400">
                  {log.datensatz_id}
                </td>
                <td class="p-4.5">
                  <span class="font-medium text-slate-700">{log.bearbeiter_vorname} {log.bearbeiter_nachname}</span>
                  <span class="block text-[10px] text-slate-400 font-mono">{log.bearbeiter_id}</span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>
