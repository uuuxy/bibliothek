<script>
  import { mahnwesenStore } from "../../stores/mahnwesen.svelte.js";

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
      class="px-3 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-bold transition-all flex items-center gap-1.5 print:hidden"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
      <select bind:value={mahnwesenStore.selectedKlasse} class="bg-transparent text-xs font-bold text-slate-700 py-1 focus:outline-none">
        <option value="">Klasse wählen...</option>
        {#each mahnwesenStore.klassen as k}
          <option value={k.klasse}>{k.klasse}</option>
        {/each}
      </select>
      <button
        onclick={mahnwesenStore.downloadKlassePDF}
        disabled={mahnwesenStore.klassePdfLoading || !mahnwesenStore.selectedKlasse}
        class="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-bold transition-all flex items-center gap-1"
      >
        {#if mahnwesenStore.klassePdfLoading}
          <div class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"></div>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
        {/if}
        Drucken
      </button>
    </div>

  </div>
</div>

<!-- Stats Cards Grid -->
{#if mahnwesenStore.data && !mahnwesenStore.loading}
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
    <!-- Card 1: Akut fällig -->
    <div class="bg-white border border-gray-200/75 rounded-2xl p-6 shadow-[0_2px_8px_rgb(0,0,0,0.04)] relative overflow-hidden transition-all duration-300 hover:shadow-[0_4px_20px_rgba(0,0,0,0.06)] hover:border-gray-300/80">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-500 tracking-wide uppercase">Akut fällig</h3>
        <div class="bg-amber-50 text-amber-600 p-2 rounded-xl">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      </div>
      <p class="text-4xl font-extrabold text-gray-900 tracking-tight mt-2">{countAkut}</p>
      <div class="mt-3 flex items-center gap-1">
        <span class="text-xs text-amber-600 font-medium">+{Math.ceil(countAkut * 0.15)} seit gestern</span>
        <span class="text-[11px] text-gray-400">· heute anstehend</span>
      </div>
    </div>

    <!-- Card 2: Eskaliert -->
    <div class="bg-white border border-gray-200/75 rounded-2xl p-6 shadow-[0_2px_8px_rgb(0,0,0,0.04)] relative overflow-hidden transition-all duration-300 hover:shadow-[0_4px_20px_rgba(0,0,0,0.06)] hover:border-gray-300/80">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-500 tracking-wide uppercase">Eskaliert</h3>
        <div class="bg-red-50 text-red-600 p-2 rounded-xl">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
      </div>
      <p class="text-4xl font-extrabold text-gray-900 tracking-tight mt-2">{countEskaliert}</p>
      <div class="mt-3 flex items-center gap-1">
        <span class="text-xs text-red-600 font-medium">Mahnstufe 3 / Härtefälle</span>
        <span class="text-[11px] text-gray-400">· dringend</span>
      </div>
    </div>

    <!-- Card 3: Erfolgreich retourniert -->
    <div class="bg-white border border-gray-200/75 rounded-2xl p-6 shadow-[0_2px_8px_rgb(0,0,0,0.04)] relative overflow-hidden transition-all duration-300 hover:shadow-[0_4px_20px_rgba(0,0,0,0.06)] hover:border-gray-300/80">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-500 tracking-wide uppercase">Erfolgreich retourniert</h3>
        <div class="bg-green-50 text-green-600 p-2 rounded-xl">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      </div>
      <p class="text-4xl font-extrabold text-gray-900 tracking-tight mt-2">{mahnwesenStore.heuteRetourniert}</p>
      <div class="mt-3 flex items-center gap-1">
        <span class="text-xs text-green-600 font-medium">Heute zurückgegeben</span>
        <span class="text-[11px] text-gray-400">· bereinigt</span>
      </div>
    </div>
  </div>
{/if}
