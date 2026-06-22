<script>
  import { apiPost } from "./apiFetch.js";
  import { toastStore } from "./stores/toastStore.svelte.js";

  /**
   * @typedef {Object} Props
   * @property {boolean} ferienLeseclubAktiv
   * @property {string} ferienLeseclubZieldatum
   * @property {string} lmfStichtag
   * @property {number} maxAusleihenSchueler
   * @property {number} fristBuchTage
   * @property {number} fristMedienTage
   * @property {number} maxOverdueDays
   * @property {number} maxOverdueItems
   */
  
  /** @type {Props} */
  let { 
    ferienLeseclubAktiv = $bindable(), 
    ferienLeseclubZieldatum = $bindable(), 
    lmfStichtag = $bindable(), 
    maxAusleihenSchueler = $bindable(), 
    fristBuchTage = $bindable(), 
    fristMedienTage = $bindable(), 
    maxOverdueDays = $bindable(), 
    maxOverdueItems = $bindable() 
  } = $props();

  let saving = $state(false);

  async function saveSettings() {
    saving = true;
    try {
      await apiPost('/api/einstellungen', {
          ferien_leseclub_aktiv: ferienLeseclubAktiv,
          ferien_leseclub_zieldatum: ferienLeseclubZieldatum || null,
          lmf_stichtag: lmfStichtag || '07-31',
          max_ausleihen_schueler: maxAusleihenSchueler,
          frist_buch_tage: fristBuchTage,
          frist_medien_tage: fristMedienTage,
          max_overdue_days: maxOverdueDays,
          max_overdue_items: maxOverdueItems
      });
      toastStore.addToast('Einstellungen gespeichert.', 'success');
    } catch {
      // Toast already shown by apiPost
    }
    saving = false;
  }
</script>

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
        aria-label="Ferien-Leseclub umschalten"
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

  <!-- Sperr-Automatik Card -->
  <div class="bg-white rounded-[24px] p-8 shadow-sm border border-slate-200/70">
    <h3 class="text-base font-bold text-slate-900 mb-2">Sperr-Automatik (Mahnwesen)</h3>
    <p class="text-xs text-slate-500 mb-5 leading-relaxed">Automatische Ausleihsperre am Kiosk für Schüler mit überfälligen Medien. Ausgenommen sind Geräte/Dauerleihen (z.B. Laptops).</p>
    
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Toleranz-Tage -->
      <div class="rounded-2xl bg-rose-50 border border-rose-100 p-5 flex flex-col items-center justify-center">
        <input type="number" bind:value={maxOverdueDays} min="0" max="365" class="w-20 bg-white border border-rose-200 rounded-xl px-2 py-2 text-xl font-bold text-rose-600 text-center focus:border-rose-400 focus:ring-2 focus:ring-rose-100 focus:outline-none transition-all" />
        <div class="text-xs text-rose-500 mt-3 font-semibold uppercase tracking-wider">Tage bis Sperre</div>
      </div>

      <!-- Toleranz-Items -->
      <div class="rounded-2xl bg-rose-50 border border-rose-100 p-5 flex flex-col items-center justify-center">
        <input type="number" bind:value={maxOverdueItems} min="1" max="50" class="w-20 bg-white border border-rose-200 rounded-xl px-2 py-2 text-xl font-bold text-rose-600 text-center focus:border-rose-400 focus:ring-2 focus:ring-rose-100 focus:outline-none transition-all" />
        <div class="text-xs text-rose-500 mt-3 font-semibold uppercase tracking-wider">Ab x Medien sperren</div>
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
