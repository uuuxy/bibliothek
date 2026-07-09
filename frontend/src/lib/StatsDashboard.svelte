<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import OverdueWidget from "./OverdueWidget.svelte";
  import StatistikDetailPanel from "./components/stats/StatistikDetailPanel.svelte";

  // State Runes (Svelte 5)
  /** @type {any} */
  let stats = $state(null);
  let loading = $state(true);
  let selectedTimeframe = $state("all");
  /** @type {'renner' | 'ladenhueter' | null} Drill-Down-Panel */
  let activePanel = $state(null);

  const TIMEFRAMES = [
    { value: "all",       label: "Alle" },
    { value: "schuljahr", label: "Schuljahr" },
    { value: "monat",     label: "Monat" },
  ];

  // Kacheln zeigen Top 5; das Drill-Down-Panel filtert die volle Liste clientseitig
  const topRenner = $derived(stats?.popular_titles?.slice(0, 5) ?? []);
  const topWarmers = $derived(stats?.shelf_warmers?.slice(0, 5) ?? []);

  // Fetch statistics from backend API.
  // limit=100 lädt die Drill-Down-Daten gleich mit — das Panel braucht
  // dadurch keinen einzigen weiteren API-Call.
  async function fetchStats() {
    loading = true;
    try {
      const params = new URLSearchParams({ limit: "100" });
      if (selectedTimeframe !== "all") params.set("zeitraum", selectedTimeframe);
      const res = await apiFetch(`/api/statistiken?${params}`);
      if (!res.ok) throw new Error("Fehler beim Laden");
      stats = await res.json();
    } catch (err) {
      console.error("Stats loading error:", err);
    } finally {
      loading = false;
    }
  }

  // Re-fetch whenever the selected timeframe changes
  $effect(() => {
    selectedTimeframe; // track dependency
    fetchStats();
  });

</script>

