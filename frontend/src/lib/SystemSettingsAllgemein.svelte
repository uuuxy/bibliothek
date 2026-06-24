<script>
  import { apiPost } from "./apiFetch.js";
  import { toastStore } from "./stores/toastStore.svelte.js";
  import SettingField from "./components/settings/SettingField.svelte";

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

<!-- Flach & edge-to-edge: logische Blöcke per Abstand + feiner Trennlinie statt Kacheln -->
<div class="space-y-10 max-w-3xl">

  {#snippet sectionHeader(title, description)}
    <div>
      <h3 class="text-base font-bold text-slate-900">{title}</h3>
      {#if description}
        <p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-2xl">{description}</p>
      {/if}
    </div>
  {/snippet}

  <!-- Ferien-Leseclub -->
  <section class="border-b border-slate-200 pb-8">
    <div class="flex items-start justify-between gap-4">
      {@render sectionHeader('Ferien-Leseclub', 'Wenn aktiv, erhalten alle neuen Ausleihen pauschal das unten definierte Ferienende als Rückgabefrist. Die Standardfristen werden überschrieben.')}
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
      <div class="mt-6 max-w-xs">
        <SettingField
          bind:value={ferienLeseclubZieldatum}
          label="Ferienende (Rückgabezieldatum)"
          type="date"
          hint="Alle Ausleihen erhalten dieses Datum als Rückgabefrist."
        />
      </div>
    {/if}
  </section>

  <!-- Rückgabefristen & Ausleih-Limits -->
  <section class="border-b border-slate-200 pb-8">
    {@render sectionHeader('Rückgabefristen & Ausleih-Limits', '')}
    <div class="mt-6 grid grid-cols-2 md:grid-cols-4 gap-x-8 gap-y-6">
      <SettingField bind:value={fristBuchTage} label="Tage / Buch" min={1} max={365} />
      <SettingField bind:value={fristMedienTage} label="Tage / Medien" min={1} max={365} />
      <SettingField bind:value={lmfStichtag} label="LMF (MM-TT)" type="text" placeholder="07-31" pattern={'\\d{2}-\\d{2}'} maxlength={5} />
      <SettingField bind:value={maxAusleihenSchueler} label="Max Ausleihen" min={1} max={50} />
    </div>
  </section>

  <!-- Sperr-Automatik -->
  <section class="border-b border-slate-200 pb-8">
    {@render sectionHeader('Sperr-Automatik (Mahnwesen)', 'Automatische Ausleihsperre am Kiosk für Schüler mit überfälligen Medien. Ausgenommen sind Geräte/Dauerleihen (z.B. Laptops).')}
    <div class="mt-6 grid grid-cols-2 gap-x-8 gap-y-6 max-w-md">
      <SettingField bind:value={maxOverdueDays} label="Tage bis Sperre" min={0} max={365} />
      <SettingField bind:value={maxOverdueItems} label="Ab x Medien sperren" min={1} max={50} />
    </div>
  </section>

  <div class="flex justify-end">
    <button
      onclick={saveSettings}
      disabled={saving}
      class="px-8 py-3 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-full transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed shadow-sm">
      {saving ? 'Wird gespeichert...' : 'Globale Einstellungen speichern'}
    </button>
  </div>
</div>
