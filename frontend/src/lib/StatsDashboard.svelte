<script>
  import { onMount } from "svelte";

  // State Runes (Svelte 5)
  /** @type {any} */
  let stats = $state(null);
  let loading = $state(true);
  let selectedTimeframe = $state("all");

  const TIMEFRAMES = [
    { value: "all",       label: "Alle" },
    { value: "schuljahr", label: "Schuljahr" },
    { value: "monat",     label: "Monat" },
  ];

  // Fetch statistics from backend API
  async function fetchStats() {
    loading = true;
    try {
      const params = selectedTimeframe !== "all" ? `?zeitraum=${selectedTimeframe}` : "";
      const res = await fetch(`/api/statistiken${params}`);
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

  onMount(() => {
    // initial fetch handled by $effect above
  });
</script>

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
      <div class="bg-white border border-slate-200 rounded-2xl shadow-xs p-6 flex flex-col justify-between space-y-2 text-left hover:border-slate-300 transition-all">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Gesamtbestand</span>
        <span class="text-4xl font-extrabold text-slate-900 font-mono leading-none py-1">{stats.loss_stats.gesamt_bestand}</span>
        <span class="text-sm text-slate-500 font-medium">Physische Buchkopien im System</span>
      </div>
      
      <div class="bg-white border border-slate-200 rounded-2xl shadow-xs p-6 flex flex-col justify-between space-y-2 text-left hover:border-slate-300 transition-all">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Verlorene / Defekte Bücher</span>
        <span class="text-4xl font-extrabold text-rose-600 font-mono leading-none py-1">{stats.loss_stats.verlorene_exemplare}</span>
        <span class="text-sm text-slate-500 font-medium">Exemplare mit Schadensfällen</span>
      </div>

      <div class="bg-white border border-slate-200 rounded-2xl shadow-xs p-6 flex flex-col justify-between space-y-2 text-left hover:border-slate-300 transition-all">
        <span class="text-sm font-semibold uppercase tracking-wider text-slate-400 font-sans">Verlustquote</span>
        <span class="text-4xl font-extrabold text-amber-600 font-mono leading-none py-1">{stats.loss_stats.verlust_quote}%</span>
        <span class="text-sm text-slate-500 font-medium">Prozentsatz verlorener Lehrmittel</span>
      </div>
    </div>

    <!-- Stats Tables Layout -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-4">
      
      <!-- Top Borrowed Books Section ("Die Renner") -->
      <div class="space-y-3 text-left">
        <h3 class="font-bold text-slate-700 text-sm uppercase tracking-wider font-sans border-b border-slate-100 pb-2">Beliebteste Titel (Die Renner)</h3>
        
        <div class="overflow-hidden border border-slate-200 rounded-2xl bg-white shadow-xs">
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
                {#each stats.popular_titles as book}
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
                    <td class="py-3 px-4 text-slate-900 font-mono font-bold text-right shrink-0">
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
        <h3 class="font-bold text-slate-700 text-sm uppercase tracking-wider font-sans border-b border-slate-100 pb-2">Ladenhüter</h3>
        
        <div class="overflow-hidden border border-slate-200 rounded-2xl bg-white shadow-xs">
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
                {#each stats.shelf_warmers as book}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="py-3.5 px-4 text-slate-800 font-bold truncate max-w-[160px]" title={book.titel}>{book.titel}</td>
                    <td class="py-3.5 px-4 text-slate-500 truncate max-w-[120px]" title={book.autor}>{book.autor}</td>
                    <td class="py-3.5 px-4 text-amber-600 font-mono font-bold text-right">{book.letzte_aus}</td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      </div>

    </div>
  {/if}
</div>
