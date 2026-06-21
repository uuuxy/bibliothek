<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { onMount } from "svelte";
  import { authStore } from "./stores/authStore.svelte.js";
  import LusdPreviewModal from "./LusdPreviewModal.svelte";
  import StudentProfile from "./StudentProfile.svelte";
  import ClassPrintStation from "./ClassPrintStation.svelte";
  import StudentCreateModal from "./StudentCreateModal.svelte";
  import Graduates from "./Graduates.svelte";
  import ActiveStudentList from "./components/students/ActiveStudentList.svelte";
  import DeletedStudentList from "./components/students/DeletedStudentList.svelte";

  // Props (Svelte 5)
  let { role = "" } = $props();

  // State Runes (Svelte 5)
  let activeTab = $state("active");
  
  /** @type {any[]} */
  let students = $state.raw([]);
  let loading = $state(false);
  let searchQuery = $state("");
  /** @type {any} */
  let activeStudent = $state(null);

  /** @type {any[]} */
  let readerGroups = $state.raw([]);
  let showCreateModal = $state(false);

  let isImporting = $state(false);
  let showPrintStation = $state(false);
  let importStatusMessage = $state("");
  let importErrorMessage = $state("");
  /** @type {HTMLInputElement | null} */
  let fileInputEl = $state(null);
  let showLusdModal = $state(false);
  let lusdFile = $state(/** @type {File|null} */ (null));

  function triggerImportPicker() {
    importStatusMessage = "";
    importErrorMessage = "";
    fileInputEl?.click();
  }

  /** @param {Event} event */
  function handleLUSDImport(event) {
    const target = /** @type {HTMLInputElement} */ (event.target);
    const file = target.files?.[0];
    if (!file) return;
    
    lusdFile = file;
    showLusdModal = true;
    target.value = ""; // Reset input so same file can be chosen again
  }

  function onLusdSuccess(/** @type {any} */ data) {
    showLusdModal = false;
    lusdFile = null;
    importStatusMessage = `Import erfolgreich: ${data.new_students} neu, ${data.class_changes} geändert, ${data.graduates} Abgänger bearbeitet.`;
    loadStudents();
    loadClasses();
  }

  // Derived filtered students list
  let filteredStudents = $derived.by(() => {
    const q = searchQuery.toLowerCase().trim();
    if (!q) return students;
    return students.filter(s => 
      (s.vorname + " " + s.nachname).toLowerCase().includes(q) ||
      s.klasse.toLowerCase().includes(q) ||
      s.barcode_id.toLowerCase().includes(q)
    );
  });

  async function loadStudents() {
    loading = true;
    try {
      const res = await apiFetch("/api/schueler");
      if (res.ok) {
        students = await res.json();
      }
    } catch (err) {
      console.error("Fehler beim Laden des Schülerverzeichnisses:", err);
    } finally {
      loading = false;
    }
  }

  async function loadClasses() {
    try {
      const res = await apiFetch("/api/readergroups");
      if (res.ok) {
        readerGroups = await res.json() || [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Lesergruppen:", err);
    }
  }

  function handleStudentCreated() {
    showCreateModal = false;
    loadStudents();
    loadClasses(); // refresh class list
  }

  onMount(() => {
    loadStudents();
    loadClasses();
  });
</script>

<div class="w-full h-full flex flex-col text-slate-800 bg-slate-50">
  
  {#if activeStudent}
    <div class="animate-fade-in flex-1 overflow-y-auto">
      <StudentProfile 
        student={activeStudent} 
        {role} 
        onDeselect={() => { activeStudent = null; loadStudents(); }} 
      />
    </div>
  {:else if showPrintStation}
    <div class="animate-fade-in w-full flex-1 overflow-y-auto">
      <ClassPrintStation onBack={() => showPrintStation = false} />
    </div>
  {:else}
    <!-- Tab Navigation Header -->
    <div class="px-8 pt-6 pb-0 border-b border-slate-200 bg-white shrink-0 shadow-sm z-10">
      <div class="max-w-6xl mx-auto flex gap-6">
        <button 
          onclick={() => activeTab = "active"}
          class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'active' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Aktive Schüler
        </button>
        <button 
          onclick={() => activeTab = "graduates"}
          class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'graduates' ? 'border-blue-600 text-blue-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Abgänger / Archiv
        </button>
        {#if role === 'admin'}
        <button 
          onclick={() => activeTab = "deleted"}
          class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === 'deleted' ? 'border-rose-600 text-rose-700' : 'border-transparent text-slate-500 hover:text-slate-800'}"
        >
          Papierkorb
        </button>
        {/if}
      </div>
    </div>

    <!-- Tab Content -->
    <div class="flex-1 overflow-y-auto p-8 w-full">
      <div class="max-w-6xl mx-auto w-full">
        {#if activeTab === "active"}
          <!-- Fullscreen Directory List -->
          <div class="w-full space-y-6 no-print animate-fade-in">
      <!-- Action & Search Bar -->
      <div class="flex items-center gap-4 bg-white p-4 rounded-2xl border border-slate-100 shadow-xs justify-between">
        <div class="flex flex-1 items-center gap-4">
          <div class="relative w-full max-w-md">
            <svg class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input type="text" aria-label="Schüler suchen" placeholder="Nach Name, Klasse oder Barcode filtern..." bind:value={searchQuery} class="w-full pl-10 pr-4 py-2 bg-slate-55 border border-slate-200 rounded-xl text-base text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" />
          </div>

          {#if role === 'admin' || role === 'mitarbeiter'}
            <button onclick={() => showCreateModal = true} aria-label="Neuen Schüler anlegen" class="inline-flex items-center gap-2 bg-blue-600 hover:bg-blue-750 text-white font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
              </svg>
              <span>+ Neuer Schüler</span>
            </button>
          {/if}

          {#if role === 'admin'}
            <input type="file" accept=".csv" bind:this={fileInputEl} onchange={handleLUSDImport} class="hidden" aria-label="LUSD Datei auswählen" />
            <button onclick={triggerImportPicker} aria-label="LUSD-Import starten" class="inline-flex items-center gap-2 bg-slate-100 hover:bg-slate-200/80 text-slate-700 font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0 border border-slate-200">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              <span>LUSD Import (CSV)</span>
            </button>
          {/if}

          {#if authStore.currentUser?.rolle === 'admin' || authStore.currentUser?.rolle === 'mitarbeiter'}
            <button onclick={() => showPrintStation = true} aria-label="Klassensatz drucken" class="inline-flex items-center gap-2 bg-slate-100 hover:bg-slate-200/80 text-slate-700 font-bold py-2 px-4 rounded-xl text-sm transition-all shadow-sm cursor-pointer shrink-0 border border-slate-200">
              <span aria-hidden="true">🖨️</span>
              <span>Klassensatz drucken</span>
            </button>
          {/if}
        </div>
        
        <div class="text-xs font-semibold text-slate-500">
          Einträge: {filteredStudents.length} / {students.length}
        </div>
      </div>

      <!-- Import Status Alert Banner -->
      {#if isImporting}
        <div class="p-4 bg-blue-55 border border-blue-100 rounded-2xl flex items-center justify-between text-xs font-semibold text-blue-700 animate-pulse">
          <div class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-t-blue-600 border-blue-250 rounded-full animate-spin mr-1" aria-hidden="true"></div>
            <span>LUSD-Schülerdaten werden importiert und abgeglichen... Bitte warten.</span>
          </div>
        </div>
      {/if}

      {#if importStatusMessage}
        <div class="p-4 bg-emerald-50 border border-emerald-100 rounded-2xl flex items-center justify-between text-xs font-semibold text-emerald-800">
          <div class="flex items-center gap-2">
            <span>✅ {importStatusMessage}</span>
          </div>
          <button onclick={() => importStatusMessage = ""} aria-label="Hinweis schließen" class="text-emerald-500 hover:text-emerald-700 font-bold text-sm bg-transparent border-none cursor-pointer">✕</button>
        </div>
      {/if}

      {#if importErrorMessage}
        <div class="p-4 bg-rose-50 border border-rose-100 rounded-2xl flex items-center justify-between text-xs font-semibold text-rose-800">
          <div class="flex items-center gap-2">
            <span>⚠️ {importErrorMessage}</span>
          </div>
          <button onclick={() => importErrorMessage = ""} aria-label="Fehler schließen" class="text-rose-500 hover:text-rose-750 font-bold text-sm bg-transparent border-none cursor-pointer">✕</button>
        </div>
      {/if}

      <!-- Table Container -->
      <ActiveStudentList 
        {filteredStudents} 
        {students} 
        {loading} 
        onSelectStudent={(s) => activeStudent = s} 
      />
          </div>
        {:else if activeTab === "graduates"}
          <div class="w-full animate-fade-in">
            <Graduates />
          </div>
        {:else if activeTab === "deleted"}
          <div class="w-full animate-fade-in space-y-6">
            <DeletedStudentList 
              onRestoreSuccess={() => {
                loadStudents();
                loadClasses();
              }}
            />
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<StudentCreateModal 
  open={showCreateModal} 
  {readerGroups} 
  onclose={() => showCreateModal = false} 
  onsuccess={handleStudentCreated} 
/>

{#if showLusdModal}
  <LusdPreviewModal 
    open={showLusdModal} 
    file={lusdFile} 
    onclose={() => { showLusdModal = false; lusdFile = null; }} 
    onsuccess={onLusdSuccess} 
  />
{/if}
