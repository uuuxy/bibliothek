<script>
  import { apiFetch } from "./apiFetch.js";
  import StudentProfile from "./StudentProfile.svelte";
  import { onMount, tick } from "svelte";

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

  // ── Derived State ───────────────────────────────────────────────────
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

  // ── Audio Feedback ──────────────────────────────────────────────────
  /** @type {AudioContext | null} */
  let _audioCtx = null;
  function getAudioCtx() {
    if (!_audioCtx) {
      _audioCtx = new (window.AudioContext || /** @type {any} */(window).webkitAudioContext)();
    }
    return _audioCtx;
  }

  function playSuccessBeep() {
    try {
      const ctx = getAudioCtx();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = "sine";
      osc.frequency.setValueAtTime(880, ctx.currentTime);
      osc.frequency.exponentialRampToValueAtTime(1320, ctx.currentTime + 0.1);
      gain.gain.setValueAtTime(0.1, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.1);
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start();
      osc.stop(ctx.currentTime + 0.1);
    } catch(e) {}
  }

  function playErrorBeep() {
    try {
      const ctx = getAudioCtx();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = "sawtooth";
      osc.frequency.setValueAtTime(150, ctx.currentTime);
      gain.gain.setValueAtTime(0.2, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start();
      osc.stop(ctx.currentTime + 0.3);
    } catch(e) {}
  }

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
          // Play loud beep twice
          playErrorBeep();
          setTimeout(playErrorBeep, 400);
          returnedBook = null; // Don't show damage modal if reserved
        } else {
          returnedBook = data.book;
          returnedLoanId = data.loan_id || (data.loanID ? data.loanID : "");
          showDamageInput = false;
          damageDescription = "";
          playSuccessBeep();
        }
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

  onMount(() => focusStudentInput());
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
            <form onsubmit={(e) => { e.preventDefault(); handleBookSubmit(); }}>
              <input type="text" id="kiosk-book-input" bind:value={bookInputVal} disabled={isScanningBook}
                     placeholder="Buch-Barcode hier scannen..." autocomplete="off"
                     class="w-full bg-slate-50 border-2 border-emerald-200 focus:border-emerald-500 focus:ring-4 focus:ring-emerald-500/20 rounded-xl px-5 py-4 text-xl font-medium outline-none transition-all placeholder:text-slate-400" />
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

{@render reservationModal()}
{@render damageModal()}

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

{#snippet reservationModal()}
  {#if showVormerkenModal}
    <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm pointer-events-none" onclick={() => showVormerkenModal = false}></div>
      <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-xl w-full relative z-10 border border-slate-200 flex flex-col max-h-[80vh]">
        <h3 class="text-xl font-bold text-slate-800 mb-4">Titel vormerken</h3>
        <p class="text-sm text-slate-500 mb-4">Suche nach ISBN oder Titel, um das Medium auf die Warteliste zu setzen.</p>
        
        <form onsubmit={(e) => { e.preventDefault(); handleVormerkenSearch(); }} class="flex gap-2 mb-6">
          <input type="text" bind:value={vormerkenQuery} placeholder="Titel oder ISBN eingeben..." class="flex-1 bg-slate-50 border border-slate-200 rounded-xl px-4 py-2 outline-none focus:border-amber-500 focus:ring-2 focus:ring-amber-500/20 transition-all" />
          <button type="submit" disabled={isSearchingVormerken || !vormerkenQuery} class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-xl font-semibold transition-colors disabled:opacity-50">Suchen</button>
        </form>

        <div class="flex-1 overflow-y-auto space-y-2 min-h-0">
          {#if isSearchingVormerken}
            <p class="text-center text-slate-500 py-4">Suche läuft...</p>
          {:else if vormerkenResults.length > 0}
            {#each vormerkenResults as res}
              <div class="flex items-center justify-between p-3 bg-slate-50 rounded-xl border border-slate-100">
                <div class="flex-1 min-w-0 pr-4">
                  <h4 class="font-bold text-slate-800 truncate">{res.titel}</h4>
                  {#if res.isbn}<p class="text-sm text-slate-500">ISBN: {res.isbn}</p>{/if}
                </div>
                <button onclick={() => handleVormerkenSubmit(res.id)} disabled={isSubmittingVormerken} class="shrink-0 px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white text-sm font-bold rounded-lg transition-colors shadow-sm disabled:opacity-50">
                  Vormerken
                </button>
              </div>
            {/each}
          {:else if vormerkenQuery && !isSearchingVormerken}
            <p class="text-center text-slate-500 py-4">Keine Titel gefunden.</p>
          {/if}
        </div>

        <div class="mt-6 pt-4 border-t border-slate-100 text-right">
          <button onclick={() => showVormerkenModal = false} class="px-4 py-2 text-slate-600 hover:bg-slate-100 font-semibold rounded-xl transition-colors">Schließen</button>
        </div>
      </div>
    </div>
  {/if}
{/snippet}

{#snippet damageModal()}
  {#if returnedBook}
    <div class="fixed inset-0 z-60 flex items-center justify-center p-4">
      <div class="absolute inset-0 bg-slate-900/40 backdrop-blur-sm pointer-events-none"></div>
      <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-md w-full relative z-10 border border-slate-200">
        <h3 class="text-xl font-bold text-slate-800 mb-2">Zustand in Ordnung?</h3>
        <p class="text-sm text-slate-500 mb-6">Bitte überprüfe <strong>{returnedBook.titel}</strong> ({returnedBook.barcode_id}) auf Schäden.</p>
        
        {#if !showDamageInput}
          <div class="grid grid-cols-2 gap-4">
            <button onclick={() => showDamageInput = true} class="py-3 px-4 rounded-xl bg-rose-50 hover:bg-rose-100 text-rose-700 font-bold transition-colors">Nein, Mangel melden</button>
            <button onclick={handleDamageOk} class="py-3 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition-colors shadow-sm focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 outline-none">Ja, alles okay</button>
          </div>
        {:else}
          <div class="space-y-4">
            <label for="damage-description" class="block text-sm font-semibold text-slate-700">Art des Mangels (Notiz)</label>
            <textarea id="damage-description" bind:value={damageDescription} rows="3" placeholder="z.B. Wasserschaden, Seite 15 fehlt..." class="w-full bg-slate-50 border border-slate-200 rounded-xl p-3 text-slate-800 focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20 outline-none resize-none transition-all"></textarea>
            <div class="flex gap-3 justify-end pt-2">
              <button onclick={() => showDamageInput = false} disabled={isSubmittingDamage} class="px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 rounded-xl transition-colors">Abbrechen</button>
              <button onclick={handleDamageSubmit} disabled={isSubmittingDamage || !damageDescription.trim()} class="px-4 py-2 text-sm font-bold text-white bg-rose-600 hover:bg-rose-700 disabled:opacity-50 rounded-xl transition-colors shadow-sm">Mangel speichern</button>
            </div>
          </div>
        {/if}
      </div>
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
