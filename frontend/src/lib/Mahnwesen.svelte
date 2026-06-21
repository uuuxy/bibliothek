<script>
  import { onMount } from "svelte";
  import { mahnwesenStore } from "./stores/mahnwesen.svelte.js";
  import { offlineSync } from "./stores/offlineSync.svelte.js";
  import MahnwesenFilters from "./components/mahnwesen/MahnwesenFilters.svelte";
  import MahnwesenTable from "./components/mahnwesen/MahnwesenTable.svelte";

  $effect(() => {
    if (offlineSync.pendingCount === 0) {
      mahnwesenStore.fetchData();
    }
  });
</script>

{#if mahnwesenStore.globalErrorToast}
  <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd" />
    </svg>
    {mahnwesenStore.globalErrorToast}
  </div>
{/if}

{#if mahnwesenStore.ferienAktiv}
  <div class="w-full mb-6 p-4 bg-amber-50 border-b border-amber-200 flex items-start gap-3 animate-fade-in">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-amber-600 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
    </svg>
    <div>
      <h3 class="text-sm font-bold text-amber-900">Achtung: Schließzeit / Ferien aktiv!</h3>
      <p class="text-xs text-amber-800 mt-1">Das Mahnwesen ist aktuell pausiert. Grund: <strong>{mahnwesenStore.ferienBezeichnung}</strong>. E-Mails und PDF-Exporte sind währenddessen serverseitig blockiert.</p>
    </div>
  </div>
{/if}

<div class="w-full h-full flex flex-col">
  {#if offlineSync.pendingCount > 0}
    <div class="p-4 bg-rose-50 border-b border-rose-200 flex items-start gap-4 animate-fade-in w-full">
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
  {:else}
    <div class="animate-fade-in flex-1 flex flex-col w-full">
      <!-- Header und Filter -->
      <MahnwesenFilters />

      <!-- Tabellen und Modals -->
      <div class="w-full">
        <MahnwesenTable />
      </div>
    </div>
  {/if}
</div>
