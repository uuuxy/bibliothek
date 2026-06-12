<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";
  import OfflineQueueBanner from "./OfflineQueueBanner.svelte";
  import CameraScanner from "./CameraScanner.svelte";
  import OmniboxInput from "./components/omnibox/OmniboxInput.svelte";
  import OmniboxResults from "./components/omnibox/OmniboxResults.svelte";
  import OmniboxScannedList from "./OmniboxScannedList.svelte";
  import OmniboxTeacherCard from "./OmniboxTeacherCard.svelte";
  import { playSoundSuccess, playSoundError } from "./audio.js";
  import { loadQueue, enqueueOfflineScan, flushOfflineQueue } from "./offlineQueue.js";

  let { onSelectBook } = $props();

  let activeStudent = $state(/** @type {any} */ (null));
  let activeTeacher = $state(/** @type {any} */ (null));
  let queryVal = $state("");

  let toast = $state(/** @type {any} */ (null));
  let flashBorder = $state(""); 
  let screenFlash = $state(""); // "success" | "error" | ""
  let lastFremdrueckgabe = $state(/** @type {any} */ (null));
  let studentProfileComponent = $state(/** @type {any} */ (null));
  let isShaking = $state(false);
  let vormerkungAlert = $state(/** @type {{titel?: string} | null} */ (null));
  let isOffline = $state(!navigator.onLine);
  let offlineQueueCount = $state(0);

  // Camera scanner state
  let showCamera = $state(false);
  let cameraScanner = $state(/** @type {any} */ (null));

  // Search State
  let debounceTimer = $state(/** @type {any} */ (null));
  let isDropdownOpen = $state(false);
  let unifiedSearchResults = $state({ students: /** @type {any[]} */ ([]), books: /** @type {any[]} */ ([]) });
  let selectedDropdownIndex = $state(-1);
  let totalDropdownItems = $derived(unifiedSearchResults.students.length + unifiedSearchResults.books.length);

  let isActive = $derived(!!(activeStudent || activeTeacher || isDropdownOpen));



  // ── Screen-edge flash (300 ms) ────────────────────────────────
  /** @param {"success"|"error"|"warning"} type */
  function triggerScreenFlash(type) {
    screenFlash = type;
    setTimeout(() => { screenFlash = ""; }, 300);
  }

  function triggerShake() {
    isShaking = true;
    setTimeout(() => { isShaking = false; }, 500);
  }

  /** @param {string} color */
  function triggerFlash(color) { flashBorder = color; setTimeout(() => { flashBorder = ""; }, 1000); }

  import { appState } from "../inventur/lib/store.svelte.js";

  $effect(() => {
    if (appState.triggerStudentScan) {
      queryVal = appState.triggerStudentScan;
      appState.triggerStudentScan = "";
      submitAction();
    }
  });



  onMount(() => {
    // SSE for live reload of student profile
    const source = new EventSource("/events");
    source.addEventListener("action", (e) => {
      try {
        const actionData = JSON.parse(/** @type {any} */ (e).data);
        if (activeStudent && actionData.student_id === activeStudent.id) {
          studentProfileComponent?.reloadProfile();
        }
      } catch (err) {
        console.error("SSE parsing error in Omnibox:", err);
      }
    });

    // Offline / Online detection
    const handleOnline = async () => {
      isOffline = false;
      offlineQueueCount = await flushOfflineQueue(showToast);
    };
    const handleOffline = () => { isOffline = true; };
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    // Initialise queue count badge
    loadQueue().then(q => { offlineQueueCount = q.length; });

    return () => {
      source.close();
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  });

  $effect(() => {
    if (!isActive && !isDropdownOpen && !showCamera) {
      setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
    }
  });

  $effect(() => {
    if (toast) {
      const timer = setTimeout(() => { toast = null; }, 4000);
      return () => clearTimeout(timer);
    }
  });

  $effect(() => {
    /** @param {KeyboardEvent} e */
    function handleKeyDown(e) {
      if (e.key === "Escape") {
        queryVal = "";
        activeStudent = null;
        activeTeacher = null;
        lastFremdrueckgabe = null;
        isDropdownOpen = false;
        if (showCamera) stopCamera();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  });

  /**
   * @param {string} message
   * @param {string} [type]
   */
  function showToast(message, type = "success") { toast = { message, type }; }

  function handleInput() {
    clearTimeout(debounceTimer);
    if (!queryVal.trim()) {
      isDropdownOpen = false;
      unifiedSearchResults = { students: [], books: [] };
      return;
    }
    debounceTimer = setTimeout(async () => {
      if (!queryVal.trim()) return;
      try {
        const res = await apiFetch(`/api/search?q=${encodeURIComponent(queryVal.trim())}`);
        if (res.ok) {
          unifiedSearchResults = await res.json();
          if (!unifiedSearchResults.students) unifiedSearchResults.students = [];
          if (!unifiedSearchResults.books) unifiedSearchResults.books = [];
          isDropdownOpen = unifiedSearchResults.students.length > 0 || unifiedSearchResults.books.length > 0;
          selectedDropdownIndex = -1;
        }
      } catch (err) {
        console.error("Search failed:", err);
      }
    }, 300);
  }



  /** @param {number} index */
  function selectDropdownItem(index) {
    const { students, books } = unifiedSearchResults;
    if (index < students.length) {
      const student = students[index];
      queryVal = student.barcode_id;
      isDropdownOpen = false;
      submitAction();
    } else {
      const book = books[index - students.length];
      queryVal = "";
      isDropdownOpen = false;
      if (onSelectBook) onSelectBook(book);
    }
  }

  /** @param {Event} [e] */
  async function submitAction(e) {
    if (e) e.preventDefault();
    if (isDropdownOpen && selectedDropdownIndex >= 0) {
      selectDropdownItem(selectedDropdownIndex);
      return;
    }

    const q = queryVal.trim();
    if (!q) return;

    queryVal = "";
    isDropdownOpen = false;
    lastFremdrueckgabe = null;

    // Re-focus immediately so USB scanner next scan is captured without delay
    setTimeout(() => document.getElementById("omnibox-input")?.focus(), 30);

    // ── Offline path: Schnellrückgabe (B- barcode, no context) ──
    // Only queue book scans without an active session; student/teacher lookups
    // are read-only and useless to replay offline.
    if (!navigator.onLine && q.startsWith("B-")) {
      offlineQueueCount = await enqueueOfflineScan(q, activeStudent?.id ?? null, activeTeacher?.id ?? null);
      triggerScreenFlash("warning");
      playSoundError();
      showToast(`📴 Offline: Barcode „${q}“ in Warteschlange gespeichert.`, "warning");
      triggerFlash("orange");
      return;
    }

    try {
      const res = await apiFetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query: q,
          active_student_id: activeStudent?.id,
          active_teacher_id: activeTeacher?.id
        }),
        signal: AbortSignal.timeout(8000) // 8 s timeout → WLAN dropout detection
      });

      if (!res.ok) throw new Error(await res.text() || "Aktion fehlgeschlagen");
      const data = await res.json();

      if (data.type === "student") {
        activeStudent = data.student;
        activeTeacher = null;
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
      } else if (data.type === "teacher") {
        activeTeacher = data.teacher;
        activeStudent = null;
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
        showToast(`📋 Handapparat-Sitzung gestartet für Lehrer/in ${data.teacher.vorname} ${data.teacher.nachname}`);
      } else if (data.type === "ausleihe") {


        if (data.fremdrueckgabe) {
          triggerScreenFlash("warning");
          playSoundError();
          triggerFlash("orange");
          const prevName = data.vorbesitzer
            ? `${data.vorbesitzer.vorname} ${data.vorbesitzer.nachname}`
            : `${data.vorbesitzer_user.vorname} ${data.vorbesitzer_user.nachname}`;
          lastFremdrueckgabe = { vorbesitzerName: prevName };
          showToast(`⚠️ Fremdrückgabe erfolgt (Vorbesitzer: ${prevName})`, "warning");
        } else {
          triggerScreenFlash("success");
          playSoundSuccess();
          triggerFlash("green");
          showToast(`📖 „${data.book.titel}" ausgeliehen an ${activeTeacher ? activeTeacher.vorname : activeStudent.vorname}.`);
        }
        studentProfileComponent?.reloadProfile();
      } else if (data.type === "rueckgabe") {
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");

        showToast(`📥 „${data.book.titel}" erfolgreich zurückgegeben.`);
        if (data.has_vormerkung) {
          vormerkungAlert = { titel: data.vormerkung_titel || data.book?.titel };
        }
        studentProfileComponent?.reloadProfile();

        if (data.student && !activeStudent && !activeTeacher) {
          activeStudent = data.student;
        } else if (data.teacher && !activeStudent && !activeTeacher) {
          activeTeacher = data.teacher;
        }
      } else if (data.type === "info") {
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
        showToast(data.message, "success");
        studentProfileComponent?.reloadProfile();
      } else if (data.type === "search_results") {
        triggerShake();
        showToast("Bitte wähle ein Ergebnis aus der Liste.", "warning");
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      const isTimeout = error?.name === "TimeoutError" || error?.name === "AbortError";

      // WLAN dropout: queue the scan if it was a book barcode
      if (isTimeout && q.startsWith("B-")) {
        offlineQueueCount = await enqueueOfflineScan(q, activeStudent?.id ?? null, activeTeacher?.id ?? null);
        triggerScreenFlash("error");
        playSoundError();
        triggerFlash("orange");
        showToast(`📴 Zeitüberschreitung – Barcode „${q}“ offline gespeichert (${offlineQueueCount} ausstehend).`, "warning");
        return;
      }

      triggerScreenFlash("error");
      playSoundError();

      if (q.startsWith("B-") && !activeStudent && !activeTeacher) {
        triggerShake();
        showToast("Bitte zuerst Schüler scannen", "warning");
      } else {
        showToast(error.message || String(error), "error");
        triggerFlash("orange");
      }
    }
  }

  // ── Undo return ──────────────────────────────────────────────
  /** @param {string} loanId @param {number} entryIndex */
  async function undoReturn(loanId, entryIndex) {
    if (!loanId) return;
    try {
      const res = await apiFetch(`/api/ausleihen/${loanId}/rueckgabe`, { method: "DELETE" });
      if (!res.ok) {
        const msg = await res.text();
        showToast(msg || "Undo fehlgeschlagen", "error");
        return;
      }

      showToast("↩️ Rückgabe rückgängig gemacht.", "success");
      studentProfileComponent?.reloadProfile();
    } catch {
      showToast("Undo fehlgeschlagen", "error");
    }
  }

  // ── Mark defect ──────────────────────────────────────────────
  /** @param {any} entry @param {number} entryIndex */
  async function markDefekt(entry, entryIndex) {
    if (!entry.book?.id) return;
    const betrag = parseFloat(prompt("Schadensgebühr (€):", "10.00") ?? "0");
    if (isNaN(betrag) || betrag < 0) { showToast("Ungültiger Betrag", "error"); return; }
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${entry.book.id}/defekt`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          loan_id: entry.loanId ?? null,
          schueler_id: entry.schuelerID ?? null,
          betrag,
          beschreibung: "Defekt/Schaden bei Rückgabe gemeldet"
        })
      });
      if (!res.ok) { showToast(await res.text() || "Fehler", "error"); return; }

      showToast(`🔴 „${entry.book.titel}" als defekt markiert. Schadensfaelle erstellt.`, "warning");
    } catch {
      showToast("Fehler beim Melden des Schadens", "error");
    }
  }

  // ── HTML5 Camera Scanner ──────────────────────────────────────
  async function startCamera() {
    showCamera = true;
    await new Promise(r => setTimeout(r, 80));
    try {
      const { Html5Qrcode } = await import("html5-qrcode");
      cameraScanner = new Html5Qrcode("camera-scan-region");
      await cameraScanner.start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 260, height: 120 } },
        (/** @type {string} */ decodedText) => {
          queryVal = decodedText.trim();
          stopCamera();
          submitAction();
        },
        () => {}
      );
    } catch {
      showCamera = false;
      showToast("Kamera konnte nicht gestartet werden", "error");
    }
  }

  async function stopCamera() {
    showCamera = false;
    if (cameraScanner) {
      try { await cameraScanner.stop(); } catch {}
      try { cameraScanner.clear(); } catch {}
      cameraScanner = null;
    }
    setTimeout(() => document.getElementById("omnibox-input")?.focus(), 50);
  }
