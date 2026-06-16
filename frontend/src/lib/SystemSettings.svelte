<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import { onMount } from 'svelte';
  import MailTemplates from './MailTemplates.svelte';
  import LitteraImportWidget from './LitteraImportWidget.svelte';
  import GlobalLMFExtendWidget from './GlobalLMFExtendWidget.svelte';
  import SignatureManager from './SignatureManager.svelte';

  let loading = $state(true);
  let saving = $state(false);
  /** @type {{msg: string, type: string} | null} */
  let toast = $state(null);

  let ferienLeseclubAktiv = $state(false);
  let ferienLeseclubZieldatum = $state('');
  let lmfStichtag = $state('07-31');
  let maxAusleihenSchueler = $state(5);
  let fristBuchTage = $state(21);
  let fristMedienTage = $state(7);

  // Klassenlehrer-Mapping
  /** @type {{klasse: string, lehrer_email: string}[]} */
  let mappingRows = $state([]);
  let mappingLoading = $state(false);
  let newMappingKlasse = $state('');
  let newMappingEmail = $state('');
  let mappingSaving = $state(false);

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

<div class="w-full space-y-6 text-slate-800 font-sans antialiased">

  <!-- Page Header -->
  <div class="border-b border-slate-100 pb-5">
    <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Administration</span>
    <h2 class="text-2xl font-bold text-slate-900">Globale Einstellungen</h2>
    <p class="text-xs text-slate-500 font-medium mt-1">Systemweite Parameter für den Schuljahresbetrieb. Nur für Admins sichtbar.</p>
  </div>

  {#if loading}
    <div class="py-16 flex justify-center items-center">
      <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else}

    <!-- Toast notification -->
    {#if toast}
      <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in
        {toast.type === 'error' ? 'bg-rose-600 text-white' : 'bg-emerald-600 text-white'}">
        {toast.msg}
      </div>
    {/if}

    <!-- Ferien-Leseclub Card -->
    <div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <h3 class="text-base font-bold text-slate-900">Ferien-Leseclub</h3>
          <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-lg">Wenn aktiv, erhalten alle neuen Ausleihen pauschal das unten definierte Ferienende als Rückgabefrist. Die Standardfristen werden überschrieben.</p>
        </div>
        <!-- Toggle -->
        <button
          type="button"
          onclick={() => (ferienLeseclubAktiv = !ferienLeseclubAktiv)}
          class="relative inline-flex h-7 w-12 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 focus:outline-none {ferienLeseclubAktiv ? 'bg-emerald-500' : 'bg-slate-200'}"
          role="switch"
          aria-checked={ferienLeseclubAktiv}
          aria-label="Ferien-Leseclub aktivieren">
          <span class="pointer-events-none inline-block h-6 w-6 rounded-full bg-white shadow transition-transform duration-200 {ferienLeseclubAktiv ? 'translate-x-5' : 'translate-x-0'}"></span>
        </button>
      </div>

      {#if ferienLeseclubAktiv}
        <div class="p-4 rounded-2xl bg-emerald-50 border border-emerald-100 space-y-3">
          <p class="text-xs font-bold text-emerald-700 flex items-center gap-1.5">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
            Ferien-Leseclub ist aktiv
          </p>
          <div>
            <label for="ferienZieldatum" class="text-xs font-semibold text-slate-600 block mb-1">Ferienende (Rückgabezieldatum)</label>
            <input
              id="ferienZieldatum"
              type="date"
              bind:value={ferienLeseclubZieldatum}
              class="w-48 bg-white border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-emerald-400 focus:ring-2 focus:ring-emerald-100 focus:outline-none text-slate-800" />
            <p class="text-[10px] text-slate-400 mt-1">Alle Ausleihen erhalten dieses Datum als Rückgabefrist.</p>
          </div>
        </div>
      {/if}
    </div>

    <!-- LMF-Stichtag Card -->
    <div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
      <div>
        <h3 class="text-base font-bold text-slate-900">LMF-Rückgabe-Stichtag</h3>
        <p class="text-xs text-slate-500 mt-1">LMF-Bücher (Lehrmittelfreiheit) erhalten dieses Datum als Rückgabefrist. Format: <code class="bg-slate-100 px-1 rounded text-slate-700">MM-TT</code></p>
      </div>
      <div class="flex items-center gap-4">
        <div>
          <label for="lmfStichtagInput" class="text-xs font-semibold text-slate-600 block mb-1">Stichtag (MM-TT)</label>
          <input
            id="lmfStichtagInput"
            type="text"
            bind:value={lmfStichtag}
            placeholder="07-31"
            pattern="\d{2}-\d{2}"
            maxlength="5"
            class="w-32 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800 font-mono" />
          <p class="text-[10px] text-slate-400 mt-1">z. B. <code>07-31</code> für 31. Juli</p>
        </div>
      </div>
    </div>

    <!-- Standardfristen Card -->
    <div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
      <h3 class="text-base font-bold text-slate-900">Standardfristen (Übersicht)</h3>
      <div class="grid grid-cols-3 gap-3">
        <div class="rounded-2xl bg-slate-50 border border-slate-100 p-4 flex flex-col items-center justify-center">
          <input type="number" bind:value={fristBuchTage} min="1" max="365" class="w-16 bg-white border border-slate-200 rounded-xl px-2 py-1 text-xl font-bold text-slate-700 text-center focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none" />
          <div class="text-xs text-slate-500 mt-2 font-medium">Tage · Buch</div>
        </div>
        <div class="rounded-2xl bg-amber-50 border border-amber-100 p-4 flex flex-col items-center justify-center">
          <input type="number" bind:value={fristMedienTage} min="1" max="365" class="w-16 bg-white border border-amber-200 rounded-xl px-2 py-1 text-xl font-bold text-amber-600 text-center focus:border-amber-400 focus:ring-2 focus:ring-amber-100 focus:outline-none" />
          <div class="text-xs text-amber-500 mt-2 font-medium">Tage · CD / DVD</div>
        </div>
        <div class="rounded-2xl bg-blue-50 border border-blue-100 p-4 text-center">
          <div class="text-sm font-bold text-blue-600 font-mono">{lmfStichtag || '07-31'}</div>
          <div class="text-xs text-blue-500 mt-1 font-medium">Datum · LMF</div>
        </div>
      </div>
    </div>

    <!-- Max Ausleihen Card -->
    <div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
      <div>
        <h3 class="text-base font-bold text-slate-900">Maximale Ausleihen pro Schüler</h3>
        <p class="text-xs text-slate-500 mt-1">Schüler können gleichzeitig nicht mehr als diese Anzahl an Büchern ausleihen.</p>
      </div>
      <div>
        <label for="maxAusleihenInput" class="text-xs font-semibold text-slate-600 block mb-1">Maximale Anzahl</label>
        <input
          id="maxAusleihenInput"
          type="number"
          min="1"
          max="50"
          bind:value={maxAusleihenSchueler}
          class="w-24 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800 text-center font-semibold" />
      </div>
    </div>

    <!-- Klassenlehrer-Mapping Card -->
    <div class="p-6 rounded-3xl bg-white border border-slate-100 shadow-xs space-y-4">
      <div>
        <h3 class="text-base font-bold text-slate-900">Klassenlehrer-Mapping</h3>
        <p class="text-xs text-slate-500 mt-1">Ordnet jeder Klasse eine E-Mail-Adresse zu. Wird im Mahnwesen als Empfänger vorausgefüllt.</p>
      </div>

      {#if mappingLoading}
        <div class="py-4 flex justify-center"><div class="w-5 h-5 border-2 border-slate-400 border-t-transparent rounded-full animate-spin"></div></div>
      {:else if mappingRows.length === 0}
        <p class="text-xs text-slate-400 font-medium py-2">Noch keine Mappings vorhanden.</p>
      {:else}
        <div class="rounded-2xl border border-slate-100 overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-500 uppercase tracking-wider">
                <th class="text-left px-4 py-2.5">Klasse</th>
                <th class="text-left px-4 py-2.5">Lehrer-E-Mail</th>
                <th class="px-4 py-2.5"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-50">
              {#each mappingRows as row}
                <tr class="hover:bg-slate-50 transition-colors">
                  <td class="px-4 py-2.5 font-semibold text-slate-700">{row.klasse}</td>
                  <td class="px-4 py-2.5 text-slate-600 text-xs">{row.lehrer_email}</td>
                  <td class="px-4 py-2.5 text-right">
                    <button onclick={() => deleteMapping(row.klasse)} class="p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer" title="Mapping löschen">
                      🗑️
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      <!-- Add new mapping -->
      <div class="flex items-end gap-3 pt-2">
        <div>
          <label for="newMappingKlasseInput" class="text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-1">Klasse</label>
          <input id="newMappingKlasseInput" type="text" bind:value={newMappingKlasse} placeholder="z. B. 7a" class="w-24 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800" />
        </div>
        <div class="flex-1">
          <label for="newMappingEmailInput" class="text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-1">E-Mail</label>
          <input id="newMappingEmailInput" type="email" bind:value={newMappingEmail} placeholder="lehrkraft@schule.de" class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none text-slate-800" />
        </div>
        <button
          onclick={upsertMapping}
          disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
          class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-bold text-sm rounded-xl transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap">
          {mappingSaving ? 'Lädt…' : 'Hinzufügen'}
        </button>
      </div>
    </div>

    <!-- Save Button -->
    <div class="flex justify-end pt-2 pb-4">
      <button
        onclick={saveSettings}
        disabled={saving}
        class="px-8 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-2xl transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-sm">
        {saving ? 'Wird gespeichert...' : 'Einstellungen speichern'}
      </button>
    </div>

    <!-- Mail Templates Component -->
    <MailTemplates />

    <!-- Global LMF Extend Widget -->
    <GlobalLMFExtendWidget />

    <!-- Signaturen Master Data Management -->
    <SignatureManager />

    <!-- Littera Import Component -->
    <div class="pt-6">
      <LitteraImportWidget />
    </div>
  {/if}
</div>
