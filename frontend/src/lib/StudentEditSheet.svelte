<script>
  import { apiClient, apiFetch } from './apiFetch.js';

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
  let errorMsg = $state('');

  let formData = $state({
    strasse: '',
    hausnummer: '',
    plz: '',
    ort: '',
    eltern_email: '',
    geburtsdatum: '',
    lusd_id: '',
  });

  /** @type {any[]} */
  let ausleihen = $state([]);
  /** @type {{ offene_mahngebuehren?: number, anzahl_beschaedigungen?: number } | null} */
  let konto = $state(null);
  let ausleihenLoading = $state(true);

  $effect(() => {
    if (student) {
      formData.strasse       = student.strasse        || '';
      formData.hausnummer    = student.hausnummer      || '';
      formData.plz           = student.plz             || '';
      formData.ort           = student.ort             || '';
      formData.eltern_email  = student.eltern_email    || '';
      formData.geburtsdatum  = student.geburtsdatum    ? student.geburtsdatum.slice(0, 10) : '';
      formData.lusd_id       = student.lusd_id         || '';

      ausleihen = student.entliehene_buecher || [];
      konto = {
        offene_mahngebuehren: student.offene_mahngebuehren ?? 0,
        anzahl_beschaedigungen: student.anzahl_beschaedigungen ?? 0,
      };
      ausleihenLoading = false;
    }
  });

  function formatCurrency(/** @type {number} */ n) {
    return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(n ?? 0);
  }

  function formatDate(/** @type {string} */ s) {
    if (!s) return '—';
    try { return new Date(s).toLocaleDateString('de-DE'); } catch { return s; }
  }

  async function save() {
    saving = true;
    errorMsg = '';
    try {
      const payload = {
        strasse:       formData.strasse       || null,
        hausnummer:    formData.hausnummer     || null,
        plz:           formData.plz            || null,
        ort:           formData.ort            || null,
        eltern_email:  formData.eltern_email   || null,
        geburtsdatum:  formData.geburtsdatum   || null,
        lusd_id:       formData.lusd_id        || null,
      };
      const res = await apiClient.patch(`/api/schueler/${student.id}`, payload);
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Speichern fehlgeschlagen');
      }
      onSave();
    } catch (e) {
      errorMsg = String(e);
    } finally {
      saving = false;
    }
  }

  /**
   * Close on Escape key
   * @param {KeyboardEvent} e
   */
  function handleKey(e) {
    if (e.key === 'Escape') onClose();
  }
</script>

<svelte:window onkeydown={handleKey} />

<!-- Scrim / Overlay -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-40 bg-slate-900/40 backdrop-blur-[2px] animate-fade-in"
  onclick={onClose}
  aria-hidden="true"
></div>

<!-- Side Sheet -->
<aside
  class="fixed top-0 right-0 z-50 h-full w-11/12 max-w-7xl
         flex flex-col
         bg-white shadow-2xl
         animate-slide-in-right"
  role="dialog"
  aria-modal="true"
  aria-label="Schüler bearbeiten"