</script>

<!-- ── Screen-edge flash overlay (Apple-style full-border glow) ── -->
{#if screenFlash}
  <div class="screen-flash screen-flash--{screenFlash}" aria-hidden="true"></div>
{/if}

<!-- ── Offline / Queue banner ── -->
<OfflineQueueBanner 
  {isOffline} 
  {offlineQueueCount} 
  flushOfflineQueue={async () => { offlineQueueCount = await flushOfflineQueue(showToast); }} 
/>

<div class="w-full mx-auto transition-all duration-500 ease-in-out {isActive ? 'w-full pt-4 justify-start' : 'max-w-2xl min-h-[60vh] justify-center'} flex flex-col items-center space-y-6">
  <div class="w-full transition-all duration-500 {isActive ? 'sticky -top-4 z-30 bg-slate-50/95 backdrop-blur-md py-4' : ''}">
    <form onsubmit={submitAction} class="w-full relative bg-white py-5 px-8 rounded-3xl border border-slate-200 shadow-2xl no-print transition-all duration-500 focus-within:border-blue-500 focus-within:ring-4 focus-within:ring-blue-50 {isActive ? 'scale-100' : 'scale-105'} {isShaking ? 'animate-shake border-rose-400' : ''} {flashBorder === 'green' ? 'ring-4 ring-emerald-500/10 border-emerald-400' : flashBorder === 'orange' ? 'ring-4 ring-amber-500/10 border-amber-400' : ''}">
      <OmniboxInput 
        bind:queryVal
        {isDropdownOpen}
        {totalDropdownItems}
        {isActive}
        {showCamera}
        onInput={handleInput}
        onSelect={selectDropdownItem}
        onIndexChange={(idx) => selectedDropdownIndex = idx}
        onEscape={() => isDropdownOpen = false}
        onToggleCamera={showCamera ? stopCamera : startCamera}
      />

      {#if isDropdownOpen && totalDropdownItems > 0}
        <OmniboxResults 
          {unifiedSearchResults} 
          {selectedDropdownIndex} 
          onSelect={selectDropdownItem} 
        />
      {/if}
    </form>
  </div>

  <!-- HTML5 Kamera-Scanner (Mobile) -->
  {#if showCamera}
    <CameraScanner {stopCamera} bind:queryVal={queryVal} {submitAction} />
  {/if}

  {#if activeStudent}
    {#if lastFremdrueckgabe}
      <div class="w-full max-w-xl p-3 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-medium flex items-center space-x-2 animate-slide-up no-print mb-2">
        <span>⚠️</span>
        <span><strong>Fremdrückgabe:</strong> Buch wurde von <strong>{lastFremdrueckgabe.vorbesitzerName}</strong> zurückgegeben und für {activeStudent.vorname} verbucht.</span>
      </div>
    {/if}
    <StudentProfile bind:this={studentProfileComponent} student={activeStudent} onDeselect={() => { activeStudent = null; lastFremdrueckgabe = null; }} onReturnClick={(barcode) => { queryVal = barcode; submitAction(); }} />
  {:else if activeTeacher}
    <OmniboxTeacherCard teacher={activeTeacher} onDeselect={() => { activeTeacher = null; lastFremdrueckgabe = null; }} />
  {/if}
</div>

<!-- Toast notifications -->
<div class="fixed top-24 left-1/2 -translate-x-1/2 w-full max-w-lg z-50 space-y-3 px-4 pointer-events-none">
  {#if toast}
    <div class="p-4 rounded-xl shadow-xl flex items-center space-x-3 backdrop-blur-md animate-slide-down pointer-events-auto border
      {toast.type === 'success' ? 'bg-emerald-50 border-emerald-100/50 text-emerald-700'
      : toast.type === 'warning' ? 'bg-amber-50 border-amber-100/50 text-amber-700'
      : 'bg-rose-50 border-rose-100/50 text-rose-700'}">
      <span class="text-sm font-semibold">{toast.message}</span>
    </div>
  {/if}
</div>

{#if vormerkungAlert}
  <div class="fixed inset-0 bg-rose-900/80 backdrop-blur-sm z-100 flex items-center justify-center p-4">
    <div class="bg-white rounded-3xl p-8 max-w-md w-full text-center shadow-2xl border-4 border-rose-500">
      <div class="text-6xl mb-4">🚨</div>
      <h2 class="text-2xl font-extrabold text-rose-700 mb-2">Achtung! Vorgemerkt!</h2>
      <p class="text-slate-700 mb-2">Dieses Medium wurde reserviert.</p>
      <p class="font-bold text-slate-900 mb-6">Separat zurücklegen!</p>
      {#if vormerkungAlert.titel}
        <p class="text-sm text-slate-500 mb-6">„{vormerkungAlert.titel}"</p>
      {/if}
      <button onclick={() => { vormerkungAlert = null; }}
        class="px-8 py-3 bg-rose-600 hover:bg-rose-700 text-white font-bold rounded-xl text-lg transition-colors cursor-pointer w-full">
        Verstanden
      </button>
    </div>
  </div>
{/if}




<style>
  /* ── Screen-edge flash overlay ─────────────────────────────── */
  .screen-flash {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 9999;
    border-radius: 0;
    animation: screen-flash-fade 300ms ease-out forwards;
  }
  .screen-flash--success {
    box-shadow:
      inset 0 0 0 6px rgba(16, 185, 129, 0.85),  /* emerald-500 */
      inset 0 0 60px 10px rgba(16, 185, 129, 0.18);
  }
  .screen-flash--error {
    box-shadow:
      inset 0 0 0 6px rgba(239, 68, 68, 0.85),   /* red-500 */
      inset 0 0 60px 10px rgba(239, 68, 68, 0.18);
  }
  .screen-flash--warning {
    box-shadow:
      inset 0 0 0 6px rgba(245, 158, 11, 0.85),  /* amber-500 */
      inset 0 0 60px 10px rgba(245, 158, 11, 0.18);
  }
  @keyframes screen-flash-fade {
    0%   { opacity: 1; }
    60%  { opacity: 0.9; }
    100% { opacity: 0; }
  }

  /* ── Shake animation ─────────────────────────────────────── */
  @keyframes shake {
    0%, 100% { transform: translate(0, 0) scale(1.05); }
    15%, 45%, 75% { transform: translate(-8px, 0) scale(1.05); }
    30%, 60% { transform: translate(8px, 0) scale(1.05); }
  }
  @keyframes activeShake {
    0%, 100% { transform: translate(0, 0) scale(1); }
    15%, 45%, 75% { transform: translate(-8px, 0) scale(1); }
    30%, 60% { transform: translate(8px, 0) scale(1); }
  }
  .animate-shake {
    animation: shake 0.4s cubic-bezier(.36,.07,.19,.97) both;
  }
  :global(.pt-4) .animate-shake {
    animation: activeShake 0.4s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
