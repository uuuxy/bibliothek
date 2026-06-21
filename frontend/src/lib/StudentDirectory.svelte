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
  import StudentDirectoryToolbar from "./components/students/StudentDirectoryToolbar.svelte";

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
  let showLusdModal = $state(false);
  let lusdFile = $state(/** @type {File|null} */ (null));

  function handleLUSDImport(/** @type {File} */ file) {
    lusdFile = file;
    showLusdModal = true;
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
        {#snippet tabButton(id, label, activeColorClass)}
          <button 
            onclick={() => activeTab = id}
            class="pb-3 text-sm font-semibold transition-colors border-b-2 {activeTab === id ? activeColorClass : 'border-transparent text-slate-500 hover:text-slate-800'}"
          >
            {label}
          </button>
        {/snippet}

        {@render tabButton("active", "Aktive Schüler", "border-blue-600 text-blue-700")}
        {@render tabButton("graduates", "Abgänger / Archiv", "border-blue-600 text-blue-700")}
        {#if role === 'admin'}
          {@render tabButton("deleted", "Papierkorb", "border-rose-600 text-rose-700")}
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
      <StudentDirectoryToolbar 
        bind:searchQuery 
        {role} 
        totalCount={students.length} 
        filteredCount={filteredStudents.length} 
        oncreate={() => showCreateModal = true} 
        onimport={handleLUSDImport} 
        onprint={() => showPrintStation = true} 
      />

      {#snippet alertBanner(type, bg, border, textClass, message, onClose)}
        <div class="p-4 {bg} border {border} rounded-2xl flex items-center justify-between text-xs font-semibold {textClass}">
          <div class="flex items-center gap-2">
            {#if type === 'loading'}
              <div class="w-4 h-4 border-2 border-t-blue-600 border-blue-250 rounded-full animate-spin mr-1" aria-hidden="true"></div>
            {:else if type === 'success'}
              <span>✅</span>
            {:else if type === 'error'}
              <span>⚠️</span>
            {/if}
            <span>{message}</span>
          </div>
          {#if onClose}
            <button onclick={onClose} aria-label="Schließen" class="text-current opacity-70 hover:opacity-100 font-bold text-sm bg-transparent border-none cursor-pointer">✕</button>
          {/if}
        </div>
      {/snippet}

      <!-- Import Status Alert Banner -->
      {#if isImporting}
        {@render alertBanner('loading', 'bg-blue-55 animate-pulse', 'border-blue-100', 'text-blue-700', 'LUSD-Schülerdaten werden importiert und abgeglichen... Bitte warten.', null)}
      {/if}

      {#if importStatusMessage}
        {@render alertBanner('success', 'bg-emerald-50', 'border-emerald-100', 'text-emerald-800', importStatusMessage, () => importStatusMessage = "")}
      {/if}

      {#if importErrorMessage}
        {@render alertBanner('error', 'bg-rose-50', 'border-rose-100', 'text-rose-800', importErrorMessage, () => importErrorMessage = "")}
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
