<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  import WebcamCapture from "./WebcamCapture.svelte";
  import DamageReportModal from "./DamageReportModal.svelte";
  import BorrowedBooksCard from "./BorrowedBooksCard.svelte";
  import StudentEditModal from "./StudentEditModal.svelte";
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
          
          <div class="flex justify-end mb-2">
            {#if (role === 'admin' || role === 'mitarbeiter') && profile.entliehene_buecher?.length > 0}
              <button onclick={downloadKontoauszugPDF} disabled={kontoauszugPdfLoading} class="px-5 py-2.5 bg-blue-50 text-blue-600 hover:bg-blue-100 disabled:opacity-50 rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer flex items-center gap-2">
                {#if kontoauszugPdfLoading}
                  <div class="w-4 h-4 border-2 border-blue-400 border-t-blue-700 rounded-full animate-spin"></div>
                {:else}
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
                {/if}
                Kontoauszug drucken
              </button>
            {/if}
          </div>

          <div class="col-span-1 md:col-span-1 relative flex flex-col gap-6 h-full min-h-[400px] animate-fade-in">
            <BorrowedBooksCard 
              books={profile.entliehene_buecher || []} 
              {onReturnClick} 
              onDamageClick={role === 'admin' || role === 'mitarbeiter' ? openDamageModal : undefined}
            />
            
            <StudentVormerkungenCard bind:vormerkungen />
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
    </div>
  </div>
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

{#if showEditModal}
  <StudentEditModal student={profile} onClose={() => showEditModal = false} onSave={handleSaveEdit} />
{/if}
