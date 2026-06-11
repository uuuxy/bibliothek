<script>
  import { apiFetch } from "./apiFetch.js";
  import WebcamCapture from "./WebcamCapture.svelte";
  import DamageReportModal from "./DamageReportModal.svelte";
  import BorrowedBooksCard from "./BorrowedBooksCard.svelte";
  import StudentEditModal from "./StudentEditModal.svelte";
  import StudentProfileCard from "./StudentProfileCard.svelte";
  import StudentPrintCard from "./StudentPrintCard.svelte";

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
  let loading = $state(true);
  let showWebcam = $state(false);
  let timestamp = $state(Date.now());

  let showDeleteConfirm = $state(false);
  let deleteError = $state("");
  let isDeleting = $state(false);

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
      const res = await apiFetch(`/api/schueler/${student.id}`);
      if (res.ok) {
        profile = await res.json();
      }
    } catch (err) {
      console.error("Fehler beim Laden des Schüler-Profils:", err);
    } finally {
      loading = false;
    }
  }

  async function deleteStudent() {
    if (profile.entliehene_buecher && profile.entliehene_buecher.length > 0) {
      deleteError = "Löschen nicht möglich: Schüler hat noch entliehene Bücher";
      return;
    }
    deleteError = "";
    isDeleting = true;
    try {
      const res = await apiFetch(`/api/schueler/${profile.id}`, { method: "DELETE" });
      if (res.ok) {
        showDeleteConfirm = false;
        onDeselect();
      } else {
        const errText = await res.text();
        try {
          const errObj = JSON.parse(errText);
          deleteError = errObj.error || "Fehler beim Löschen des Schülers.";
        } catch {
          deleteError = errText || "Fehler beim Löschen des Schülers.";
        }
      }
    } catch (err) {
      deleteError = "Netzwerkfehler beim Löschen des Schülers.";
      console.error(err);
    } finally {
      isDeleting = false;
    }
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
      const res = await apiFetch(`/api/damage/report`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          loan_id: /** @type {any} */ (damageBook).ausleihe_id,
          schueler_id: student.id,
          copy_id: /** @type {any} */ (damageBook).exemplar_id,
          beschreibung: reason,
          betrag: amount
        })
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

  <div class="w-full grid grid-cols-1 lg:grid-cols-3 gap-6 items-start text-slate-800 animate-fade-in no-print font-sans">
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
      <!-- Tabs -->
      <div class="flex gap-4 border-b border-slate-200 px-2 pt-2">
        <button onclick={() => activeTab = "ausleihen"} class="pb-3 px-2 text-sm font-bold transition-all border-b-2 {activeTab === 'ausleihen' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-500 hover:text-slate-800'}">
          Ausleihen & Historie
        </button>
        <button onclick={() => activeTab = "stammdaten"} class="pb-3 px-2 text-sm font-bold transition-all border-b-2 {activeTab === 'stammdaten' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-500 hover:text-slate-800'}">
          Stammdaten & Adresse
        </button>
      </div>

      <div class="flex-1 relative">
        {#if activeTab === "ausleihen"}
          {@render rightTop?.()}
          
          <div class="flex justify-end mb-2">
            {#if (role === 'admin' || role === 'mitarbeiter') && profile.entliehene_buecher?.length > 0}
              <button onclick={downloadKontoauszugPDF} disabled={kontoauszugPdfLoading} class="px-4 py-2 bg-blue-50 text-blue-600 hover:bg-blue-100 disabled:opacity-50 rounded-xl text-sm font-bold transition-colors cursor-pointer flex items-center gap-2">
                {#if kontoauszugPdfLoading}
                  <div class="w-4 h-4 border-2 border-blue-400 border-t-blue-700 rounded-full animate-spin"></div>
                {:else}
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
                {/if}
                Kontoauszug drucken
              </button>
            {/if}
          </div>

          <div class="col-span-1 md:col-span-1 relative flex flex-col h-full min-h-[400px] animate-fade-in">
            <BorrowedBooksCard 
              books={profile.entliehene_buecher || []} 
              {onReturnClick} 
              onDamageClick={role === 'admin' || role === 'mitarbeiter' ? openDamageModal : undefined}
            />
          </div>
        {:else if activeTab === "stammdaten"}
          <div class="w-full pt-2 animate-fade-in space-y-8">
            <div class="flex justify-between items-center border-b border-slate-100 pb-4">
              <h3 class="text-xl font-bold text-slate-800 flex items-center gap-2">
                <svg class="w-6 h-6 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V8a2 2 0 00-2-2h-5m-4 0V5a2 2 0 114 0v1m-4 0a2 2 0 104 0m-5 8a2 2 0 100-4 2 2 0 000 4zm0 0c1.306 0 2.417.835 2.83 2M9 14a3.001 3.001 0 00-2.83 2M15 11h3m-3 4h2"/></svg>
                Stammdaten & Adresse
              </h3>
              <div class="flex items-center gap-2">
                {#if role === 'admin' || role === 'mitarbeiter'}
                  <button onclick={downloadRechnungPDF} disabled={rechnungPdfLoading} class="px-4 py-2 bg-slate-100 text-slate-700 hover:bg-slate-200 disabled:opacity-50 rounded-xl text-sm font-bold transition-colors cursor-pointer flex items-center gap-2">
                    {#if rechnungPdfLoading}
                      <div class="w-4 h-4 border-2 border-slate-400 border-t-slate-700 rounded-full animate-spin"></div>
                    {:else}
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
                    {/if}
                    Ersatzforderung drucken (PDF)
                  </button>
                {/if}
                {#if role === 'admin'}
                  <button onclick={() => showEditModal = true} class="px-4 py-2 bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-xl text-sm font-bold transition-colors cursor-pointer flex items-center gap-2">
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
                    Bearbeiten
                  </button>
                {/if}
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
              <div class="space-y-6">
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Geburtsdatum</p>
                  <p class="text-slate-800 font-semibold">{formatDate(profile.geburtsdatum)}</p>
                </div>
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">LUSD ID</p>
                  <p class="text-slate-800 font-semibold">{profile.lusd_id || 'Keine Angabe'}</p>
                </div>
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">System-ID</p>
                  <p class="text-slate-500 font-mono text-xs">{profile.id}</p>
                </div>
              </div>

              <div class="space-y-6">
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Postanschrift</p>
                  {#if profile.strasse}
                    <p class="text-slate-800 font-semibold">{profile.strasse} {profile.hausnummer}</p>
                    <p class="text-slate-800 font-semibold">{profile.plz} {profile.ort}</p>
                  {:else}
                    <p class="text-slate-400 italic text-sm">Keine Adresse hinterlegt</p>
                  {/if}
                </div>
                <div>
                  <p class="text-xs font-bold text-slate-400 uppercase tracking-wider mb-1">Eltern E-Mail</p>
                  {#if profile.eltern_email}
                    <a href="mailto:{profile.eltern_email}" class="text-blue-600 hover:underline font-semibold">{profile.eltern_email}</a>
                  {:else}
                    <p class="text-slate-400 italic text-sm">Keine E-Mail hinterlegt</p>
                  {/if}
                </div>
              </div>
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

{#if showWebcam}
  <WebcamCapture studentId={profile.id} onCapture={handlePhotoCaptured} onClose={() => showWebcam = false} />
{/if}

{#if profile}
  <StudentPrintCard {profile} {timestamp} />
{/if}

{#if showDeleteConfirm}
  <div class="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 backdrop-blur-xs p-4 animate-fade-in" role="dialog" aria-modal="true">
    <div class="w-full max-w-md rounded-3xl border border-slate-200 bg-white p-6 shadow-2xl text-slate-800 text-left">
      <h3 class="text-lg font-bold text-rose-600 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
        <span>Schüler löschen</span>
      </h3>
      {#if profile.entliehene_buecher && profile.entliehene_buecher.length > 0}
        <div class="mt-4 p-4 bg-rose-50 border border-rose-100 rounded-2xl text-sm font-semibold text-rose-700">
          Löschen nicht möglich: Schüler hat noch entliehene Bücher
        </div>
        <div class="mt-6 flex justify-end">
          <button onclick={() => { showDeleteConfirm = false; deleteError = ""; }} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 transition-colors cursor-pointer">Schließen</button>
        </div>
      {:else}
        <p class="mt-4 text-sm text-slate-600 leading-relaxed font-sans">
          Sind Sie sicher, dass Sie den Schüler <strong>{profile.vorname} {profile.nachname}</strong> unwiderruflich aus der Datenbank löschen möchten? Alle historischen Ausleihen werden anonymisiert.
        </p>
        {#if deleteError}
          <div class="mt-4 p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600">
            {deleteError}
          </div>
        {/if}
        <div class="mt-6 flex justify-end gap-3">
          <button onclick={() => { showDeleteConfirm = false; deleteError = ""; }} disabled={isDeleting} class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer">Abbrechen</button>
          <button onclick={deleteStudent} disabled={isDeleting} class="rounded-xl bg-red-650 px-4 py-2 text-sm font-bold text-white hover:bg-red-750 disabled:opacity-60 transition-colors cursor-pointer">
            {#if isDeleting}Löschen...{:else}Unwiderruflich löschen{/if}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

{#if showDamageModal && damageBook}
  <DamageReportModal book={damageBook} isSubmitting={isSubmittingDamage} onCancel={() => showDamageModal = false} onSubmit={submitDamageReport} />
{/if}

{#if showEditModal}
  <StudentEditModal student={profile} onClose={() => showEditModal = false} onSave={handleSaveEdit} />
{/if}
