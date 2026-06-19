<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { onMount } from 'svelte';
  import MailTemplates from './MailTemplates.svelte';
  import LitteraImportWidget from './LitteraImportWidget.svelte';
  import PermissionManager from './PermissionManager.svelte';

  // --- STATE ---
  let loading = $state(true);
  let saving = $state(false);
  /** @type {{msg: string, type: string} | null} */
  let toast = $state(null);

  // Tabs
  const tabs = ["Allgemein", "Team & Rechte", "Mahnwesen-Routing", "Daten-Import", "System"];
  let activeTab = $state("Allgemein");
  
  // Progressive Disclosure
  let showAdvancedImport = $state(false);

  // Global Settings (Allgemein)
  let ferienLeseclubAktiv = $state(false);
  let ferienLeseclubZieldatum = $state('');
  let lmfStichtag = $state('07-31');
  let maxAusleihenSchueler = $state(5);
  let fristBuchTage = $state(21);
  let fristMedienTage = $state(7);

  // Klassenlehrer-Mapping (Mahnwesen-Routing)
  /** @type {{klasse: string, lehrer_email: string}[]} */
  let mappingRows = $state([]);
  let mappingLoading = $state(false);
  let newMappingKlasse = $state('');
  let newMappingEmail = $state('');
  let mappingSaving = $state(false);

  // --- LOGIC ---
  /**
   * @param {string} msg
   * @param {string} [type]
   */
  function showToast(msg, type = 'success') {
    toast = { msg, type };
    setTimeout(() => { toast = null; }, 3500);
  }

  async function loadSettings() {
    try {
      const res = await apiClient.get('/api/einstellungen');
      if (res.ok) {
        const data = await res.json();
        ferienLeseclubAktiv = data.ferien_leseclub_aktiv ?? false;
        ferienLeseclubZieldatum = data.ferien_leseclub_zieldatum ?? '';
        lmfStichtag = data.lmf_stichtag ?? '07-31';
        maxAusleihenSchueler = data.max_ausleihen_schueler ?? 5;
        fristBuchTage = data.frist_buch_tage ?? 21;
        fristMedienTage = data.frist_medien_tage ?? 7;
      }
    } catch { /* use defaults */ }
  }

  async function fetchMapping() {
    mappingLoading = true;
    try {
      const res = await apiFetch('/api/klassen-mapping');
      if (res.ok) mappingRows = await res.json();
    } catch { /* ignore */ } finally {
      mappingLoading = false;
    }
  }

  onMount(async () => {
    await loadSettings();
    await fetchMapping();
    loading = false;
  });

  async function saveSettings() {
    saving = true;
    try {
      const res = await apiClient.post('/api/einstellungen', {
          ferien_leseclub_aktiv: ferienLeseclubAktiv,
          ferien_leseclub_zieldatum: ferienLeseclubZieldatum || null,
          lmf_stichtag: lmfStichtag || '07-31',
          max_ausleihen_schueler: maxAusleihenSchueler,
          frist_buch_tage: fristBuchTage,
          frist_medien_tage: fristMedienTage
        });
      if (res.ok) {
        showToast('Einstellungen gespeichert.');
      } else {
        showToast((await res.text()) || 'Fehler beim Speichern', 'error');
      }
    } catch {
      showToast('Netzwerkfehler', 'error');
    }
    saving = false;
  }

  async function upsertMapping() {
    if (!newMappingKlasse.trim() || !newMappingEmail.trim()) return;
    mappingSaving = true;
    try {
      const res = await apiClient.post('/api/klassen-mapping', { klasse: newMappingKlasse.trim(), lehrer_email: newMappingEmail.trim()
      });
      if (res.ok) {
        newMappingKlasse = '';
        newMappingEmail = '';
        await fetchMapping();
        showToast('Mapping gespeichert.');
      } else {
        showToast((await res.text()) || 'Fehler beim Speichern', 'error');
      }
    } catch {
      showToast('Netzwerkfehler', 'error');
    } finally {
      mappingSaving = false;
    }
  }

  /** @param {string} klasse */
  async function deleteMapping(klasse) {
    try {
      const res = await apiFetch(`/api/klassen-mapping/${encodeURIComponent(klasse)}`, { method: 'DELETE' });
      if (res.ok || res.status === 204) {
        await fetchMapping();
        showToast(`Mapping für ${klasse} gelöscht.`);
      } else {
        showToast('Fehler beim Löschen', 'error');
      }
    } catch {
      showToast('Netzwerkfehler', 'error');
    }
  }