>
  <!-- ── Header ─────────────────────────────────────────────────────────── -->
  <header class="shrink-0 flex items-center justify-between gap-4 px-8 py-5 border-b border-slate-100 bg-white">
    <div class="flex items-center gap-4 min-w-0">
      <!-- Avatar / Initials -->
      <div class="w-12 h-12 rounded-2xl bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center text-white font-black text-base shrink-0 shadow-md">
        {(student?.vorname?.[0] ?? '') + (student?.nachname?.[0] ?? '')}
      </div>
      <div class="min-w-0">
        <h2 class="text-xl font-black text-slate-900 leading-tight truncate">
          {student?.vorname} {student?.nachname}
        </h2>
        <p class="text-xs font-semibold text-slate-500 mt-0.5 flex items-center gap-2">
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 font-bold text-[10px] uppercase tracking-wider">
            Klasse {student?.klasse}
          </span>
          <span class="text-slate-300">·</span>
          <span class="font-mono text-slate-400 text-[11px]">{student?.id?.slice(0, 8)}…</span>
        </p>
      </div>
    </div>
    <button
      onclick={onClose}
      aria-label="Schließen"
      class="w-10 h-10 shrink-0 flex items-center justify-center rounded-2xl text-slate-400
             hover:text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
    >
      <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
      </svg>
    </button>
  </header>

  <!-- ── Error Banner ───────────────────────────────────────────────────── -->
  {#if errorMsg}
    <div class="shrink-0 mx-8 mt-4 px-4 py-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-xl text-sm font-semibold flex items-center gap-2">
      <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      {errorMsg}
    </div>
  {/if}

  <!-- ── Body: 3-column grid ────────────────────────────────────────────── -->
  <div class="flex-1 overflow-hidden px-8 py-6">
    <div class="h-full grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">

      <!-- ── COLUMN 1: Stammdaten-Formular ──────────────────────────────── -->
      <section class="flex flex-col gap-4 overflow-y-auto pr-1">
        <!-- Section label -->
        <div class="flex items-center gap-2 mb-1">
          <div class="w-1 h-5 rounded-full bg-blue-500"></div>
          <h3 class="text-xs font-black text-slate-500 uppercase tracking-widest">Stammdaten</h3>
        </div>

        <!-- Name (read-only from LUSD) -->
        <div class="p-4 rounded-2xl bg-slate-50 border border-slate-100 space-y-3">
          <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-2">Aus LUSD (nur lesen)</p>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Vorname</p>
              <p class="text-sm font-semibold text-slate-800">{student?.vorname || '—'}</p>
            </div>
            <div>
              <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Nachname</p>
              <p class="text-sm font-semibold text-slate-800">{student?.nachname || '—'}</p>
            </div>
            <div>
              <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Klasse</p>
              <p class="text-sm font-semibold text-slate-800">{student?.klasse || '—'}</p>
            </div>
            <div>
              <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Barcode-ID</p>
              <p class="text-sm font-mono text-slate-700">{student?.barcode_id || '—'}</p>
            </div>
          </div>
        </div>

        <!-- Editable fields -->
        <div class="space-y-3">
          <!-- Geburtsdatum -->
          <div>
            <label for="sheet-geburtsdatum" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Geburtsdatum</label>
            <input
              id="sheet-geburtsdatum"
              type="date"
              bind:value={formData.geburtsdatum}
              class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                     focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
            />
          </div>

          <!-- LUSD-ID -->
          <div>
            <label for="sheet-lusd" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">LUSD-ID</label>
            <input
              id="sheet-lusd"
              type="text"
              bind:value={formData.lusd_id}
              placeholder="z. B. 12345"
              class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                     focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all font-mono"
            />
          </div>
        </div>

        <!-- Postanschrift -->
        <div>
          <p class="text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-2">Postanschrift</p>
          <div class="space-y-2.5">
            <div class="grid grid-cols-3 gap-2">
              <div class="col-span-2">
                <label for="sheet-strasse" class="block text-[10px] font-semibold text-slate-400 mb-1">Straße</label>
                <input
                  id="sheet-strasse"
                  type="text"
                  bind:value={formData.strasse}
                  placeholder="Musterstraße"
                  class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                         focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
              <div>
                <label for="sheet-hnr" class="block text-[10px] font-semibold text-slate-400 mb-1">Nr.</label>
                <input
                  id="sheet-hnr"
                  type="text"
                  bind:value={formData.hausnummer}
                  placeholder="12a"
                  class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                         focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
            </div>
            <div class="grid grid-cols-3 gap-2">
              <div>
                <label for="sheet-plz" class="block text-[10px] font-semibold text-slate-400 mb-1">PLZ</label>
                <input
                  id="sheet-plz"
                  type="text"
                  bind:value={formData.plz}
                  placeholder="12345"
                  maxlength="5"
                  class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                         focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all font-mono"
                />
              </div>
              <div class="col-span-2">
                <label for="sheet-ort" class="block text-[10px] font-semibold text-slate-400 mb-1">Ort</label>
                <input
                  id="sheet-ort"
                  type="text"
                  bind:value={formData.ort}
                  placeholder="Musterstadt"
                  class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                         focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Eltern E-Mail -->
        <div>
          <label for="sheet-email" class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1">Eltern E-Mail</label>
          <input
            id="sheet-email"
            type="email"
            bind:value={formData.eltern_email}
            placeholder="eltern@schule.de"
            class="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
                   focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all"
          />
        </div>
      </section>

      <!-- ── COLUMN 2: Aktuelle Ausleihen ───────────────────────────────── -->
      <section class="flex flex-col gap-3 overflow-hidden">
        <div class="flex items-center justify-between mb-1">
          <div class="flex items-center gap-2">
            <div class="w-1 h-5 rounded-full bg-amber-400"></div>
            <h3 class="text-xs font-black text-slate-500 uppercase tracking-widest">Aktuelle Ausleihen</h3>
          </div>
          {#if ausleihen.length > 0}
            <span class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-amber-100 text-amber-700 text-[10px] font-black">
              {ausleihen.length}
            </span>
          {/if}
        </div>

        <div class="flex-1 overflow-y-auto rounded-2xl border border-slate-100 bg-slate-50">
          {#if ausleihenLoading}
            <div class="h-full flex items-center justify-center py-10">
              <div class="w-6 h-6 border-2 border-amber-400 border-t-transparent rounded-full animate-spin"></div>
            </div>
          {:else if ausleihen.length === 0}
            <div class="h-full flex flex-col items-center justify-center gap-3 py-12 text-slate-400">
              <svg class="w-10 h-10 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
              </svg>
              <p class="text-sm font-semibold">Keine aktiven Ausleihen</p>
            </div>
          {:else}
            <ul class="divide-y divide-slate-100">
              {#each ausleihen as buch (buch.exemplar_id ?? buch.barcode_id)}
                {@const isOverdue = buch.rueckgabe_datum && new Date(buch.rueckgabe_datum) < new Date()}
                <li class="flex items-start gap-3 px-4 py-3 hover:bg-white/80 transition-colors">
                  <div class="w-8 h-8 rounded-xl flex items-center justify-center shrink-0 mt-0.5
                               {isOverdue ? 'bg-rose-100' : 'bg-amber-50'}">
                    <svg class="w-4 h-4 {isOverdue ? 'text-rose-500' : 'text-amber-500'}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
                    </svg>
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-semibold text-slate-800 truncate leading-tight">{buch.titel || '—'}</p>
                    <p class="text-[10px] text-slate-500 mt-0.5 truncate">{buch.autor || ''}</p>
                    <div class="flex items-center gap-2 mt-1">
                      <span class="font-mono text-[10px] text-slate-400">{buch.barcode_id}</span>
                      {#if buch.rueckgabe_datum}
                        <span class="text-[10px] font-semibold {isOverdue ? 'text-rose-600' : 'text-slate-500'}">
                          · bis {formatDate(buch.rueckgabe_datum)}
                          {#if isOverdue}<span class="ml-1 px-1 py-0.5 rounded-md bg-rose-100 text-rose-600 text-[9px] font-black">ÜBERFÄLLIG</span>{/if}
                        </span>
                      {/if}
                    </div>
                  </div>
                </li>
              {/each}
            </ul>
          {/if}
        </div>
      </section>

      <!-- ── COLUMN 3: Konto & Mahnungen ────────────────────────────────── -->
      <section class="flex flex-col gap-4">
        <div class="flex items-center gap-2 mb-1">
          <div class="w-1 h-5 rounded-full bg-rose-400"></div>
          <h3 class="text-xs font-black text-slate-500 uppercase tracking-widest">Konto & Mahnungen</h3>
        </div>

        <!-- Mahngebühren -->
        <div class="p-4 rounded-2xl border {(konto?.offene_mahngebuehren ?? 0) > 0 ? 'bg-rose-50 border-rose-200' : 'bg-slate-50 border-slate-100'}">
          <p class="text-[10px] font-bold uppercase tracking-wider mb-1 {(konto?.offene_mahngebuehren ?? 0) > 0 ? 'text-rose-400' : 'text-slate-400'}">Offene Mahngebühren</p>
          <p class="text-2xl font-black {(konto?.offene_mahngebuehren ?? 0) > 0 ? 'text-rose-600' : 'text-emerald-600'}">
            {formatCurrency(konto?.offene_mahngebuehren ?? 0)}
          </p>
          {#if (konto?.offene_mahngebuehren ?? 0) === 0}
            <p class="text-xs text-emerald-600 font-semibold mt-1 flex items-center gap-1">
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
              Konto ausgeglichen
            </p>
          {/if}
        </div>

        <!-- Schadensfälle -->
        <div class="p-4 rounded-2xl border {(konto?.anzahl_beschaedigungen ?? 0) > 0 ? 'bg-orange-50 border-orange-200' : 'bg-slate-50 border-slate-100'}">
          <p class="text-[10px] font-bold uppercase tracking-wider mb-1 {(konto?.anzahl_beschaedigungen ?? 0) > 0 ? 'text-orange-400' : 'text-slate-400'}">Schadensfälle</p>
          <p class="text-2xl font-black {(konto?.anzahl_beschaedigungen ?? 0) > 0 ? 'text-orange-600' : 'text-slate-700'}">
            {konto?.anzahl_beschaedigungen ?? 0}
          </p>
          <p class="text-xs text-slate-500 font-semibold mt-1">gemeldete Beschädigungen</p>
        </div>

        <!-- Statistiken -->
        <div class="p-4 rounded-2xl bg-slate-50 border border-slate-100 space-y-3">
          <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">Statistik</p>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <p class="text-[10px] text-slate-400 font-semibold">Aktive Ausleihen</p>
              <p class="text-xl font-black text-slate-800">{ausleihen.length}</p>
            </div>
            <div>
              <p class="text-[10px] text-slate-400 font-semibold">Überfällig</p>
              <p class="text-xl font-black {ausleihen.filter(b => b.rueckgabe_datum && new Date(b.rueckgabe_datum) < new Date()).length > 0 ? 'text-rose-600' : 'text-slate-800'}">
                {ausleihen.filter(b => b.rueckgabe_datum && new Date(b.rueckgabe_datum) < new Date()).length}
              </p>
            </div>
          </div>
        </div>

        <!-- System Info -->
        <div class="p-4 rounded-2xl bg-slate-50 border border-slate-100 space-y-2">
          <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">System-Info</p>
          <div class="space-y-1.5">
            <div class="flex justify-between items-center">
              <span class="text-[10px] text-slate-400">Status</span>
              <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold
                           {student?.status === 'aktiv' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-200 text-slate-600'}">
                <span class="w-1.5 h-1.5 rounded-full {student?.status === 'aktiv' ? 'bg-emerald-500' : 'bg-slate-400'}"></span>
                {student?.status === 'aktiv' ? 'Aktiv' : student?.status || 'Unbekannt'}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-[10px] text-slate-400">Abgangsjahr</span>
              <span class="text-[10px] font-semibold text-slate-600">{student?.abgaenger_jahr || '—'}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-[10px] text-slate-400">Schüler-ID</span>
              <span class="text-[10px] font-mono text-slate-500">{student?.barcode_id || '—'}</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>

  <!-- ── Footer / Save Button ───────────────────────────────────────────── -->
  <footer class="shrink-0 flex items-center justify-between gap-4 px-8 py-4 border-t border-slate-100 bg-white">
    <p class="text-xs text-slate-400 font-medium">
      Änderungen an Adresse, E-Mail, Geburtsdatum und LUSD-ID können gespeichert werden.
      Stammdaten (Name, Klasse) werden über LUSD synchronisiert.
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
               rounded-xl transition-all cursor-pointer disabled:opacity-50 shadow-sm hover:shadow-md
               flex items-center gap-2.5"
      >
        {#if saving}
          <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
        {:else}
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
          </svg>
        {/if}
        Änderungen speichern
      </button>
    </div>
  </footer>
</aside>

<style>
  @keyframes slide-in-right {
    from { transform: translateX(100%); opacity: 0.4; }
    to   { transform: translateX(0);    opacity: 1; }
  }
  .animate-slide-in-right {
    animation: slide-in-right 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
</style>
