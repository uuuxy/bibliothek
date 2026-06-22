<script>
  import WebcamCapture from "./WebcamCapture.svelte";
  import DamageReportModal from "./DamageReportModal.svelte";
  import StudentLockModal from "./StudentLockModal.svelte";
  import BorrowedBooksCard from "./BorrowedBooksCard.svelte";
  import StudentEditSheet from './StudentEditSheet.svelte';
  import StudentProfileCard from "./StudentProfileCard.svelte";
  import StudentPrintCard from "./StudentPrintCard.svelte";
  import StudentProfileDeleteModal from "./StudentProfileDeleteModal.svelte";
  import StudentProfileStammdaten from "./StudentProfileStammdaten.svelte";
  import StudentVormerkungenCard from "./StudentVormerkungenCard.svelte";
  import StudentProfileActions from "./StudentProfileActions.svelte";
  import StudentPrintReceipt from "./StudentPrintReceipt.svelte";
  import { useStudentProfile } from "./useStudentProfile.svelte.js";

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

  const st = useStudentProfile();

  $effect(() => {
    if (student?.id) st.fetchProfile(student.id);
  });

  export function reloadProfile() {
    st.fetchProfile(student?.id);
  }

  function printCard() {
    const styleEl = document.createElement("style");
    styleEl.textContent = "@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }";
    document.head.appendChild(styleEl);
    document.body.setAttribute("data-print-mode", "card-single");
    window.print();
    document.head.removeChild(styleEl);
    document.body.removeAttribute("data-print-mode");
  }
</script>

{#if st.loading}
  <div class="w-full py-12 flex justify-center items-center">
    <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if st.profile}
  {#if st.globalErrorToast}
    <div class="fixed top-6 right-6 z-50 px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold animate-fade-in bg-rose-600 text-white flex items-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd" />
      </svg>
      {st.globalErrorToast}
    </div>
  {/if}

  {#if !st.showEditModal}
  <div class="w-full grid grid-cols-1 lg:grid-cols-3 gap-6 items-start text-slate-800 animate-fade-in no-print print:hidden font-sans">
    <!-- Left Column Profile Card -->
    <StudentProfileCard 
      bind:profile={st.profile}
      {role}
      timestamp={st.timestamp}
      bind:showWebcam={st.showWebcam}
      bind:showDeleteConfirm={st.showDeleteConfirm}
      {onDeselect}
      onPrint={printCard}
      {leftActions}
    />

    <!-- Right: Timeline / Loans List / Stammdaten (2 cols) -->
    <div class="lg:col-span-2 space-y-6 flex flex-col h-full">

      {#if role === 'admin' || role === 'mitarbeiter'}
        <StudentProfileActions 
          profile={st.profile}
          kontoauszugPdfLoading={st.kontoauszugPdfLoading}
          rechnungPdfLoading={st.rechnungPdfLoading}
          downloadKontoauszugPDF={st.downloadKontoauszugPDF}
          downloadRechnungPDF={st.downloadRechnungPDF}
          showLockModal={() => st.showLockModal = true}
        />
      {/if}

      <!-- Tabs -->
      <div class="flex gap-6 border-b border-slate-200">
        <button onclick={() => st.activeTab = "ausleihen"} class="pb-2 text-sm font-bold transition-all border-b-2 {st.activeTab === 'ausleihen' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-600 hover:text-slate-800'}">
          Ausleihen & Historie
        </button>
        <button onclick={() => st.activeTab = "stammdaten"} class="pb-2 text-sm font-bold transition-all border-b-2 {st.activeTab === 'stammdaten' ? 'border-blue-600 text-blue-600' : 'border-transparent text-slate-600 hover:text-slate-800'}">
          Stammdaten & Adresse
        </button>
      </div>

      <div class="flex-1 relative">
        {#if st.activeTab === "ausleihen"}
          {@render rightTop?.()}
          <div class="col-span-1 md:col-span-1 relative flex flex-col gap-6 h-full min-h-[400px] animate-fade-in mt-4">
            <BorrowedBooksCard 
              books={st.profile.entliehene_buecher || []} 
              {onReturnClick} 
              onDamageClick={role === 'admin' || role === 'mitarbeiter' ? st.openDamageModal : undefined}
            />
            
            {#if st.vormerkungen.length > 0}
              <StudentVormerkungenCard bind:vormerkungen={st.vormerkungen} />
            {/if}
          </div>
        {:else if st.activeTab === "stammdaten"}
          <StudentProfileStammdaten 
            profile={st.profile} 
            {role} 
            rechnungPdfLoading={st.rechnungPdfLoading} 
            onDownloadRechnung={st.downloadRechnungPDF} 
            onEdit={() => st.showEditModal = true} 
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
          <button onclick={() => st.showDeleteConfirm = true} class="shrink-0 px-6 py-3 bg-white border border-rose-200 text-rose-600 hover:bg-rose-600 hover:text-white rounded-full text-sm font-bold transition-all shadow-sm hover:shadow cursor-pointer">
            Schüler archivieren / löschen
          </button>
        </div>
      {/if}
    </div>
  </div>
  {:else}
    <StudentEditSheet student={st.profile} {role} onClose={() => st.showEditModal = false} onSave={() => st.handleSaveEdit(student?.id)} />
  {/if}
{/if}

{#if st.showWebcam}
  <WebcamCapture studentId={st.profile.id} onCapture={() => st.handlePhotoCaptured(student?.id)} onClose={() => st.showWebcam = false} />
{/if}

{#if st.profile}
  <StudentPrintCard profile={st.profile} timestamp={st.timestamp} />
{/if}

<StudentProfileDeleteModal 
  open={st.showDeleteConfirm} 
  profile={st.profile} 
  onclose={() => st.showDeleteConfirm = false} 
  onsuccess={() => st.handleDeleteSuccess(onDeselect)} 
/>

{#if st.showDamageModal && st.damageBook}
  <DamageReportModal book={st.damageBook} isSubmitting={st.isSubmittingDamage} onCancel={() => st.showDamageModal = false} onSubmit={(r, a) => st.submitDamageReport(student?.id, r, a)} />
{/if}

<StudentLockModal bind:open={st.showLockModal} profile={st.profile} onsuccess={st.handleLockSuccess} />

{#if st.profile}
  <StudentPrintReceipt profile={st.profile} />
{/if}