</script>

<div class="max-w-6xl mx-auto w-full space-y-8 text-slate-800 font-sans antialiased pb-20">

  <!-- Header -->
  <div class="space-y-6">
    <div>
      <h2 class="text-3xl font-bold text-slate-900 tracking-tight">Einstellungen</h2>
      <p class="text-sm text-slate-500 mt-2">Globale Systemparameter, Nutzerverwaltung und Datenimporte.</p>
    </div>

    <!-- Tabs -->
    <div class="flex gap-4 border-b border-slate-200">
      {#each tabs as tab}
        <button
          onclick={() => (activeTab = tab)}
          class="relative px-2 py-3 text-sm font-semibold transition-colors focus:outline-none {activeTab === tab ? 'text-blue-600' : 'text-slate-500 hover:text-slate-800'}"
        >
          {tab}
          {#if activeTab === tab}
            <div class="absolute bottom-0 left-0 w-full h-1 bg-blue-600 rounded-t-full"></div>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="py-20 flex justify-center items-center">
      <div class="w-10 h-10 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else}
    
    <!-- Toasts -->
    {#if toast}
      <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in
        {toast.type === 'error' ? 'bg-rose-600 text-white' : 'bg-emerald-600 text-white'}">
        {toast.msg}
      </div>
    {/if}

    <!-- Tab Content -->
    <div class="pt-2 animate-fade-in">
      
      <!-- TAB: ALLGEMEIN -->
      {#if activeTab === 'Allgemein'}
        <div class="space-y-6">
          
          <!-- Ferien-Leseclub Card -->
          <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
            <div class="flex items-start justify-between gap-4">
              <div>
                <h3 class="text-base font-bold text-slate-900">Ferien-Leseclub</h3>
                <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">Wenn aktiv, erhalten alle neuen Ausleihen pauschal das unten definierte Ferienende als Rückgabefrist. Die Standardfristen werden überschrieben.</p>
              </div>
              <button
                type="button"
                onclick={() => (ferienLeseclubAktiv = !ferienLeseclubAktiv)}
                class="relative inline-flex h-8 w-14 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 {ferienLeseclubAktiv ? 'bg-emerald-500' : 'bg-slate-200'}"
                role="switch"
                aria-checked={ferienLeseclubAktiv}
              >
                <span class="pointer-events-none inline-block h-7 w-7 transform rounded-full bg-white shadow-sm transition duration-200 ease-in-out {ferienLeseclubAktiv ? 'translate-x-6' : 'translate-x-0'}"></span>
              </button>
            </div>

            {#if ferienLeseclubAktiv}
              <div class="mt-5 p-5 rounded-2xl bg-emerald-50 border border-emerald-100 space-y-3">
                <p class="text-xs font-bold text-emerald-700 flex items-center gap-1.5">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
                  Ferien-Leseclub ist aktiv
                </p>
                <div>
                  <label for="ferienZieldatum" class="text-xs font-semibold text-slate-600 block mb-1">Ferienende (Rückgabezieldatum)</label>
                  <input
                    id="ferienZieldatum"
                    type="date"
                    bind:value={ferienLeseclubZieldatum}
                    class="w-48 bg-white border border-emerald-200 rounded-xl px-4 py-2.5 text-sm focus:border-emerald-500 focus:ring-2 focus:ring-emerald-200 focus:outline-none text-slate-800 transition-all" />
                  <p class="text-[10px] text-slate-500 mt-1.5">Alle Ausleihen erhalten dieses Datum als Rückgabefrist.</p>
                </div>
              </div>
            {/if}
          </div>

          <!-- Standardfristen & LMF Card -->
          <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
            <h3 class="text-base font-bold text-slate-900 mb-5">Rückgabefristen & Ausleih-Limits</h3>
            
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              
              <!-- Buch Frist -->
              <div class="rounded-2xl bg-slate-50 border border-slate-100 p-5 flex flex-col items-center justify-center">
                <input type="number" bind:value={fristBuchTage} min="1" max="365" class="w-20 bg-white border border-slate-200 rounded-xl px-2 py-2 text-xl font-bold text-slate-700 text-center focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none transition-all" />
                <div class="text-xs text-slate-500 mt-3 font-semibold uppercase tracking-wider">Tage / Buch</div>
              </div>

              <!-- Medien Frist -->
              <div class="rounded-2xl bg-amber-50 border border-amber-100 p-5 flex flex-col items-center justify-center">
                <input type="number" bind:value={fristMedienTage} min="1" max="365" class="w-20 bg-white border border-amber-200 rounded-xl px-2 py-2 text-xl font-bold text-amber-600 text-center focus:border-amber-400 focus:ring-2 focus:ring-amber-100 focus:outline-none transition-all" />
                <div class="text-xs text-amber-500 mt-3 font-semibold uppercase tracking-wider">Tage / Medien</div>
              </div>

              <!-- LMF Stichtag -->
              <div class="rounded-2xl bg-blue-50 border border-blue-100 p-5 flex flex-col items-center justify-center">
                <input
                  type="text"
                  bind:value={lmfStichtag}
                  placeholder="07-31"
                  pattern="\d{2}-\d{2}"
                  maxlength="5"
                  class="w-20 bg-white border border-blue-200 rounded-xl px-2 py-2 text-lg font-bold text-blue-600 font-mono text-center focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none transition-all" />
                <div class="text-xs text-blue-500 mt-3 font-semibold uppercase tracking-wider">LMF (MM-TT)</div>
              </div>

              <!-- Max Ausleihen -->
              <div class="rounded-2xl bg-purple-50 border border-purple-100 p-5 flex flex-col items-center justify-center">
                <input
                  type="number"
                  min="1"
                  max="50"
                  bind:value={maxAusleihenSchueler}
                  class="w-20 bg-white border border-purple-200 rounded-xl px-2 py-2 text-xl font-bold text-purple-600 text-center focus:border-purple-400 focus:ring-2 focus:ring-purple-100 focus:outline-none transition-all" />
                <div class="text-xs text-purple-500 mt-3 font-semibold uppercase tracking-wider">Max Ausleihen</div>
              </div>

            </div>
          </div>

          <div class="flex justify-end pt-4 pb-4">
            <button
              onclick={saveSettings}
              disabled={saving}
              class="px-8 py-3.5 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-2xl transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-md">
              {saving ? 'Wird gespeichert...' : 'Globale Einstellungen speichern'}
            </button>
          </div>
        </div>

      <!-- TAB: TEAM & RECHTE -->
      {:else if activeTab === 'Team & Rechte'}
        <div class="space-y-6">
          <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
            <h3 class="text-xl font-bold text-slate-900 mb-6">Account- und Rollenverwaltung</h3>
            <PermissionManager />
          </div>
        </div>

      <!-- TAB: MAHNWESEN-ROUTING -->
      {:else if activeTab === 'Mahnwesen-Routing'}
        <div class="space-y-6">
          
          <!-- Klassenlehrer-Mapping Card -->
          <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
            <div class="mb-6">
              <h3 class="text-xl font-bold text-slate-900">E-Mail Routing für Mahnungen</h3>
              <p class="text-sm text-slate-500 mt-1">Ordnet jeder Klasse eine E-Mail-Adresse zu. Wird im Mahnwesen als Empfänger für Benachrichtigungen vorausgefüllt.</p>
            </div>

            {#if mappingLoading}
              <div class="py-8 flex justify-center"><div class="w-8 h-8 border-4 border-slate-400 border-t-transparent rounded-full animate-spin"></div></div>
            {:else if mappingRows.length === 0}
              <div class="p-6 bg-slate-50 rounded-2xl border border-slate-100 text-center">
                <p class="text-sm text-slate-500 font-medium">Noch keine Mappings vorhanden.</p>
              </div>
            {:else}
              <div class="rounded-2xl border border-slate-200 overflow-hidden mb-6">
                <table class="w-full text-sm">
                  <thead>
                    <tr class="bg-slate-50 border-b border-slate-200 text-xs font-bold text-slate-500 uppercase tracking-wider">
                      <th class="text-left px-5 py-3">Klasse</th>
                      <th class="text-left px-5 py-3">Lehrer-E-Mail</th>
                      <th class="px-5 py-3 text-right">Aktion</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-100">
                    {#each mappingRows as row}
                      <tr class="hover:bg-slate-50 transition-colors">
                        <td class="px-5 py-3 font-semibold text-slate-800">{row.klasse}</td>
                        <td class="px-5 py-3 text-slate-600">{row.lehrer_email}</td>
                        <td class="px-5 py-3 text-right">
                          <button onclick={() => deleteMapping(row.klasse)} class="p-2 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors cursor-pointer" title="Mapping löschen">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                          </button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}

            <!-- Add new mapping -->
            <div class="p-5 bg-slate-50 rounded-2xl border border-slate-200 flex flex-col md:flex-row items-end gap-4">
              <div class="w-full md:w-32">
                <label for="newMappingKlasseInput" class="text-[10px] font-bold text-slate-500 uppercase tracking-wider block mb-1.5">Klasse</label>
                <input id="newMappingKlasseInput" type="text" bind:value={newMappingKlasse} placeholder="z.B. 7a" class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all" />
              </div>
              <div class="flex-1 w-full">
                <label for="newMappingEmailInput" class="text-[10px] font-bold text-slate-500 uppercase tracking-wider block mb-1.5">E-Mail</label>
                <input id="newMappingEmailInput" type="email" bind:value={newMappingEmail} placeholder="lehrkraft@schule.de" class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all" />
              </div>
              <button
                onclick={upsertMapping}
                disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
                class="w-full md:w-auto px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap shadow-sm">
                {mappingSaving ? 'Lädt…' : 'Hinzufügen'}
              </button>
            </div>
          </div>

        </div>

      <!-- TAB: DATEN-IMPORT -->
      {:else if activeTab === 'Daten-Import'}
        
        <!-- Littera Import Card -->
        <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
          <div class="mb-8">
            <h3 class="text-xl font-bold text-slate-900">Littera CSV-Import</h3>
            <p class="text-sm text-slate-500 mt-1">Importiere oder aktualisiere Schüler- und Buchdaten aus einer bestehenden Littera-Installation.</p>
          </div>

          <div class="grid grid-cols-1 gap-8">
            <div class="col-span-1">
              <LitteraImportWidget />
            </div>

            <!-- Progressive Disclosure -->
            <div class="pt-6 border-t border-slate-100">
              <button 
                class="flex items-center gap-2 text-sm font-semibold text-slate-600 hover:text-blue-600 transition-colors cursor-pointer w-full text-left focus:outline-none"
                onclick={() => (showAdvancedImport = !showAdvancedImport)}
              >
                <svg class="w-5 h-5 transition-transform duration-200 {showAdvancedImport ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                Erweiterte Experten-Einstellungen anzeigen
              </button>

              {#if showAdvancedImport}
                <div class="mt-6 p-6 bg-slate-50 rounded-2xl border border-slate-200 space-y-6 animate-fade-in">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div>
                      <label class="block text-xs font-bold text-slate-700 uppercase tracking-wide mb-2">Zeichensatz (Encoding)</label>
                      <select class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all cursor-pointer">
                        <option>UTF-8 (Empfohlen)</option>
                        <option>ISO-8859-1 (Windows-Standard)</option>
                      </select>
                    </div>
                    <div>
                      <label class="block text-xs font-bold text-slate-700 uppercase tracking-wide mb-2">CSV Trennzeichen</label>
                      <input type="text" value=";" class="w-full bg-white border border-slate-200 rounded-xl px-4 py-2.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-200 outline-none transition-all font-mono" />
                    </div>
                  </div>
                </div>
              {/if}
            </div>
          </div>
        </div>

      <!-- TAB: SYSTEM -->
      {:else if activeTab === 'System'}
        <div class="space-y-6">
          
          <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
            <h3 class="text-xl font-bold text-slate-900 mb-6">Mail-Templates</h3>
            <MailTemplates />
          </div>

        </div>
      {/if}
      
    </div>
  {/if}
</div>
