<script>
  import { apiClient } from './apiFetch.js';

  /**
   * @type {{
   *   student: any,
   *   onClose: () => void,
   *   onSave: () => void,
   *   role?: string
   * }}
   */
  let { student, onClose, onSave, role = '' } = $props();

  let saving = $state(false);

  /** @type {{ msg: string, type: 'success' | 'error' } | null} */
  let snackbar = $state(null);
  /** @type {ReturnType<typeof setTimeout> | null} */
  let snackbarTimer = null;

  let formData = $state({
    vorname: '',
    nachname: '',
    geburtsdatum: '',
    lusd_id: '',
    klasse: '',
    barcode_id: '',
    abgaenger_jahr: '',
    status: '',
    strasse: '',
    hausnummer: '',
    plz: '',
    ort: '',
    eltern_email: '',
  });

  $effect(() => {
    if (student) {
      formData.vorname      = student.vorname       || '';
      formData.nachname     = student.nachname      || '';
      formData.geburtsdatum = student.geburtsdatum  ? student.geburtsdatum.slice(0, 10) : '';
      formData.lusd_id      = student.lusd_id       || '';
      formData.klasse       = student.klasse        || '';
      formData.barcode_id   = student.barcode_id    || '';
      formData.abgaenger_jahr = student.abgaenger_jahr?.toString() || '';
      formData.status       = student.status        || '';
      formData.strasse      = student.strasse       || '';
      formData.hausnummer   = student.hausnummer    || '';
      formData.plz          = student.plz           || '';
      formData.ort          = student.ort           || '';
      formData.eltern_email = student.eltern_email  || '';
    }
  });

  /**
   * Show a self-dismissing snackbar.
   * @param {string} msg
   * @param {'success'|'error'} [type]
   */
  function showSnackbar(msg, type = 'success') {
    if (snackbarTimer) clearTimeout(snackbarTimer);
    snackbar = { msg, type };
    snackbarTimer = setTimeout(() => { snackbar = null; }, 3000);
  }

  async function save() {
    saving = true;
    try {
      const payload = {
        vorname:        formData.vorname        || null,
        nachname:       formData.nachname       || null,
        geburtsdatum:   formData.geburtsdatum   || null,
        lusd_id:        formData.lusd_id        || null,
        klasse:         formData.klasse         || null,
        barcode_id:     formData.barcode_id     || null,
        abgaenger_jahr: formData.abgaenger_jahr ? parseInt(formData.abgaenger_jahr, 10) : null,
        status:         formData.status         || null,
        strasse:        formData.strasse        || null,
        hausnummer:     formData.hausnummer     || null,
        plz:            formData.plz            || null,
        ort:            formData.ort            || null,
        eltern_email:   formData.eltern_email   || null,
      };
      const res = await apiClient.patch(`/api/schueler/${student.id}`, payload);
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Speichern fehlgeschlagen');
      }
      showSnackbar('Änderungen gespeichert.');
      onSave();
    } catch (/** @type {any} */ e) {
      showSnackbar(e?.message || String(e), 'error');
    } finally {
      saving = false;
    }
  }
</script>

