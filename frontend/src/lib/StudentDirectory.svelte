<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";
  import Modal from "./Modal.svelte";

  // Props (Svelte 5)
  let { role = "" } = $props();

  // State Runes (Svelte 5)
  /** @type {any[]} */
  let students = $state.raw([]);
  let loading = $state(false);
  let searchQuery = $state("");
  /** @type {any} */
  let activeStudent = $state(null);

  /** @type {string[]} */
  let existingClasses = $state.raw([]);
  let showCreateModal = $state(false);
  let newVorname = $state("");
  let newNachname = $state("");
  let newKlasse = $state("");
  let customKlasseInput = $state(false);
  let newBarcode = $state("");
  let createError = $state("");
  let isSaving = $state(false);

  let isImporting = $state(false);
  let importStatusMessage = $state("");
  let importErrorMessage = $state("");
  /** @type {HTMLInputElement | null} */
  let fileInputEl = $state(null);

  function triggerImportPicker() {
    importStatusMessage = "";
    importErrorMessage = "";
    fileInputEl?.click();
  }

  /** @param {Event} event */
  async function handleLUSDImport(event) {
    const target = /** @type {HTMLInputElement} */ (event.target);
    const file = target.files?.[0];
    if (!file) return;

    isImporting = true;
    importStatusMessage = "";
    importErrorMessage = "";

    const formData = new FormData();
    formData.append("file", file);

    try {
      const res = await apiFetch("/api/students/import", {
        method: "POST",
        body: formData
      });
      if (res.ok) {
        const data = await res.json();
        importStatusMessage = `${data.imported} Schüler erfolgreich importiert/aktualisiert.`;
        await loadStudents(); // Reload table
        await loadClasses(); // Reload classes
      } else {
        const errText = await res.text();
        try {
          const errObj = JSON.parse(errText);
          importErrorMessage = `${errObj.error || "Fehler beim Verarbeiten der CSV."}`;
        } catch {
          importErrorMessage = `${errText || "Unerwarteter Server-Fehler."}`;
        }
      }
    } catch (err) {
      importErrorMessage = "Netzwerkfehler beim Hochladen der Importdatei.";
      console.error(err);
    } finally {
      isImporting = false;
      target.value = "";
    }
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
      const res = await apiFetch("/api/klassen");
      if (res.ok) {
        existingClasses = await res.json();
      }
    } catch (err) {
      console.error("Fehler beim Laden der Klassen:", err);
    }
  }

  async function createStudent() {
    createError = "";
    if (!newVorname.trim() || !newNachname.trim() || !newKlasse.trim()) {
      createError = "Vorname, Nachname und Klasse sind Pflichtfelder.";
      return;
    }
    isSaving = true;
    try {
      const res = await apiFetch("/api/schueler", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          vorname: newVorname.trim(),
          nachname: newNachname.trim(),
          klasse: newKlasse.trim(),
          barcode_id: newBarcode.trim()
        })
      });
      if (res.ok) {
        showCreateModal = false;
        newVorname = "";
        newNachname = "";
        newKlasse = "";
        newBarcode = "";
        customKlasseInput = false;
        await loadStudents();
        await loadClasses(); // refresh class list
      } else {
        const errText = await res.text();
        try {
          const errObj = JSON.parse(errText);
          createError = errObj.error || "Fehler beim Anlegen des Schülers.";
        } catch {
          createError = errText || "Fehler beim Anlegen des Schülers.";
        }
      }
    } catch (err) {
      createError = "Netzwerkfehler beim Anlegen des Schülers.";
      console.error(err);
    } finally {
      isSaving = false;
    }
  }

  onMount(() => {
    loadStudents();
    loadClasses();
  });
</script>