{#snippet drillDownHeader(label, panel)}
  <button
    onclick={() => (activePanel = panel)}
    class="w-full flex items-center justify-between border-b border-slate-100 pb-2 group cursor-pointer text-left"
    aria-label="{label} — Detailansicht öffnen"
  >
    <h3 class="font-bold text-slate-700 text-sm uppercase tracking-wider font-sans group-hover:text-slate-900 transition-colors">{label}</h3>
    <span class="flex items-center gap-1 text-[11px] font-bold text-slate-400 group-hover:text-blue-600 transition-colors">
      Alle
      <svg class="w-3.5 h-3.5 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M7 17L17 7M7 7h10v10" /></svg>
    </span>
  </button>
{/snippet}

<div class="w-full space-y-6 text-slate-800">
  
  <!-- Header Info & Period Filter -->
  <div class="flex flex-col md:flex-row md:items-center md:justify-end gap-4 border-b border-slate-100 pb-5">

    <!-- Time Filter Buttons -->
    <div class="flex items-center gap-2 self-start md:self-center">
      <span class="text-sm font-semibold text-slate-400 uppercase tracking-wider font-sans">Zeitraum:</span>
      <div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200">
        {#each TIMEFRAMES as tf}
          <button
            onclick={() => selectedTimeframe = tf.value}
            class="px-4 py-1.5 text-sm font-bold rounded-lg cursor-pointer transition-all {selectedTimeframe === tf.value ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-700'}"
          >{tf.label}</button>
        {/each}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="py-12 flex justify-center items-center">
      <div class="w-8 h-8 border-2 border-t-blue-500 border-blue-500/20 rounded-full animate-spin"></div>
    </div>
  {:else if stats}
    <!-- Inventory Metrics & Loss Rate Card Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="p-6 flex flex-col justify-between space-y-2 text-left border border-gray-200 sm:border-0 sm:border-l sm:first:border-l-0">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Gesamtbestand</span>
        <span class="text-4xl font-extrabold text-slate-900 leading-none py-1">{stats.loss_stats.gesamt_bestand}</span>
        <span class="text-sm text-slate-500 font-medium">Physische Buchkopien im System</span>
      </div>
      
      <div class="p-6 flex flex-col justify-between space-y-2 text-left border border-gray-200 sm:border-0 sm:border-l sm:first:border-l-0">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Verlorene / Defekte Bücher</span>
        <span class="text-4xl font-extrabold text-rose-600 leading-none py-1">{stats.loss_stats.verlorene_exemplare}</span>
        <span class="text-sm text-slate-500 font-medium">Exemplare mit Schadensfällen</span>
      </div>
 
      <div class="p-6 flex flex-col justify-between space-y-2 text-left border border-gray-200 sm:border-0 sm:border-l sm:first:border-l-0">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Verlustquote</span>
        <span class="text-4xl font-extrabold text-amber-600 leading-none py-1">{stats.loss_stats.verlust_quote}%</span>
        <span class="text-sm text-slate-500 font-medium">Prozentsatz verlorener Lehrmittel</span>
      </div>
    </div>

    <!-- Stats Tables Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 pt-6">
      
      <!-- Overdue Widget -->
      <div class="space-y-3 text-left h-full">
        <OverdueWidget />
      </div>
      <!-- Top Borrowed Books Section ("Die Renner") -->
      <div class="space-y-3 text-left">
        {@render drillDownHeader("Beliebteste Titel (Die Renner)", "renner")}
        
        <div class="w-full">
          <table class="w-full text-left text-base border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-100 text-sm font-bold text-slate-400 font-sans uppercase tracking-wider">
                <th class="py-3 px-4">Buchtitel</th>
                <th class="py-3 px-4 text-right">Ausleihen</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 text-sm text-slate-650 font-semibold">
              {#if !stats.popular_titles || stats.popular_titles.length === 0}
                <tr>
                  <td colspan="2" class="py-12 text-center text-xs text-slate-400 font-medium">
                    <span class="text-2xl block mb-2">📊</span>
                    Noch keine Ausleihen registriert
                  </td>
                </tr>
              {:else}
                {#each topRenner as book}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="py-3 px-4 flex items-center gap-3">
                      <!-- Cover Thumbnail -->
                      {#if book.cover_url}
                        <img 
                          src={book.cover_url} 
                          alt="Cover" 
                          class="w-10 aspect-3/4 object-cover rounded shadow-sm border border-slate-100/50 shrink-0" 
                        />
                      {:else}
                        <div class="w-10 aspect-3/4 bg-slate-50 border border-slate-150 rounded flex items-center justify-center text-slate-400 text-xs shadow-sm shrink-0 font-medium">
                          📖
                        </div>
                      {/if}
                      
                      <!-- Title & Author -->
                      <div class="min-w-0">
                        <span class="font-bold text-slate-800 text-sm truncate block" title={book.titel}>{book.titel}</span>
                        <span class="text-slate-450 text-xs block font-medium truncate" title={book.autor}>{book.autor}</span>
                      </div>
                    </td>
                    <td class="py-3 px-4 text-slate-900 font-bold text-right shrink-0">
                      {book.count}x geliehen
                    </td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      </div>

      <!-- Shelf Warmers Table -->
      <div class="space-y-3 text-left">
        {@render drillDownHeader("Ladenhüter", "ladenhueter")}
        
        <div class="w-full">
          <table class="w-full text-left text-base border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-100 text-sm font-bold text-slate-400 font-sans uppercase tracking-wider">
                <th class="py-3 px-4">Buchtitel</th>
                <th class="py-3 px-4">Autor</th>
                <th class="py-3 px-4 text-right">Zuletzt geliehen</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 text-sm text-slate-650 font-semibold">
              {#if !stats.shelf_warmers || stats.shelf_warmers.length === 0}
                <tr>
                  <td colspan="3" class="py-12 text-center text-xs text-slate-400 font-medium">
                    <span class="text-2xl block mb-2">🕳️</span>
                    Keine Ladenhüter identifiziert
                  </td>
                </tr>
              {:else}
                {#each topWarmers as book}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="py-3.5 px-4 text-slate-800 font-bold truncate max-w-[160px]" title={book.titel}>{book.titel}</td>
                    <td class="py-3.5 px-4 text-slate-500 truncate max-w-[120px]" title={book.autor}>{book.autor}</td>
                    <td class="py-3.5 px-4 text-amber-600 font-bold text-right">{book.letzte_aus}</td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      </div>

    </div>
  {/if}

  {#if activePanel === "renner"}
    <StatistikDetailPanel kind="renner" items={stats?.popular_titles ?? []} onClose={() => (activePanel = null)} />
  {:else if activePanel === "ladenhueter"}
    <StatistikDetailPanel kind="ladenhueter" items={stats?.shelf_warmers ?? []} onClose={() => (activePanel = null)} />
  {/if}
</div>
