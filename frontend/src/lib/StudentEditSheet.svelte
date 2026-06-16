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
    strasse: '',
    hausnummer: '',
    plz: '',
    ort: '',
    eltern_email: '',
    geburtsdatum: '',
  });

  $effect(() => {
    if (student) {
      formData.strasse      = student.strasse       || '';
      formData.hausnummer   = student.hausnummer    || '';
      formData.plz          = student.plz           || '';
      formData.ort          = student.ort           || '';
      formData.eltern_email = student.eltern_email  || '';
      formData.geburtsdatum = student.geburtsdatum
        ? student.geburtsdatum.slice(0, 10)
        : '';
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
        strasse:      formData.strasse      || null,
        hausnummer:   formData.hausnummer   || null,
        plz:          formData.plz          || null,
        ort:          formData.ort          || null,
        eltern_email: formData.eltern_email || null,
        geburtsdatum: formData.geburtsdatum || null,
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

  /** @param {KeyboardEvent} e */
  function handleKey(e) {
    if (e.key === 'Escape') onClose();
  }
</script>

<svelte:window onkeydown={handleKey} />

<!-- Snackbar (floating, self-dismissing, layout-neutral) -->
{#if snackbar}
  <div
    class="fixed bottom-8 left-1/2 -translate-x-1/2 z-[200]
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

<!-- Scrim -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-40 bg-slate-900/40 backdrop-blur-[2px] animate-fade-in"
  onclick={onClose}
  aria-hidden="true"
></div>

<!-- Side Sheet -->
<div
  class="fixed top-0 right-0 z-50 h-full w-11/12 max-w-5xl
         flex flex-col bg-white shadow-2xl animate-slide-in-right"
  role="dialog"
  aria-modal="true"
  aria-label="Schüler bearbeiten"
>

  <!-- ── Header ─────────────────────────────────────────────────────────── -->
  <header class="shrink-0 flex items-center justify-between gap-4 px-10 py-6 border-b border-slate-100">
    <div class="flex items-center gap-4 min-w-0">
      <!-- Avatar -->
      <div class="w-11 h-11 rounded-2xl bg-linear-to-br from-blue-500 to-indigo-600
                  flex items-center justify-center text-white font-black text-sm shrink-0 shadow-md">
        {(student?.vorname?.[0] ?? '') + (student?.nachname?.[0] ?? '')}
      </div>
      <div class="min-w-0">
        <h2 class="text-lg font-black text-slate-900 leading-tight">
          {student?.vorname} {student?.nachname}
        </h2>
        <p class="text-xs text-slate-400 font-semibold mt-0.5 flex items-center gap-1.5">
          <span class="font-mono">{student?.barcode_id}</span>
          <span class="text-slate-200">·</span>
          Klasse {student?.klasse}
        </p>
      </div>
    </div>
    <button
      onclick={onClose}
      aria-label="Schließen"
      class="w-9 h-9 shrink-0 flex items-center justify-center rounded-xl
             text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
    >
      <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
      </svg>
    </button>
  </header>

  <!-- ── Scrollable Body ────────────────────────────────────────────────── -->
  <div class="flex-1 overflow-y-auto px-10 py-8 space-y-10">

    <!-- ── Persönliche Daten (read-only) ──────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-400 uppercase tracking-[0.12em] mb-5 flex items-center gap-2">
        <div class="w-3.5 h-0.5 rounded-full bg-slate-300"></div>
        Persönliche Daten
        <span class="text-slate-300 normal-case font-medium tracking-normal text-[9px]">(aus LUSD – schreibgeschützt)</span>
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-6">
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Vorname</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-semibold text-slate-600 select-text">
            {student?.vorname || '—'}
          </div>
        </div>
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Nachname</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-semibold text-slate-600 select-text">
            {student?.nachname || '—'}
          </div>
        </div>
        <div>
          <label for="sheet-geburtsdatum" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
            Geburtsdatum
          </label>
          <input
            id="sheet-geburtsdatum"
            type="date"
            bind:value={formData.geburtsdatum}
            class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                   focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">LUSD-ID</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-mono text-slate-500 select-text">
            {student?.lusd_id || '—'}
          </div>
        </div>
      </div>
    </section>

    <!-- Divider -->
    <div class="border-t border-slate-100"></div>

    <!-- ── Schuldaten (read-only) ─────────────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-400 uppercase tracking-[0.12em] mb-5 flex items-center gap-2">
        <div class="w-3.5 h-0.5 rounded-full bg-slate-300"></div>
        Schuldaten
        <span class="text-slate-300 normal-case font-medium tracking-normal text-[9px]">(aus LUSD – schreibgeschützt)</span>
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-6">
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Klasse</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-semibold text-slate-600 select-text">
            {student?.klasse || '—'}
          </div>
        </div>
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Schüler-ID / Barcode</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-mono text-slate-500 select-text">
            {student?.barcode_id || '—'}
          </div>
        </div>
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Abgangsjahr</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl text-sm font-semibold text-slate-600 select-text">
            {student?.abgaenger_jahr || '—'}
          </div>
        </div>
        <div>
          <label class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Status</label>
          <div class="px-4 py-3 bg-slate-50 border border-slate-100 rounded-xl flex items-center gap-2">
            <span class="w-2 h-2 rounded-full shrink-0 {student?.status === 'aktiv' ? 'bg-emerald-500' : 'bg-slate-400'}"></span>
            <span class="text-sm font-semibold text-slate-600">
              {student?.status === 'aktiv' ? 'Aktiv' : student?.status || 'Unbekannt'}
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- Divider -->
    <div class="border-t border-slate-100"></div>

    <!-- ── Kontaktdaten (editable) ────────────────────────────────────── -->
    <section>
      <h3 class="text-[10px] font-black text-slate-500 uppercase tracking-[0.12em] mb-5 flex items-center gap-2">
        <div class="w-3.5 h-0.5 rounded-full bg-blue-400"></div>
        Kontaktdaten
        <span class="text-slate-300 normal-case font-medium tracking-normal text-[9px]">(bearbeitbar)</span>
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-6">

        <!-- Straße + Hausnummer -->
        <div class="md:col-span-2 grid grid-cols-3 gap-4">
          <div class="col-span-2">
            <label for="sheet-strasse" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
              Straße
            </label>
            <input
              id="sheet-strasse"
              type="text"
              bind:value={formData.strasse}
              placeholder="Musterstraße"
              class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                     placeholder:text-slate-300
                     focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
            />
          </div>
          <div>
            <label for="sheet-hausnummer" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
              Hausnr.
            </label>
            <input
              id="sheet-hausnummer"
              type="text"
              bind:value={formData.hausnummer}
              placeholder="12a"
              class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                     placeholder:text-slate-300
                     focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
            />
          </div>
        </div>

        <!-- PLZ + Ort -->
        <div>
          <label for="sheet-plz" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
            PLZ
          </label>
          <input
            id="sheet-plz"
            type="text"
            bind:value={formData.plz}
            placeholder="12345"
            maxlength="5"
            class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                   font-mono placeholder:text-slate-300
                   focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
        <div>
          <label for="sheet-ort" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
            Ort
          </label>
          <input
            id="sheet-ort"
            type="text"
            bind:value={formData.ort}
            placeholder="Musterstadt"
            class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                   placeholder:text-slate-300
                   focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>

        <!-- E-Mail -->
        <div class="md:col-span-2">
          <label for="sheet-email" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">
            Eltern E-Mail
          </label>
          <input
            id="sheet-email"
            type="email"
            bind:value={formData.eltern_email}
            placeholder="eltern@schule.de"
            class="w-full px-4 py-3 bg-white border border-slate-200 rounded-xl text-sm text-slate-800
                   placeholder:text-slate-300
                   focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>

      </div>
    </section>

    <!-- Bottom spacing so footer shadow doesn't cut off content -->
    <div class="h-2"></div>
  </div>

  <!-- ── Footer ─────────────────────────────────────────────────────────── -->
  <footer class="shrink-0 flex items-center justify-between gap-4 px-10 py-5 border-t border-slate-100 bg-white">
    <p class="text-xs text-slate-400 font-medium leading-relaxed max-w-sm">
      Nur <span class="text-slate-600 font-semibold">Kontaktdaten</span> und <span class="text-slate-600 font-semibold">Geburtsdatum</span>
      können hier gespeichert werden. Alle anderen Felder werden über LUSD synchronisiert.
    </p>
    <div class="flex items-center gap-3 shrink-0">
      <button
        onclick={onClose}
        disabled={saving}
        class="px-5 py-2.5 text-sm font-semibold text-slate-600 bg-slate-100 hover:bg-slate-200
               rounded-xl transition-colors cursor-pointer disabled:opacity-50"
      >
        Abbrechen
      </button>
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
  </footer>
</div>

<style>
  @keyframes slide-in-right {
    from { transform: translateX(100%); opacity: 0.5; }
    to   { transform: translateX(0);    opacity: 1; }
  }
  .animate-slide-in-right {
    animation: slide-in-right 0.28s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
</style>