<div class="w-full animate-fade-in text-slate-800">
  
  {#if activeStudent}
    <!-- Detail View: Student Profile -->
    <!-- Back button is no-print; profile's print section must not be inside no-print -->
    <div class="w-full text-left space-y-4">
      <div class="no-print">
        <button onclick={() => { activeStudent = null; loadStudents(); }} class="inline-flex items-center gap-2 text-xs font-bold text-blue-600 hover:text-blue-750 transition-colors py-1 cursor-pointer">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
          <span>Zurück zum Schülerverzeichnis</span>
        </button>
      </div>
      <StudentProfile student={activeStudent} onDeselect={() => { activeStudent = null; loadStudents(); }} role={role} />
    </div>
  {:else}
    <!-- Fullscreen Directory List -->
    <div class="w-full space-y-6 no-print">
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

      {#snippet avatar(/** @type {any} */ s)}
        <div class="relative w-8 h-8 rounded-full overflow-hidden border border-slate-100/80 bg-slate-50 flex items-center justify-center shrink-0">
          {#if s.foto_url}
            <img src={s.foto_url} alt="Passbild von {s.vorname} {s.nachname}" class="w-full h-full object-cover" />
          {:else}
            <div class="w-full h-full flex items-center justify-center bg-slate-100 text-slate-500 font-bold text-xs uppercase" aria-hidden="true">
              {s.vorname.charAt(0)}{s.nachname.charAt(0)}
            </div>
          {/if}
        </div>
      {/snippet}

      {#snippet statusBadge(/** @type {any} */ s)}
        <div class="inline-flex items-center justify-end gap-1.5 py-1">
          {#if s.ueberfaellig_count > 0}
            <span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse" aria-hidden="true"></span>
            <span class="text-xs font-semibold text-rose-600">Überfällig</span>
          {:else if s.ist_gesperrt}
            <span class="w-1.5 h-1.5 rounded-full bg-amber-500" aria-hidden="true"></span>
            <span class="text-xs font-semibold text-amber-600">Gesperrt</span>
          {:else}
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
            <span class="text-xs font-semibold text-emerald-600">Alles ok</span>
          {/if}
        </div>
      {/snippet}

      <!-- Table Container -->
      <div class="bg-white rounded-2xl border border-slate-100 overflow-hidden shadow-xs w-full">
        {#if loading}
          <div class="py-16 flex justify-center items-center">
            <div class="w-8 h-8 border-4 border-t-blue-600 border-slate-200 rounded-full animate-spin" aria-hidden="true"></div>
          </div>
        {:else if filteredStudents.length === 0}
          <div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            <span class="text-xs font-semibold">Keine Schüler im Verzeichnis gefunden.</span>
          </div>
        {:else}
          <div class="overflow-x-auto w-full text-left">
            <table class="w-full text-base text-slate-700">
              <thead class="bg-slate-50 border-b border-slate-100 uppercase tracking-wider text-sm font-bold text-slate-500 font-sans">
                <tr>
                  <th class="px-6 py-4 w-16">Foto</th>
                  <th class="px-6 py-4">Name</th>
                  <th class="px-6 py-4 w-24">Klasse</th>
                  <th class="px-6 py-4 w-44 text-right">Geliehene Bücher</th>
                  <th class="px-6 py-4 w-36 text-right">Status</th>
                  <th class="px-6 py-4 w-10"></th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                {#each filteredStudents as s}
                  <tr 
                    onclick={() => activeStudent = s} 
                    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); activeStudent = s; } }}
                    tabindex="0"
                    role="button"
                    aria-label="Profil von {s.vorname} {s.nachname} (Klasse {s.klasse || 'N/A'}) anzeigen"
                    class="hover:bg-slate-50/50 cursor-pointer transition-colors group focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
                  >
                    <!-- Photo Avatar -->
                    <td class="px-6 py-3">
                      {@render avatar(s)}
                    </td>
                    
                    <!-- Name & Barcode ID -->
                    <td class="px-6 py-3 font-semibold text-slate-800">
                      {s.vorname} {s.nachname}
                      <div class="text-[9px] text-slate-400 font-normal mt-0.5">{s.barcode_id}</div>
                    </td>
                    
                    <!-- Klasse -->
                    <td class="px-6 py-3 font-medium text-slate-600">
                      Kl. {s.klasse || 'N/A'}
                    </td>
                    
                    <!-- Books count -->
                    <td class="px-6 py-3 text-right">
                      <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold {s.ausgeliehen_count > 0 ? 'bg-blue-50 text-blue-700' : 'bg-slate-100 text-slate-500'}">
                        {s.ausgeliehen_count || 0}
                      </span>
                    </td>
                    
                    <!-- Status -->
                    <td class="px-6 py-3 text-right">
                      {@render statusBadge(s)}
                    </td>
                    
                    <!-- Arrow link -->
                    <td class="px-6 py-3 text-right">
                      <svg class="w-4 h-4 text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                      </svg>
                    </td>
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

<Modal open={showCreateModal} onclose={() => { showCreateModal = false; createError = ""; }} size="md">
  {#snippet header()}
    <h3 class="text-sm font-bold text-slate-800">Neuen Schüler anlegen</h3>
  {/snippet}
  {#snippet children()}
    <div class="p-6 space-y-4">
      {#if createError}
        <div class="p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600">
          {createError}
        </div>
      {/if}

      <label class="block text-xs font-bold uppercase tracking-wider text-slate-400">Vorname *
        <input type="text" bind:value={newVorname} placeholder="z.B. Max" class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans" />
      </label>

      <label class="block text-xs font-bold uppercase tracking-wider text-slate-400">Nachname *
        <input type="text" bind:value={newNachname} placeholder="z.B. Mustermann" class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans" />
      </label>

      <label class="block text-xs font-bold uppercase tracking-wider text-slate-400">Klasse *
        <div class="mt-1.5 flex gap-2">
          {#if !customKlasseInput}
            <select bind:value={newKlasse} onchange={(e) => { const sel = /** @type {HTMLSelectElement} */ (e.target); if (sel && sel.value === "__custom__") { customKlasseInput = true; newKlasse = ""; } }} class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all cursor-pointer font-sans">
              <option value="">-- Klasse auswählen --</option>
              {#each existingClasses as k}
                <option value={k}>{k}</option>
              {/each}
              <option value="__custom__">Neue Klasse eingeben...</option>
            </select>
          {:else}
            <div class="relative w-full">
              <input type="text" bind:value={newKlasse} placeholder="z.B. 10b" class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans" />
              <button type="button" onclick={() => { customKlasseInput = false; newKlasse = ""; }} class="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-semibold text-blue-600 hover:text-blue-750 transition-colors bg-transparent border-none cursor-pointer">Auswahl</button>
            </div>
          {/if}
        </div>
      </label>

      <label class="block text-xs font-bold uppercase tracking-wider text-slate-400">Barcode-ID (optional)
        <input type="text" bind:value={newBarcode} placeholder="Wird automatisch generiert, wenn leer" class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all" />
      </label>

      <div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
        <button onclick={() => { showCreateModal = false; createError = ""; }} disabled={isSaving} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer font-sans">Abbrechen</button>
        <button onclick={createStudent} disabled={isSaving} class="rounded-xl bg-blue-600 px-4 py-2 text-sm font-bold text-white hover:bg-blue-750 disabled:opacity-60 transition-colors cursor-pointer font-sans">
          {#if isSaving}Speichern...{:else}Speichern{/if}
        </button>
      </div>
    </div>
  {/snippet}
</Modal>
