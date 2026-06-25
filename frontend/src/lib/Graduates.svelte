<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";
  import LusdImportModal from "./components/students/LusdImportModal.svelte";

  // State Runes
  /** @type {any[]} */
  let graduates = $state([]);
  let loading = $state(true);

  // Laufzettel print state
  let loadingLaufzettel = $state(false);

  async function printLaufzettel() {
    loadingLaufzettel = true;
    try {
      const response = await apiFetch('/api/abgaenger/pdf');
      if (!response.ok) {
        throw new Error("Failed to load PDF");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = "Laufzettel.pdf";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();
    } catch (err) {
      console.error("Laufzettel load error:", err);
    } finally {
      loadingLaufzettel = false;
    }
  }
  
  let showImportModal = $state(false);

  // Fetch graduates list from backend api
  async function fetchGraduates() {
    try {
      const res = await apiFetch("/api/abgaenger");
      if (!res.ok) throw new Error("Fehler beim Laden");
      graduates = await res.json();
    } catch (err) {
      console.error("Graduates error:", err);
    } finally {
      loading = false;
    }
  }

  function closeImportModal() {
    showImportModal = false;
    fetchGraduates();
  }

  onMount(() => {
    // Initial fetch
    fetchGraduates();

    // Listen to Go SSE events for instant UI synchronization
    const source = new EventSource("/events");
    
    // When a book is returned or transferred via the Omnibox,
    // refetch the graduates list to verify if the student is cleared.
    source.addEventListener("action", (e) => {
      try {
        const actionData = JSON.parse(e.data);
        if (actionData.event === "rueckgabe" || actionData.event === "fremdrueckgabe") {
          fetchGraduates();
        }
      } catch (err) {
        console.error("Failed to parse SSE payload:", err);
      }
    });

    return () => {
      source.close();
    };
  });
</script>

<div class="w-full space-y-6 text-slate-800">
  
  <!-- Header Info -->
  <div class="flex items-center justify-end border-b border-slate-100 pb-5">
    <div class="flex items-center space-x-4">
      <button
        onclick={printLaufzettel}
        disabled={loadingLaufzettel || graduates.length === 0}
        class="no-print px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer">
        {#if loadingLaufzettel}
          <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
          Lade Daten…
        {:else}
          🖨️ Laufzettel drucken
        {/if}
      </button>
      <button onclick={() => showImportModal = true} class="no-print px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold rounded-xl text-xs flex items-center gap-1.5 transition-all shadow-xs cursor-pointer">
        📥 LUSD-Datei importieren
      </button>
      <div class="h-2.5 w-2.5 rounded-full bg-emerald-500 animate-pulse shrink-0" title="Live-Synchronisation aktiv"></div>
    </div>
  </div>

  {#if loading}
    <div class="py-12 flex justify-center items-center">
      <div class="w-8 h-8 border-2 border-t-blue-600 border-blue-100 rounded-full animate-spin"></div>
    </div>
  {:else if graduates.length === 0}
    <!-- Completed clearing UI state -->
    <div class="py-12 text-center space-y-3 animate-fade-in">
      <div class="w-16 h-16 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-600 mx-auto">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <h3 class="font-bold text-slate-800">Alle Abgänger entlastet!</h3>
      <p class="text-xs text-slate-500 max-w-xs mx-auto">Keine offenen Lehrmittel oder unbezahlten Schadensfälle in den Klassen 9h, 10r und 13.</p>
    </div>
  {:else}
    <!-- Active list of graduates with dues -->
    <div class="overflow-x-auto">
      <table class="w-full text-left text-base border-collapse">
        <thead>
          <tr class="border-b border-slate-100 text-slate-450 text-sm uppercase">
            <th class="py-3 px-4">Klasse</th>
            <th class="py-3 px-4">Name</th>
            <th class="py-3 px-4">Barcode-ID</th>
            <th class="py-3 px-4">Sperr-Status</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-50">
          {#each graduates as student (student.id)}
            <tr class="hover:bg-slate-50/85 transition-colors animate-slide-up">
              <td class="py-3.5 px-4 font-bold text-blue-600">{student.klasse}</td>
              <td class="py-3.5 px-4 text-slate-700 font-semibold">{student.vorname} {student.nachname}</td>
              <td class="py-3.5 px-4 text-slate-400 text-xs">{student.barcode_id}</td>
              <td class="py-3.5 px-4">
                {#if student.ist_gesperrt}
                  <span class="text-[10px] px-2 py-0.5 rounded bg-rose-50 border border-rose-100 text-rose-600 font-semibold">Sperre aktiv</span>
                {:else}
                  <span class="text-[10px] px-2 py-0.5 rounded bg-slate-100 text-slate-400 font-medium">Bereit</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showImportModal}
  <LusdImportModal onClose={closeImportModal} />
{/if}