<!-- Snackbar (floating, self-dismissing, layout-neutral) -->
{#if snackbar}
  <div
    class="fixed bottom-8 left-1/2 -translate-x-1/2 z-200
           flex items-center gap-3 px-5 py-3.5
           rounded-2xl shadow-2xl
           text-sm font-semibold
           animate-fade-in
           {snackbar.type === 'error'
             ? 'bg-rose-700 text-white'
             : 'bg-slate-900 text-white'}"
  >
    {#if snackbar.type === 'error'}
      <svg class="w-4 h-4 shrink-0 text-rose-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
    {:else}
      <svg class="w-4 h-4 shrink-0 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
      </svg>
    {/if}
    {snackbar.msg}
  </div>
{/if}

<!-- Full Page View (Replaces the side sheet) -->
<div class="w-full h-full bg-white flex flex-col animate-fade-in">

  <!-- ── Header ─────────────────────────────────────────────────────────── -->
  <header class="shrink-0 flex items-center justify-between gap-4 px-8 py-5 border-b border-slate-100">
    <div class="flex items-center gap-4 min-w-0">
      <!-- Back Button -->
      <button
        onclick={onClose}
        aria-label="Zurück"
        class="w-10 h-10 shrink-0 flex items-center justify-center rounded-xl bg-slate-50
               text-slate-500 hover:text-slate-800 hover:bg-slate-100 transition-colors cursor-pointer"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7"/>
        </svg>
      </button>

      <div class="min-w-0">
        <h2 class="text-xl font-black text-slate-900 leading-tight">
          Schüler bearbeiten
        </h2>
        <p class="text-xs text-slate-500 font-medium mt-0.5">
          {student?.vorname} {student?.nachname} · {student?.barcode_id}
        </p>
      </div>
    </div>
    
    <div class="flex items-center gap-3 shrink-0">
      <button
        onclick={save}
        disabled={saving}
        class="px-6 py-2.5 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700
               rounded-xl transition-all shadow-sm hover:shadow-md cursor-pointer disabled:opacity-50
               flex items-center gap-2.5"
      >
        {#if saving}
          <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
          Speichert…
        {:else}
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
          </svg>
          Speichern
        {/if}
      </button>
    </div>
  </header>

  <!-- ── Scrollable Body ────────────────────────────────────────────────── -->
  <div class="flex-1 overflow-y-auto px-8 py-6 space-y-8">

    <!-- ── Persönliche Daten ──────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2">
        <div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
        Persönliche Daten
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div>
          <label for="vorname" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Vorname</label>
          <input
            id="vorname"
            type="text"
            bind:value={formData.vorname}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="nachname" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Nachname</label>
          <input
            id="nachname"
            type="text"
            bind:value={formData.nachname}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="geburtsdatum" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Geburtsdatum</label>
          <input
            id="geburtsdatum"
            type="date"
            bind:value={formData.geburtsdatum}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="lusd_id" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">LUSD-ID</label>
          <input
            id="lusd_id"
            type="text"
            bind:value={formData.lusd_id}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-mono text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
      </div>
    </section>

    <!-- ── Schuldaten ─────────────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2">
        <div class="w-2.5 h-2.5 rounded-full bg-slate-300"></div>
        Schuldaten
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div>
          <label for="klasse" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Klasse</label>
          <input
            id="klasse"
            type="text"
            bind:value={formData.klasse}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="barcode" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Schüler-ID / Barcode</label>
          <input
            id="barcode"
            type="text"
            bind:value={formData.barcode_id}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-mono text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="abgangsjahr" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Abgangsjahr</label>
          <input
            id="abgangsjahr"
            type="number"
            bind:value={formData.abgaenger_jahr}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="status" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Status</label>
          <select
            id="status"
            bind:value={formData.status}
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm font-semibold text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          >
            <option value="aktiv">Aktiv</option>
            <option value="inaktiv">Inaktiv</option>
            <option value="abgaenger">Abgänger</option>
          </select>
        </div>
      </div>
    </section>

    <!-- ── Kontaktdaten ────────────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-4 flex items-center gap-2">
        <div class="w-2.5 h-2.5 rounded-full bg-blue-400"></div>
        Kontaktdaten
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="lg:col-span-2 grid grid-cols-4 gap-4">
          <div class="col-span-3">
            <label for="strasse" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Straße</label>
            <input
              id="strasse"
              type="text"
              bind:value={formData.strasse}
              placeholder="Musterstraße"
              class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                     focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
            />
          </div>
          <div class="col-span-1">
            <label for="hausnummer" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Nr.</label>
            <input
              id="hausnummer"
              type="text"
              bind:value={formData.hausnummer}
              placeholder="12a"
              class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                     focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
            />
          </div>
        </div>
        
        <div>
          <label for="plz" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">PLZ</label>
          <input
            id="plz"
            type="text"
            bind:value={formData.plz}
            placeholder="12345"
            maxlength="5"
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800 font-mono
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        
        <div>
          <label for="ort" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Ort</label>
          <input
            id="ort"
            type="text"
            bind:value={formData.ort}
            placeholder="Musterstadt"
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>

        <div class="lg:col-span-2">
          <label for="email" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Eltern E-Mail</label>
          <input
            id="email"
            type="email"
            bind:value={formData.eltern_email}
            placeholder="eltern@schule.de"
            class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                   focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
      </div>
    </section>

    <!-- Bottom spacing -->
    <div class="h-4"></div>
  </div>
</div>
