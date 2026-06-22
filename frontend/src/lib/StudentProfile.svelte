<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import WebcamCapture from "./WebcamCapture.svelte";
  import DamageReportModal from "./DamageReportModal.svelte";
  import BorrowedBooksCard from "./BorrowedBooksCard.svelte";
  import StudentEditSheet from './StudentEditSheet.svelte';
  import StudentProfileCard from "./StudentProfileCard.svelte";
  import StudentPrintCard from "./StudentPrintCard.svelte";
  import StudentProfileDeleteModal from "./StudentProfileDeleteModal.svelte";
  import StudentProfileStammdaten from "./StudentProfileStammdaten.svelte";
  import StudentVormerkungenCard from "./StudentVormerkungenCard.svelte";

  /** 
   * @typedef {Object} Props
   * @property {any} student - The selected student object
   * @property {() => void} onDeselect - Callback when profile is closed
   * @property {string} [role] - Active user role (admin, mitarbeiter, etc)
   * @property {(barcode: string) => void} [onReturnClick] - Callback for returning a book
   * @property {import('svelte').Snippet} [leftActions] - Optional slot for left card actions
   * @property {import('svelte').Snippet} [rightTop] - Optional slot for right content top
   */

  /** @type {Props} */
  let { student, onDeselect, role = "", onReturnClick = undefined, leftActions, rightTop } = $props();

  // ── State variables ───────────────────────────────────────────────────────
  /** @type {any} */
  let profile = $state(null);
  /** @type {any[]} */
  let vormerkungen = $state([]);
  let loading = $state(true);
  let showWebcam = $state(false);
  let timestamp = $state(Date.now());

  let showDeleteConfirm = $state(false);

  // Active Tab for Right Side ('ausleihen' | 'stammdaten')
  let activeTab = $state("ausleihen");
  let showEditModal = $state(false);

  // Damage Report State
  let showDamageModal = $state(false);
  /** @type {any} */
  let damageBook = $state(null);
  let isSubmittingDamage = $state(false);

  // ── Profile loading and deletion ──────────────────────────────────────────
  async function fetchProfile() {
    if (!student) return;
    loading = true;
    try {
      const [resProfile, resVormerkungen] = await Promise.all([
        apiFetch(`/api/schueler/${student.id}`),
        apiFetch(`/api/vormerkungen?schueler_id=${student.id}`)
      ]);
      if (resProfile.ok) profile = await resProfile.json();
      if (resVormerkungen.ok) vormerkungen = await resVormerkungen.json();
    } catch (err) {
      console.error("Fehler beim Laden des Schüler-Profils:", err);
    } finally {
      loading = false;
    }
  }

  function handleDeleteSuccess() {
    showDeleteConfirm = false;
    onDeselect();
  }

  // Reload profile when the student prop changes
  $effect(() => {
    if (student) {
      fetchProfile();
    }
  });

  export function reloadProfile() {
    fetchProfile();
  }

  function handleSaveEdit() {
    showEditModal = false;
    fetchProfile();
  }

  function formatDate(dateString) {
    if (!dateString) return "Keine Angabe";
    try {
      const d = new Date(dateString);
      return d.toLocaleDateString("de-DE", { day: '2-digit', month: '2-digit', year: 'numeric' });
    } catch {
      return dateString;
    }
  }

  function handlePhotoCaptured() {
    timestamp = Date.now();
    showWebcam = false;
    fetchProfile();
  }

  // ── Single-card print trigger ─────────────────────────────────────────────
  function printCard() {
    const styleEl = document.createElement("style");
    styleEl.textContent = "@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }";
    document.head.appendChild(styleEl);
    document.body.setAttribute("data-print-mode", "card-single");
    window.print();
    document.head.removeChild(styleEl);
    document.body.removeAttribute("data-print-mode");
  }

  let rechnungPdfLoading = $state(false);
  let globalErrorToast = $state(/** @type {string|null} */ (null));

  async function downloadRechnungPDF() {
    if (!profile) return;
    rechnungPdfLoading = true;
    globalErrorToast = null;
    try {
      const res = await apiFetch(`/api/print/rechnung/${profile.id}`);
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || "Keine ausstehenden Rechnungen gefunden");
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Rechnung_${profile.vorname}_${profile.nachname}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      globalErrorToast = String(e);
      setTimeout(() => globalErrorToast = null, 4000);
    } finally {
      rechnungPdfLoading = false;
    }
  }

  let kontoauszugPdfLoading = $state(false);

  async function downloadKontoauszugPDF() {
    if (!profile) return;
    kontoauszugPdfLoading = true;
    globalErrorToast = null;
    try {
      const res = await apiFetch(`/api/print/kontoauszug/${profile.id}`);
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || "Keine aktiven Ausleihen gefunden");
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Kontoauszug_${profile.vorname}_${profile.nachname}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      globalErrorToast = String(e);
      setTimeout(() => globalErrorToast = null, 4000);
    } finally {
      kontoauszugPdfLoading = false;
    }
  }

  function openDamageModal(book) {
    damageBook = book;
    showDamageModal = true;
  }

  async function submitDamageReport(reason, amount) {
    if (!damageBook) return;
    isSubmittingDamage = true;
    try {
      const res = await apiClient.post(`/api/damage/report`, {
          loan_id: /** @type {any} */ (damageBook).ausleihe_id,
          schueler_id: student.id,
          copy_id: /** @type {any} */ (damageBook).exemplar_id,
          beschreibung: reason,
          betrag: amount
        });
      if (res.ok) {
        const json = await res.json();
        window.open(`/api/schadensfaelle/${json.schadens_id}/pdf`, '_blank');
        showDamageModal = false;
        fetchProfile();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Melden.");
      }
    } catch (e) {
      alert("Netzwerkfehler.");
    } finally {
      isSubmittingDamage = false;
    }
  }
