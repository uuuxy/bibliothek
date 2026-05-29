<script>
  import { onMount } from "svelte";

  // State Runes
  /** @type {any} */
  let stats = $state(null);
  let loading = $state(true);

  // Fetch statistics from backend API
  async function fetchStats() {
    try {
      const res = await fetch("/api/statistiken");
      if (!res.ok) throw new Error("Fehler beim Laden");
      stats = await res.json();
    } catch (err) {
      console.error("Stats loading error:", err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchStats();
  });
</script>

<div class="w-full space-y-6 text-slate-800">
  
  <!-- Header Info -->
  <div class="flex items-center justify-between border-b border-slate-100 pb-5">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Bestandsanalyse</span>
      <h2 class="text-2xl font-bold text-slate-900">Bibliotheksstatistiken</h2>
      <p class="text-xs text-slate-500 font-medium">Schlichte Übersicht zur Optimierung des Bestands und Verlustverfolgung.</p>
    </div>
  </div>

  {#if loading}
    <div class="py-12 flex justify-center items-center">
      <div class="w-8 h-8 border-2 border-t-emerald-400 border-emerald-400/20 rounded-full animate-spin"></div>
    </div>
  {:else if stats}
    <!-- Inventory Metrics & Loss Rate Card Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="bg-white border border-slate-200 rounded-xl shadow-sm p-6 flex flex-col justify-between space-y-2 text-left">
        <span class="text-[10px] uppercase font-bold tracking-wider text-slate-400 font-mono">Gesamtbestand</span>
        <span class="text-4xl font-extrabold text-slate-900 font-mono leading-none py-1">{stats.loss_stats.gesamt_bestand}</span>
        <span class="text-[10px] text-slate-550 font-medium">Physische Buchkopien im System</span>
      </div>
      
      <div class="bg-white border border-slate-200 rounded-xl shadow-sm p-6 flex flex-col justify-between space-y-2 text-left">
        <span class="text-[10px] uppercase font-bold tracking-wider text-slate-400 font-mono">Verlorene / Defekte Bücher</span>
        <span class="text-4xl font-extrabold text-rose-600 font-mono leading-none py-1">{stats.loss_stats.verlorene_exemplare}</span>
        <span class="text-[10px] text-slate-550 font-medium">Exemplare mit Schadensfällen</span>
      </div>

      <div class="bg-white border border-slate-200 rounded-xl shadow-sm p-6 flex flex-col justify-between space-y-2 text-left">
        <span class="text-[10px] uppercase font-bold tracking-wider text-slate-400 font-mono">Verlustquote</span>
        <span class="text-4xl font-extrabold text-amber-600 font-mono leading-none py-1">{stats.loss_stats.verlust_quote}%</span>
        <span class="text-[10px] text-slate-550 font-medium">Prozentsatz verlorener Lehrmittel</span>
      </div>
    </div>

    <!-- Stats Tables Layout -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-4">
      
      <!-- Top Borrowers Table -->
      <div class="space-y-3 text-left">
        <h3 class="font-bold text-slate-800 text-sm uppercase tracking-wider font-mono border-b border-slate-100 pb-2">Top-Ausleiher (Klassen)</h3>
        {#if !stats.top_classes || stats.top_classes.length === 0}
          <p class="text-xs text-slate-550 py-6 text-center font-mono bg-white border border-slate-100 rounded-xl">Keine Ausleihen registriert</p>
        {:else}
          <div class="overflow-hidden border border-slate-150 rounded-xl bg-white shadow-xs">
            <table class="w-full text-left text-xs border-collapse">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-100 text-slate-500 font-mono">
                  <th class="py-2.5 px-4 font-semibold">Rang</th>
                  <th class="py-2.5 px-4 font-semibold">Klasse</th>
                  <th class="py-2.5 px-4 font-semibold text-right">Ausleihen</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                {#each stats.top_classes as row, idx}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="py-3 px-4 font-mono text-slate-400">{idx + 1}.</td>
                    <td class="py-3 px-4 text-slate-800 font-bold">{row.klasse}</td>
                    <td class="py-3 px-4 text-emerald-600 font-mono font-bold text-right">{row.count}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

      <!-- Shelf Warmers Table -->
      <div class="space-y-3 text-left">
        <h3 class="font-bold text-slate-800 text-sm uppercase tracking-wider font-mono border-b border-slate-100 pb-2">Ladenhüter</h3>
        {#if !stats.shelf_warmers || stats.shelf_warmers.length === 0}
          <p class="text-xs text-slate-550 py-6 text-center font-mono bg-white border border-slate-100 rounded-xl">Keine Ladenhüter identifiziert</p>
        {:else}
          <div class="overflow-hidden border border-slate-150 rounded-xl bg-white shadow-xs">
            <table class="w-full text-left text-xs border-collapse">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-100 text-slate-500 font-mono">
                  <th class="py-2.5 px-4 font-semibold">Buchtitel</th>
                  <th class="py-2.5 px-4 font-semibold">Autor</th>
                  <th class="py-2.5 px-4 font-semibold text-right">Zuletzt geliehen</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                {#each stats.shelf_warmers as book}
                  <tr class="hover:bg-slate-50/50 transition-colors">
                    <td class="py-3 px-4 text-slate-800 font-bold truncate max-w-[140px]" title={book.titel}>{book.titel}</td>
                    <td class="py-3 px-4 text-slate-500 truncate max-w-[100px]" title={book.autor}>{book.autor}</td>
                    <td class="py-3 px-4 text-amber-600 font-mono font-bold text-right">{book.letzte_aus}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

    </div>
  {/if}
</div>
