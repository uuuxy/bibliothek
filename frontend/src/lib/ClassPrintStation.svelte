<script>
  import { onMount } from "svelte";
  import { apiFetch } from "./apiFetch.js";
  import StudentPrintCard from "./StudentPrintCard.svelte";

  let { onBack } = $props();

  let classes = $state([]);
  let selectedClass = $state("");
  let students = $state([]);
  let loadingClasses = $state(true);
  let loadingStudents = $state(false);
  let error = $state("");
  const timestamp = Date.now();

  onMount(async () => {
    try {
      const res = await apiFetch("/api/klassen");
      if (res.ok) {
        classes = await res.json();
      } else {
        error = "Klassen konnten nicht geladen werden.";
      }
    } catch (e) {
      error = "Fehler beim Laden der Klassen.";
    } finally {
      loadingClasses = false;
    }
  });

  async function handleClassChange() {
    if (!selectedClass) {
      students = [];
      return;
    }
    loadingStudents = true;
    error = "";
    try {
      const res = await apiFetch(`/api/schueler?klasse=${encodeURIComponent(selectedClass)}`);
      if (res.ok) {
        students = await res.json();
      } else {
        error = "Schüler dieser Klasse konnten nicht geladen werden.";
        students = [];
      }
    } catch (e) {
      error = "Fehler beim Laden der Schüler.";
      students = [];
    } finally {
      loadingStudents = false;
    }
  }

  function handlePrint() {
    window.print();
  }
</script>

<style>
  @media print {
    @page {
      margin: 0;
      size: 85.6mm 54mm;
    }
    
    :global(body) {
      background: white !important;
      -webkit-print-color-adjust: exact;
      print-color-adjust: exact;
    }

    .card-wrapper {
      page-break-after: always;
      break-after: page;
      width: 85.6mm;
      height: 54mm;
      position: relative;
      overflow: hidden;
    }
    
    /* Remove any margins/paddings added by the app layout during print */
    :global(#app), :global(main) {
      padding: 0 !important;
      margin: 0 !important;
    }
  }
  
  @media screen {
    .card-wrapper {
      width: 85.6mm;
      height: 54mm;
      position: relative;
      overflow: hidden;
      border: 1px solid #e2e8f0;
      border-radius: 0.5rem;
      box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
      margin-bottom: 1rem;
      background: white;
    }
  }
</style>

<div class="w-full max-w-4xl mx-auto p-6 space-y-6">
  <!-- UI: Print hidden -->
  <div class="print:hidden space-y-6">
    <button 
      onclick={onBack}
      class="inline-flex items-center gap-2 text-sm font-semibold text-slate-500 hover:text-slate-800 transition-colors"
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
      </svg>
      Zurück zur Schülerliste
    </button>

    <div class="p-6">
      <div>
        <h2 class="text-xl font-bold text-slate-800 mb-1">Massendruck für Klassen</h2>
      <p class="text-sm text-slate-500">Wähle eine Klasse, um alle zugehörigen Ausweise zu laden und an den Kartendrucker zu senden.</p>
    </div>

    {#if error}
      <div class="p-4 bg-red-50 text-red-700 rounded-lg text-sm border border-red-200">
        {error}
      </div>
    {/if}

    <div class="flex items-end gap-4">
      <div class="flex-1 max-w-xs">
        <label for="class-select" class="block text-sm font-medium text-slate-700 mb-1">Klasse auswählen</label>
        <select 
          id="class-select"
          bind:value={selectedClass} 
          onchange={handleClassChange}
          disabled={loadingClasses || loadingStudents}
          class="w-full border-slate-300 rounded-lg shadow-sm focus:border-blue-500 focus:ring-blue-500 disabled:opacity-50"
        >
          <option value="">Bitte wählen...</option>
          {#each classes as klasse}
            <option value={klasse}>{klasse}</option>
          {/each}
        </select>
      </div>

      {#if students.length > 0}
        <button 
          onclick={handlePrint}
          class="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg shadow-sm flex items-center gap-2 transition-colors"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
          </svg>
          Druckvorgang an Kartendrucker senden
        </button>
      {/if}
    </div>
    </div>
  </div>

  <!-- Loading State (UI only) -->
  {#if loadingStudents}
    <div class="print:hidden flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
    </div>
  {/if}

  <!-- Preview (UI) & Print Output -->
  {#if students.length > 0 && !loadingStudents}
    <div class="print:hidden mb-4 flex items-center justify-between">
      <h3 class="font-bold text-slate-800">Vorschau ({students.length} Ausweise)</h3>
      <span class="text-xs bg-slate-100 text-slate-600 px-2 py-1 rounded">Nur für Datacard-Drucker</span>
    </div>

    <!-- The actual print block. On screen we show it as a grid or list of cards. -->
    <div class="flex flex-wrap gap-4 print:block">
      {#each students as student}
        <div class="card-wrapper">
          <StudentPrintCard profile={student} {timestamp} />
        </div>
      {/each}
    </div>
  {/if}
</div>