</script>

{#if loading}
  <div class="w-full py-12 flex justify-center items-center">
    <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if profile}
  {#if globalErrorToast}
    <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd" />
      </svg>
      {globalErrorToast}
    </div>
  {/if}

  {#if !showEditModal}
  <div class="w-full grid grid-cols-1 lg:grid-cols-3 gap-6 items-start text-slate-800 animate-fade-in no-print print:hidden font-sans">
    <!-- Left Column Profile Card -->
    <StudentProfileCard 
      bind:profile={profile}
      {role}
      {timestamp}
      bind:showWebcam={showWebcam}
      bind:showDeleteConfirm={showDeleteConfirm}
      {onDeselect}
      onPrint={printCard}
      {leftActions}
    />

    <!-- Right: Timeline / Loans List / Stammdaten (2 cols) -->
    <div class="lg:col-span-2 space-y-6 flex flex-col h-full">

      {#if role === 'admin' || role === 'mitarbeiter'}
      <!-- Aktionen / Dokumente -->
      <div class="bg-slate-50 border border-slate-200 rounded-2xl p-4 shadow-sm flex flex-col gap-3">
        <h4 class="text-xs font-bold text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
          Dokumente & Aktionen
        </h4>
        <div class="flex flex-wrap gap-3 items-center">
          <button onclick={downloadKontoauszugPDF} disabled={kontoauszugPdfLoading || !(profile.entliehene_buecher?.length > 0)} class="px-4 py-2 bg-white border border-slate-200 text-slate-700 hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
            {#if kontoauszugPdfLoading}
              <div class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"></div>
            {:else}
              <svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
            {/if}
            Kontoauszug
          </button>
          
          <button onclick={downloadRechnungPDF} disabled={rechnungPdfLoading || !profile.has_open_damages} title={!profile.has_open_damages ? 'Keine offenen Forderungen' : 'Ersatzforderung drucken'} class="px-4 py-2 bg-white border border-slate-200 text-slate-700 hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
            {#if rechnungPdfLoading}
              <div class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"></div>
            {:else}
              <svg class="w-4 h-4 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            {/if}
            Ersatzforderung
          </button>

          <button onclick={() => window.print()} disabled={!(profile.entliehene_buecher?.length > 0)} title={!(profile.entliehene_buecher?.length > 0) ? 'Keine offenen Ausleihen' : 'Druckansicht der Ausleihen'} class="px-4 py-2 bg-white border border-slate-200 text-slate-700 hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
            <svg class="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
            Ausleihen-Liste
          </button>
        </div>
      </div>
      {/if}

      <!-- Tabs -->
      <div class="flex gap-6 border-b border-slate-200">
        <button onclick={() => activeTab = "ausleihen"} class="pb-2 text-sm font-bold transition-all border-b-2 {activeTab === 'ausleihen' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-600 hover:text-slate-800'}">
          Ausleihen & Historie
        </button>
        <button onclick={() => activeTab = "stammdaten"} class="pb-2 text-sm font-bold transition-all border-b-2 {activeTab === 'stammdaten' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-600 hover:text-slate-800'}">
          Stammdaten & Adresse
        </button>
      </div>

      <div class="flex-1 relative">
        {#if activeTab === "ausleihen"}
          {@render rightTop?.()}
          <div class="col-span-1 md:col-span-1 relative flex flex-col gap-6 h-full min-h-[400px] animate-fade-in mt-4">
            <BorrowedBooksCard 
              books={profile.entliehene_buecher || []} 
              {onReturnClick} 
              onDamageClick={role === 'admin' || role === 'mitarbeiter' ? openDamageModal : undefined}
            />
            
            {#if vormerkungen.length > 0}
              <StudentVormerkungenCard bind:vormerkungen />
            {/if}
          </div>
        {:else if activeTab === "stammdaten"}
          <StudentProfileStammdaten 
            {profile} 
            {role} 
            {rechnungPdfLoading} 
            onDownloadRechnung={downloadRechnungPDF} 
            onEdit={() => showEditModal = true} 
          />
        {/if}
      </div>

      {#if role === 'admin'}
        <div class="mt-8 border-2 border-rose-100 bg-rose-50/50 rounded-2xl p-6 flex flex-col md:flex-row gap-6 items-center justify-between shadow-sm">
          <div>
            <h3 class="text-rose-700 font-bold text-lg flex items-center gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" /></svg>
              Gefahrenzone
            </h3>
            <p class="text-rose-600/80 text-sm mt-1 max-w-xl">
              Das Löschen dieses Schülerprofils entfernt die Person aus dem regulären System. Offene Ausleihen oder Forderungen müssen vorher beglichen werden.
            </p>
          </div>
          <button onclick={() => showDeleteConfirm = true} class="shrink-0 px-6 py-3 bg-white border border-rose-200 text-rose-600 hover:bg-rose-600 hover:text-white rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer">
            Schüler archivieren / löschen
          </button>
        </div>
      {/if}
    </div>
  </div>
  {:else}
    <StudentEditSheet student={profile} {role} onClose={() => showEditModal = false} onSave={handleSaveEdit} />
  {/if}
{/if}

{#if showWebcam}
  <WebcamCapture studentId={profile.id} onCapture={handlePhotoCaptured} onClose={() => showWebcam = false} />
{/if}

{#if profile}
  <StudentPrintCard {profile} {timestamp} />
{/if}

<StudentProfileDeleteModal 
  open={showDeleteConfirm} 
  {profile} 
  onclose={() => showDeleteConfirm = false} 
  onsuccess={handleDeleteSuccess} 
/>

{#if showDamageModal && damageBook}
  <DamageReportModal book={damageBook} isSubmitting={isSubmittingDamage} onCancel={() => showDamageModal = false} onSubmit={submitDamageReport} />
{/if}

{#if profile}
  <!-- Print Container für Ausleihen -->
  <div class="hidden print:block print:absolute print:top-0 print:left-0 print:w-full print:bg-white print:text-black print:p-8">
    <div class="text-center mb-8 border-b border-slate-300 pb-4">
      <h1 class="text-2xl font-bold">Ausleih-Quittung</h1>
      <h2 class="text-lg text-slate-600">Schulbibliothek</h2>
    </div>
    
    <div class="flex justify-between mb-8">
      <div>
        <p class="text-sm text-slate-500">Schüler/in</p>
        <p class="font-bold text-lg">{profile.vorname} {profile.nachname}</p>
        <p class="text-sm">{profile.klasse || ''}</p>
      </div>
      <div class="text-right">
        <p class="text-sm text-slate-500">Datum</p>
        <p class="font-bold">{new Date().toLocaleDateString('de-DE')}</p>
      </div>
    </div>

    <div class="mb-4">
      <h3 class="font-bold text-lg mb-2 border-b border-slate-300 pb-2">Offene Ausleihen</h3>
      {#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
        <table class="w-full text-left text-sm border-collapse">
          <thead>
            <tr class="border-b border-slate-300">
              <th class="py-2 px-2 font-semibold w-12">Cover</th>
              <th class="py-2 px-2 font-semibold">Titel</th>
              <th class="py-2 px-2 font-semibold text-center">Barcode/Signatur</th>
              <th class="py-2 px-2 font-semibold text-center">Ausgeliehen am</th>
              <th class="py-2 px-2 font-semibold text-right">Rückgabe bis</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200">
            {#each profile.entliehene_buecher as book}
              <tr>
                <td class="py-3 px-2">
                  {#if book.cover_url}
                    <img src={book.cover_url} alt="Cover" class="w-8 h-12 object-cover rounded shadow-sm" />
                  {:else}
                    <div class="w-8 h-12 bg-slate-100 rounded flex items-center justify-center text-xs text-slate-400">📖</div>
                  {/if}
                </td>
                <td class="py-3 px-2">
                  <div class="font-bold">{book.titel}</div>
                  <div class="text-xs text-slate-500">{book.autor}</div>
                </td>
                <td class="py-3 px-2 text-center font-mono text-xs">{book.barcode || book.signatur || '-'}</td>
                <td class="py-3 px-2 text-center">{formatDate(book.ausleih_datum)}</td>
                <td class="py-3 px-2 text-right font-bold {new Date(book.rueckgabe_datum) < new Date() ? 'text-red-600' : ''}">
                  {formatDate(book.rueckgabe_datum)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {:else}
        <p class="text-slate-500 italic">Keine offenen Ausleihen.</p>
      {/if}
    </div>
  </div>
{/if}
