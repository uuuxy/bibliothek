<script>
  import { mahnwesenStore } from "../../stores/mahnwesen.svelte.js";

  let countAlle = $derived(
    mahnwesenStore.klassen.reduce((sum, k) => sum + k.schueler.length, 0)
  );

  let countAkut = $derived(
    mahnwesenStore.klassen.reduce((sum, k) =>
      sum + k.schueler.filter(s => {
        const isLehrer = s.klasse && s.klasse.toLowerCase() === 'lehrer';
        const maxTage = s.medien.reduce((max, m) => m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max, 0);
        return maxTage > 0 && maxTage <= 14 && !isLehrer;
      }).length, 0)
  );

  let countEskaliert = $derived(
    mahnwesenStore.klassen.reduce((sum, k) =>
      sum + k.schueler.filter(s => {
        const isLehrer = s.klasse && s.klasse.toLowerCase() === 'lehrer';
        const maxTage = s.medien.reduce((max, m) => m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max, 0);
        return maxTage > 14 && !isLehrer;
      }).length, 0)
  );

  let countKollegium = $derived(
    mahnwesenStore.klassen.reduce((sum, k) =>
      sum + k.schueler.filter(s => s.klasse && s.klasse.toLowerCase() === 'lehrer').length, 0)
  );
</script>

<div class="flex items-center justify-between">
  <div>
    <h1 class="text-2xl font-bold text-slate-800">Mahnwesen</h1>
    <p class="text-sm text-slate-500 mt-0.5">Überfällige Ausleihen nach Klassen sortiert.</p>
  </div>
  
  <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-xl print:hidden">
    <button 
      class="px-4 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode === 'datum' ? 'bg-white text-slate-800 shadow-sm' : 'text-slate-500 hover:text-slate-700'}"
      onclick={() => { mahnwesenStore.mahnMode = 'datum'; mahnwesenStore.fetchData(); }}
    >
      Datum
    </button>
    <button 
      class="px-4 py-1.5 rounded-lg text-sm font-medium transition-colors {mahnwesenStore.mahnMode === 'jahrgang' ? 'bg-white text-slate-800 shadow-sm' : 'text-slate-500 hover:text-slate-700'}"
      onclick={() => { mahnwesenStore.mahnMode = 'jahrgang'; mahnwesenStore.fetchData(); }}
    >
      Jahrgang
    </button>
  </div>
  <div class="flex items-center gap-2 print:hidden">
    <button
      onclick={mahnwesenStore.fetchData}
      aria-label="Daten neu laden"
      class="px-3 py-2 rounded-xl border border-slate-200 bg-white text-slate-600 hover:bg-slate-50 text-xs font-semibold transition-all flex items-center gap-1.5"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    </button>
    <button
      onclick={() => window.print()}
      aria-label="Seite drucken"
      class="px-3 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-bold transition-all flex items-center gap-1.5 print:hidden"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
      </svg>
      Drucken
    </button>
    <button
      onclick={mahnwesenStore.downloadPDF}
      disabled={mahnwesenStore.pdfLoading}
      aria-label="Mahnliste gesamt als PDF herunterladen"
      class="px-3 py-2 rounded-xl bg-slate-700 hover:bg-slate-800 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-1.5"
    >
      {#if mahnwesenStore.pdfLoading}
        <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
      {/if}
      Mahnliste (gesamt) als PDF
    </button>
    <button
      onclick={mahnwesenStore.downloadElternPDF}
      disabled={mahnwesenStore.elternPdfLoading}
      aria-label="Mahnlauf starten (PDF)"
      class="px-3 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-1.5"
    >
      {#if mahnwesenStore.elternPdfLoading}
        <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
      {/if}
      Mahnlauf starten (PDF)
    </button>

    <div class="flex items-center gap-1 bg-slate-100 rounded-xl px-2 py-1">
      <select bind:value={mahnwesenStore.selectedKlasse} aria-label="Klasse wählen" class="bg-transparent text-xs font-bold text-slate-700 py-1 focus:outline-none">
        <option value="">Klasse wählen...</option>
        {#each mahnwesenStore.klassen as k}
          <option value={k.klasse}>{k.klasse}</option>
        {/each}
      </select>
      <button
        onclick={mahnwesenStore.downloadKlassePDF}
        disabled={mahnwesenStore.klassePdfLoading || !mahnwesenStore.selectedKlasse}
        aria-label="Mahnliste der ausgewählten Klasse als PDF herunterladen"
        class="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-1"
      >
        {#if mahnwesenStore.klassePdfLoading}
          <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        {/if}
        Drucken
      </button>
    </div>

  </div>
</div>

<!-- Tabs Navigation -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
  <div class="flex space-x-1 border-b border-gray-200 mt-6 print:hidden">
    <!-- Alle Tab -->
    <button
      class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter === 'Alle' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
      onclick={() => mahnwesenStore.activeFilter = 'Alle'}
    >
      Alle
      <span class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter === 'Alle' && countAlle > 0 ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'}">
        {countAlle}
      </span>
    </button>

    <!-- Akut fällig Tab -->
    <button
      class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter === '1. Erinnerung' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
      onclick={() => mahnwesenStore.activeFilter = '1. Erinnerung'}
    >
      Akut fällig
      <span class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter === '1. Erinnerung' && countAkut > 0 ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'}">
        {countAkut}
      </span>
    </button>

    <!-- Eskaliert Tab -->
    <button
      class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter === 'Mahnung' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
      onclick={() => mahnwesenStore.activeFilter = 'Mahnung'}
    >
      Eskaliert
      <span class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter === 'Mahnung' && countEskaliert > 0 ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'}">
        {countEskaliert}
      </span>
    </button>

    <!-- Kollegium Tab -->
    <button
      class="flex items-center px-4 py-2 text-sm font-medium transition-colors {mahnwesenStore.activeFilter === 'Lehrerkollegium' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'}"
      onclick={() => mahnwesenStore.activeFilter = 'Lehrerkollegium'}
    >
      Kollegium
      <span class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {mahnwesenStore.activeFilter === 'Lehrerkollegium' && countKollegium > 0 ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'}">
        {countKollegium}
      </span>
    </button>
  </div>
{/if}
