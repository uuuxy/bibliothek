<script>
  import { apiFetch } from "./apiFetch.js";
  import StudentProfile from "./StudentProfile.svelte";
  import KioskReservationModal from "./KioskReservationModal.svelte";
  import KioskChecklistModal from "./KioskChecklistModal.svelte";
  import KioskDamageModal from "./KioskDamageModal.svelte";
  import { onMount, tick } from "svelte";
  import { playSuccessBeep, playErrorBeep } from "./audio.js";

  // ── States ──────────────────────────────────────────────────────────
  /** @type {any} */
  let activeStudent = $state(null);
  let studentInputVal = $state("");
  let bookInputVal = $state("");

  /** @type {any[]} */
  let scannedBooks = $state([]);

  /** @type {any} */
  let toast = $state(null);
  let screenFlash = $state(""); // "success" | "error" | ""
  let isShaking = $state(false);
  let isScanningStudent = $state(false);
  let isScanningBook = $state(false);

  // ── Damage Modal State ──────────────────────────────────────────────
  /** @type {any} */
  let returnedBook = $state(null);
  let returnedLoanId = $state("");
  let showDamageInput = $state(false);
  let damageDescription = $state("");
  let isSubmittingDamage = $state(false);

  // ── Vormerken Modal State ───────────────────────────────────────────
  let showVormerkenModal = $state(false);
  let vormerkenQuery = $state("");
  /** @type {any[]} */
  let vormerkenResults = $state([]);
  let isSearchingVormerken = $state(false);
  let isSubmittingVormerken = $state(false);

  // ── Geraete Checklist Modal State ─────────────────────────────────────
  let showChecklistModal = $state(false);
  /** @type {any} */
  let pendingGeraet = $state(null);
  let pendingGeraetQuery = $state("");
  let checklistItems = $derived.by(() => {
    if (!pendingGeraet || !pendingGeraet.zubehoer) return [];
    return pendingGeraet.zubehoer.split(",").map(i => i.trim()).filter(Boolean);
  });
  let checkedItems = $state(new Set());
  let isSubmittingChecklist = $state(false);

  // ── Derived State ───────────────────────────────────────────────────
  let systemSettings = $state({ max_ausleihen_schueler: 5 });

  let activeLoansCount = $derived(activeStudent?.active_loans?.length || 0);
  let isLimitReached = $derived(activeLoansCount >= systemSettings.max_ausleihen_schueler);

  // A student is blocked if they have overdue books (active_loans that are overdue)
  let isStudentBlocked = $derived.by(() => {
    if (!activeStudent) return false;
    const now = new Date().getTime();
    return activeStudent.active_loans?.some((/** @type {any} */ loan) => {
      if (!loan.rueckgabe_frist) return false;
      const frist = new Date(loan.rueckgabe_frist).getTime();
      return now > frist;
    }) ?? false;
  });



  // ── Logic ───────────────────────────────────────────────────────────
  function triggerFlash(/** @type {"success"|"error"} */ type, msg = "") {
    screenFlash = type;
    if (type === "error") {
      isShaking = true;
      playErrorBeep();
      setTimeout(() => isShaking = false, 500);
    } else {
      playSuccessBeep();
    }
    if (msg) toast = { type, message: msg };
    setTimeout(() => { screenFlash = ""; }, 500);
    setTimeout(() => { toast = null; }, 4000);
  }

  function clearSession() {
    activeStudent = null;
    scannedBooks = [];
    studentInputVal = "";
    bookInputVal = "";
    focusStudentInput();
  }

  function focusStudentInput() {
    tick().then(() => document.getElementById("kiosk-student-input")?.focus());
  }

  function focusBookInput() {
    tick().then(() => document.getElementById("kiosk-book-input")?.focus());
  }

  async function handleStudentSubmit() {
    const val = studentInputVal.trim();
    if (!val) return;
    isScanningStudent = true;
    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: val })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      if (data.type === "student") {
        activeStudent = data.student;
        scannedBooks = [];
        triggerFlash("success");
        if (isStudentBlocked) {
          triggerFlash("error", "Ausleihsperre! Überfällige Mahnungen vorhanden.");
        } else {
          focusBookInput();
        }
      } else {
        throw new Error("Barcode ist kein Schülerausweis.");
      }
    } catch (e) {
      triggerFlash("error", e instanceof Error ? e.message : "Schüler nicht gefunden.");
      studentInputVal = "";
      focusStudentInput();
    } finally {
      isScanningStudent = false;
    }
  }

  async function handleBookSubmit() {
    const val = bookInputVal.trim();
    bookInputVal = "";
    if (!val || !activeStudent || isStudentBlocked) return;
    
    isScanningBook = true;
    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: val, active_student_id: activeStudent.id })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      if (data.type === "ausleihe") {
        scannedBooks = [data.book, ...scannedBooks];
        triggerFlash("success");
        if (data.book.zustand_notiz) {
          toast = { type: "error", message: `Achtung: Bekannter Mangel: ${data.book.zustand_notiz}` };
        }
      } else if (data.type === "rueckgabe") {
        if (data.has_vormerkung) {
          triggerFlash("error");
          toast = { type: "error", message: `ACHTUNG: Reserviert für ${data.vormerkung_user || 'eine/n Schüler/in'}! Bitte gesondert zurücklegen.` };
          playErrorBeep();
          setTimeout(playErrorBeep, 400);
          returnedBook = null;
        } else {
          returnedBook = data.book || data.geraet;
          returnedLoanId = data.loan_id || (data.loanID ? data.loanID : "");
          showDamageInput = false;
          damageDescription = "";
          playSuccessBeep();
        }
      } else if (data.type === "geraet_check") {
        pendingGeraet = data.geraet;
        pendingGeraetQuery = val;
        checkedItems = new Set();
        showChecklistModal = true;
      } else {
        throw new Error("Unerwartete Antwort vom Server.");
      }
    } catch (e) {
      triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Buchen.");
      focusBookInput();
    } finally {
      isScanningBook = false;
    }
  }

  function handleDamageOk() {
    returnedBook = null;
    triggerFlash("success", "Buch zurückgegeben!");
    focusBookInput();
  }

  async function handleDamageSubmit() {
    if (!damageDescription.trim() || !returnedBook) return;
    isSubmittingDamage = true;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${returnedBook.id}/defekt`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          loan_id: returnedLoanId || undefined, 
          schueler_id: activeStudent?.id || undefined,
          betrag: 0,
          beschreibung: damageDescription.trim()
        })
      });
      if (!res.ok) throw new Error(await res.text());
      triggerFlash("success", "Mangel gespeichert! Exemplar gesperrt.");
      returnedBook = null;
    } catch(e) {
      triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Speichern des Mangels");
    } finally {
      isSubmittingDamage = false;
      focusBookInput();
    }
  }

  async function handleVormerkenSearch() {
    if (!vormerkenQuery.trim()) return;
    isSearchingVormerken = true;
    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: vormerkenQuery })
      });
      if (!res.ok) throw new Error("Fehler bei der Suche");
      const data = await res.json();
      vormerkenResults = data.search_results || [];
    } catch(e) {
      triggerFlash("error", "Suche fehlgeschlagen");
    } finally {
      isSearchingVormerken = false;
    }
  }

  async function handleVormerkenSubmit(/** @type {string} */ titelId) {
    if (!activeStudent) return;
    isSubmittingVormerken = true;
    try {
      const res = await apiFetch("/api/vormerkungen", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ titel_id: titelId, schueler_id: activeStudent.id, notiz: "Vorgemerkt im Kiosk" })
      });
      if (!res.ok) throw new Error(await res.text());
      triggerFlash("success", "Erfolgreich vorgemerkt!");
      showVormerkenModal = false;
      vormerkenQuery = "";
      vormerkenResults = [];
    } catch(e) {
      triggerFlash("error", "Fehler beim Vormerken");
    } finally {
      isSubmittingVormerken = false;
    }
  }

  async function handleChecklistSubmit() {
    if (!pendingGeraet || !activeStudent || checklistItems.length !== checkedItems.size) return;
    isSubmittingChecklist = true;
    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          query: pendingGeraetQuery, 
          active_student_id: activeStudent.id,
          confirmed_checklist: true
        })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      
      if (data.type === "ausleihe") {
        scannedBooks = [data.geraet, ...scannedBooks];
        triggerFlash("success");
      } else if (data.type === "rueckgabe") {
        returnedBook = data.geraet;
        returnedLoanId = data.loan_id || (data.loanID ? data.loanID : "");
        showDamageInput = false;
        damageDescription = "";
        playSuccessBeep();
      }
      showChecklistModal = false;
      pendingGeraet = null;
    } catch (e) {
      triggerFlash("error", e instanceof Error ? e.message : "Fehler beim Buchen des Geräts.");
    } finally {
      isSubmittingChecklist = false;
      focusBookInput();
    }
  }

  onMount(async () => {
    try {
      const res = await apiFetch("/api/einstellungen");
      if (res.ok) {
        systemSettings = await res.json();
      }
    } catch(e) {}
    focusStudentInput();
  });
</script>

<!-- Flash Overlay -->
{#if screenFlash}
  <div class="fixed inset-0 z-50 pointer-events-none transition-colors duration-300
    {screenFlash === 'success' ? 'bg-emerald-500/20' : 'bg-rose-500/30'}"></div>
{/if}

<!-- Toast -->
{#if toast}
  <div class="fixed top-8 left-1/2 -translate-x-1/2 z-50 p-4 rounded-xl shadow-xl text-white font-medium
    {toast.type === 'error' ? 'bg-rose-600' : 'bg-emerald-600'}">
    {toast.message}
  </div>
{/if}

<div class="max-w-4xl mx-auto w-full space-y-8 relative">
  <!-- 1. Schülerausweis Scan-Bereich -->
  {@render studentScanSection()}

  <!-- 2. Profil & 3. Buch-Scan (Nur wenn Schüler aktiv) -->
  {#if activeStudent}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <StudentProfile student={activeStudent} onDeselect={clearSession} />
        <button class="mt-4 w-full py-2 bg-slate-200 hover:bg-slate-300 text-slate-700 rounded-lg font-medium transition-colors"
                onclick={clearSession}>
          Sitzung beenden (Anderen Schüler scannen)
        </button>
      </div>

      <div class="space-y-6">
        <!-- Ausleih-Sperre Meldung -->
        {#if isStudentBlocked}
          <div class="bg-rose-100 border border-rose-200 text-rose-800 p-4 rounded-xl flex items-start space-x-3">
            <svg class="w-6 h-6 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
            <div>
              <h3 class="font-bold">Ausleihsperre aktiv</h3>
              <p class="text-sm">Dieser Schüler hat noch überfällige Mahnungen offen und darf keine neuen Medien ausleihen.</p>
            </div>
          </div>
        {:else}
          <!-- Buch-Scanner Input -->
          <div class="bg-white p-6 rounded-2xl shadow-sm border border-slate-200 {isShaking ? 'animate-shake' : ''}">
            <h3 class="text-lg font-bold text-slate-800 mb-4">Medien scannen</h3>
            
            {#if isLimitReached}
              <div class="mb-4 bg-red-50 border border-red-200 text-red-800 p-3 rounded-xl text-sm flex items-start space-x-2">
                <svg class="w-5 h-5 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                <span>Limit von {systemSettings.max_ausleihen_schueler} Medien erreicht. Keine weitere Ausleihe möglich!</span>
              </div>
            {/if}

            <form onsubmit={(e) => { e.preventDefault(); handleBookSubmit(); }}>
              <input type="text" id="kiosk-book-input" bind:value={bookInputVal} disabled={isScanningBook || isLimitReached}
                     placeholder="Buch-Barcode hier scannen..." autocomplete="off"
                     class="w-full bg-slate-50 border-2 border-emerald-200 focus:border-emerald-500 focus:ring-4 focus:ring-emerald-500/20 rounded-xl px-5 py-4 text-xl font-medium outline-none transition-all placeholder:text-slate-400 disabled:opacity-50 disabled:cursor-not-allowed" />
            </form>
            
            <button onclick={() => showVormerkenModal = true} class="mt-4 w-full py-3 bg-amber-100 hover:bg-amber-200 text-amber-800 border border-amber-200 rounded-xl font-semibold transition-colors flex items-center justify-center space-x-2 shadow-sm">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
              <span>Medium vormerken (Warteliste)</span>
            </button>
          </div>
          
          <!-- Scanned Books List -->
          {#if scannedBooks.length > 0}
            <div class="bg-white p-4 rounded-2xl shadow-sm border border-slate-200 space-y-3">
              <h4 class="font-semibold text-slate-600 text-sm uppercase tracking-wider">Aktuell verbucht</h4>
              {#each scannedBooks as book (book.id)}
                <div class="flex items-center justify-between p-3 bg-emerald-50 rounded-xl border border-emerald-100">
                  <div class="flex-1 min-w-0">
                    <p class="font-medium text-slate-800 truncate">{book.titel}</p>
                    <p class="text-sm text-slate-500">{book.barcode_id}</p>
                  </div>
                  <svg class="w-5 h-5 text-emerald-600 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                </div>
              {/each}
            </div>
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</div>

<KioskReservationModal bind:showVormerkenModal bind:vormerkenQuery {isSearchingVormerken} {vormerkenResults} {isSubmittingVormerken} {handleVormerkenSearch} {handleVormerkenSubmit} />
<KioskChecklistModal bind:showChecklistModal bind:pendingGeraet {checklistItems} bind:checkedItems {isSubmittingChecklist} {handleChecklistSubmit} />
<KioskDamageModal bind:returnedBook bind:showDamageInput bind:damageDescription {isSubmittingDamage} {handleDamageOk} {handleDamageSubmit} />

{#snippet studentScanSection()}
  {#if !activeStudent}
    <div class="bg-white p-8 rounded-2xl shadow-sm border border-slate-200 text-center max-w-xl mx-auto {isShaking ? 'animate-shake' : ''}">
      <div class="w-16 h-16 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center mx-auto mb-6">
        <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/></svg>
      </div>
      <h2 class="text-2xl font-bold text-slate-800 mb-2">Ausleihe starten</h2>
      <p class="text-slate-500 mb-8">Scanne zuerst den Schülerausweis, um das Profil aufzurufen.</p>
      <form onsubmit={(e) => { e.preventDefault(); handleStudentSubmit(); }}>
        <input type="text" id="kiosk-student-input" bind:value={studentInputVal} disabled={isScanningStudent}
               placeholder="S-XXXXXX scannen..." autocomplete="off"
               class="w-full bg-slate-50 border-2 border-blue-200 focus:border-blue-500 focus:ring-4 focus:ring-blue-500/20 rounded-xl px-5 py-4 text-xl font-medium outline-none transition-all text-center placeholder:text-slate-400" />
      </form>
    </div>
  {/if}
{/snippet}

<style>
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-8px); }
    75% { transform: translateX(8px); }
  }
  .animate-shake {
    animation: shake 0.3s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
