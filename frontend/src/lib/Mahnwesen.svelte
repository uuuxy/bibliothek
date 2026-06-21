<script>
  import { onMount } from "svelte";
  import { mahnwesenStore } from "./stores/mahnwesen.svelte.js";
  import { offlineSync } from "./stores/offlineSync.svelte.js";
  import MahnwesenFilters from "./components/mahnwesen/MahnwesenFilters.svelte";
  import MahnwesenTable from "./components/mahnwesen/MahnwesenTable.svelte";

  $effect(() => {
    if (offlineSync.pendingCount === 0 && activeView) {
      mahnwesenStore.fetchData();
    }
  });
  
  let activeView = $state("");

  const iconSchueler = '<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />';
  const iconAbitur = '<path stroke-linecap="round" stroke-linejoin="round" d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />';
  const iconLehrer = '<path stroke-linecap="round" stroke-linejoin="round" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />';
</script>

{#snippet navCard(title, description, iconPath, viewId, colorClass, bgClass)}
  <button 
    onclick={() => activeView = viewId}
    class="bg-white rounded-2xl p-8 flex flex-col items-center justify-center text-center border border-gray-200 transition-all duration-200 hover:-translate-y-1 hover:shadow-lg hover:border-blue-500 cursor-pointer w-full"
  >
    <div class={`w-16 h-16 rounded-full flex items-center justify-center mb-4 ${bgClass} ${colorClass}`}>
      <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        {@html iconPath}
      </svg>
    </div>
    <h3 class="text-lg font-bold text-slate-800">{title}</h3>
    <p class="text-sm text-slate-500 mt-2">{description}</p>
  </button>
{/snippet}

{#if mahnwesenStore.globalErrorToast}
  <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd" />
    </svg>
    {mahnwesenStore.globalErrorToast}
  </div>
{/if}

{#if mahnwesenStore.ferienAktiv}
  <div class="max-w-5xl mx-auto mb-6 p-4 bg-amber-50 border border-amber-200 rounded-xl flex items-start gap-3 animate-fade-in">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-amber-600 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
    </svg>
    <div>
      <h3 class="text-sm font-bold text-amber-900">Achtung: Schließzeit / Ferien aktiv!</h3>
      <p class="text-xs text-amber-800 mt-1">Das Mahnwesen ist aktuell pausiert. Grund: <strong>{mahnwesenStore.ferienBezeichnung}</strong>. E-Mails und PDF-Exporte sind währenddessen serverseitig blockiert.</p>
    </div>
  </div>
{/if}

<div class="max-w-5xl mx-auto space-y-6">
  {#if offlineSync.pendingCount > 0}
    <div class="p-6 bg-rose-50 border border-rose-200 rounded-2xl flex items-start gap-4 animate-fade-in shadow-sm">
      <div class="bg-rose-100 p-3 rounded-full shrink-0">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <div>
        <h2 class="text-lg font-bold text-rose-900">Mahnwesen blockiert</h2>
        <p class="text-sm text-rose-800 mt-1">
          Es befinden sich noch <strong>{offlineSync.pendingCount} ungesynchronisierte Offline-Ausleihe(n)/Rückgabe(n)</strong> auf diesem Gerät. 
        </p>
        <p class="text-xs text-rose-700 mt-2 bg-rose-100/50 p-2 rounded-lg inline-block">
          Bitte stelle die Internetverbindung wieder her. Das System synchronisiert die Daten automatisch im Hintergrund, sobald du wieder online bist. Danach wird das Mahnwesen automatisch wieder freigegeben.
        </p>
      </div>
    </div>
  {:else if !activeView}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 animate-fade-in mt-4">
      {@render navCard("Klasse 5 - 10", "Mahnungen für die Sekundarstufe I", iconSchueler, "klasse5-10", "text-blue-600", "bg-blue-50")}
      {@render navCard("Abitur", "Mahnungen für Oberstufe & Abgänger", iconAbitur, "abitur", "text-emerald-600", "bg-emerald-50")}
      {@render navCard("Lehrerkollegium", "Ausstehende Medien von Lehrkräften", iconLehrer, "lehrer", "text-amber-600", "bg-amber-50")}
    </div>
  {:else}
    <div class="animate-fade-in">
      <button 
        onclick={() => activeView = ''}
        class="mb-6 text-sm font-semibold text-slate-500 hover:text-slate-800 flex items-center gap-2 transition-colors"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Zurück zur Zielgruppen-Auswahl
      </button>
      
      <!-- Header und Filter -->
      <MahnwesenFilters />

      <!-- Tabellen und Modals -->
      <MahnwesenTable />
    </div>
  {/if}
</div>
