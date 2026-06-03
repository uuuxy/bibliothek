<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";

  let { onSelectBook } = $props();

  let activeStudent = $state(/** @type {any} */ (null));
  let activeTeacher = $state(/** @type {any} */ (null));
  let queryVal = $state("");
  let scannedBooks = $state(/** @type {any[]} */ ([]));
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

  // ── Web Audio API: synthesised sounds (no files needed) ──────
  /** @type {AudioContext | null} */
  let _audioCtx = null;
  function getAudioCtx() {
    if (!_audioCtx) {
      _audioCtx = new (window.AudioContext || /** @type {any} */(window).webkitAudioContext)();
    }
    return _audioCtx;
  }

  /** Play a pleasant success "pling" – two sine tones, short attack/decay */
  function playSoundSuccess() {
    try {
      const ctx = getAudioCtx();
      const notes = [880, 1320]; // A5 + E6 – bright, positive interval
      notes.forEach((freq, i) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.type = "sine";
        osc.frequency.setValueAtTime(freq, ctx.currentTime + i * 0.08);
        gain.gain.setValueAtTime(0, ctx.currentTime + i * 0.08);
        gain.gain.linearRampToValueAtTime(0.18, ctx.currentTime + i * 0.08 + 0.01);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + i * 0.08 + 0.25);
        osc.start(ctx.currentTime + i * 0.08);
        osc.stop(ctx.currentTime + i * 0.08 + 0.28);
      });
    } catch (e) { /* AudioContext blocked before user gesture – silently skip */ }
  }

  /** Play a short low "buzz" error tone */
  function playSoundError() {
    try {
      const ctx = getAudioCtx();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.type = "square";
      osc.frequency.setValueAtTime(220, ctx.currentTime);        // A3
      osc.frequency.setValueAtTime(180, ctx.currentTime + 0.08); // low glide
      gain.gain.setValueAtTime(0.15, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.32);
      osc.start(ctx.currentTime);
      osc.stop(ctx.currentTime + 0.35);
    } catch (e) { /* silently skip */ }
  }

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

  // ── Offline Queue (localStorage) ─────────────────────────────
  const QUEUE_KEY = "bibliothek_offline_queue";

  function loadQueue() {
    try {
      return JSON.parse(localStorage.getItem(QUEUE_KEY) || "[]");
    } catch { return []; }
  }

  function saveQueue(/** @type {any[]} */ q) {
    localStorage.setItem(QUEUE_KEY, JSON.stringify(q));
    offlineQueueCount = q.length;
  }

  /** Enqueue a failed barcode scan for later retry */
  function enqueueOffline(/** @type {string} */ barcode, /** @type {string|null} */ studentID, /** @type {string|null} */ teacherID) {
    const q = loadQueue();
    // Deduplicate: skip if the exact same barcode+student is already queued
    const alreadyQueued = q.some(
      (/** @type {any} */ item) => item.barcode === barcode && item.studentID === studentID && item.teacherID === teacherID
    );
    if (!alreadyQueued) {
      q.push({ barcode, studentID, teacherID, queuedAt: Date.now() });
      saveQueue(q);
    }
    offlineQueueCount = q.length;
  }

  /** Drain the offline queue: replay all pending scans against the API */
  async function flushOfflineQueue() {
    const q = loadQueue();
    if (q.length === 0) return;

    showToast(`📡 Verbindung wiederhergestellt – ${q.length} Offline-Scan(s) werden nachgesendet…`, "success");

    const remaining = [];
    for (const item of q) {
      try {
        const res = await apiFetch("/api/action", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            query: item.barcode,
            active_student_id: item.studentID ?? undefined,
            active_teacher_id: item.teacherID ?? undefined
          }),
          signal: AbortSignal.timeout(8000)
        });
        if (!res.ok) {
          // Business error (e.g., book not found) – drop permanently, don't retry
          console.warn("[OfflineQueue] Permanent error for", item.barcode, res.status);
        }
        // success – item dropped (not pushed to remaining)
      } catch {
        // Network still unavailable – keep in queue
        remaining.push(item);
      }
    }

    saveQueue(remaining);
    if (remaining.length === 0) {
      showToast(`✅ Alle Offline-Scans erfolgreich nachgesendet.`, "success");
      playSoundSuccess();
    } else {
      showToast(`⚠️ ${remaining.length} Scan(s) konnten noch nicht übertragen werden.`, "warning");
    }
  }

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
    const handleOnline = () => {
      isOffline = false;
      flushOfflineQueue();
    };
    const handleOffline = () => { isOffline = true; };
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    // Initialise queue count badge
    offlineQueueCount = loadQueue().length;

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
        scannedBooks = [];
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

  /** @param {KeyboardEvent} e */
  function handleKeydownInput(e) {
    if (!isDropdownOpen || totalDropdownItems === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      selectedDropdownIndex = (selectedDropdownIndex + 1) % totalDropdownItems;
      scrollDropdownToSelected();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      selectedDropdownIndex = selectedDropdownIndex <= 0 ? totalDropdownItems - 1 : selectedDropdownIndex - 1;
      scrollDropdownToSelected();
    } else if (e.key === "Enter" && selectedDropdownIndex >= 0) {
      e.preventDefault();
      selectDropdownItem(selectedDropdownIndex);
    } else if (e.key === "Escape") {
      isDropdownOpen = false;
    }
  }

  function scrollDropdownToSelected() {
    setTimeout(() => {
      const el = document.getElementById(`dropdown-item-${selectedDropdownIndex}`);
      if (el) el.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }, 10);
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
      enqueueOffline(q, activeStudent?.id ?? null, activeTeacher?.id ?? null);
      triggerScreenFlash("warning");
      playSoundError();
      showToast(`📴 Offline: Barcode „${q}" in Warteschlange gespeichert.`, "warning");
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
        scannedBooks = [];
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
      } else if (data.type === "teacher") {
        activeTeacher = data.teacher;
        activeStudent = null;
        scannedBooks = [];
        triggerScreenFlash("success");
        playSoundSuccess();
        triggerFlash("green");
        showToast(`📋 Handapparat-Sitzung gestartet für Lehrer/in ${data.teacher.vorname} ${data.teacher.nachname}`);
      } else if (data.type === "ausleihe") {
        scannedBooks = [{ book: data.book, action: "ausleihe", date: new Date(), dueDate: data.due_date, loanId: data.loan_id, schuelerID: data.student?.id ?? activeStudent?.id }, ...scannedBooks];

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
        scannedBooks = [{ book: data.book, action: "rueckgabe", date: new Date(), loanId: data.loan_id, schuelerID: data.student?.id }, ...scannedBooks];
        showToast(`📥 „${data.book.titel}" erfolgreich zurückgegeben.`);
        if (data.has_vormerkung) {
          vormerkungAlert = { titel: data.vormerkung_titel || data.book?.titel };
        }
        studentProfileComponent?.reloadProfile();

        if (data.student && !activeStudent && !activeTeacher) {
          activeStudent = data.student;
          scannedBooks = [];
        } else if (data.teacher && !activeStudent && !activeTeacher) {
          activeTeacher = data.teacher;
          scannedBooks = [];
        }
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      const isTimeout = error?.name === "TimeoutError" || error?.name === "AbortError";

      // WLAN dropout: queue the scan if it was a book barcode
      if (isTimeout && q.startsWith("B-")) {
        enqueueOffline(q, activeStudent?.id ?? null, activeTeacher?.id ?? null);
        triggerScreenFlash("error");
        playSoundError();
        triggerFlash("orange");
        showToast(`📴 Zeitüberschreitung – Barcode „${q}" offline gespeichert (${offlineQueueCount} ausstehend).`, "warning");
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
      scannedBooks = scannedBooks.filter((_, i) => i !== entryIndex);
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
      scannedBooks = scannedBooks.map((e, i) => i === entryIndex ? { ...e, defekt: true } : e);
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
{#if isOffline || offlineQueueCount > 0}
  <div class="fixed top-4 right-4 z-200 flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold shadow-lg border
    {isOffline ? 'bg-rose-50 border-rose-200 text-rose-700' : 'bg-amber-50 border-amber-200 text-amber-700'} animate-slide-down">
    {#if isOffline}
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636a9 9 0 010 12.728M5.636 5.636a9 9 0 000 12.728M12 12h.01"/>
      </svg>
      <span>Offline{offlineQueueCount > 0 ? ` · ${offlineQueueCount} Scan(s) ausstehend` : ""}</span>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      <span>{offlineQueueCount} Offline-Scan(s) ausstehend</span>
      <button onclick={flushOfflineQueue} class="ml-1 underline hover:no-underline cursor-pointer">Jetzt senden</button>
    {/if}
  </div>
{/if}

<div class="w-full transition-all duration-500 ease-in-out {isActive ? 'max-w-4xl pt-4 justify-start' : 'max-w-2xl min-h-[60vh] justify-center'} flex flex-col items-center space-y-6">
  <div class="w-full transition-all duration-500 {isActive ? 'sticky -top-4 z-30 bg-slate-50/95 backdrop-blur-md py-4' : ''}">
    <form onsubmit={submitAction} class="w-full relative bg-white py-5 px-8 rounded-3xl border border-slate-200 shadow-2xl no-print transition-all duration-500 focus-within:border-blue-500 focus-within:ring-4 focus-within:ring-blue-50 {isActive ? 'scale-100' : 'scale-105'} {isShaking ? 'animate-shake border-rose-400' : ''} {flashBorder === 'green' ? 'ring-4 ring-emerald-500/10 border-emerald-400' : flashBorder === 'orange' ? 'ring-4 ring-amber-500/10 border-amber-400' : ''}">
      <input 
        id="omnibox-input" 
        type="text" 
        role="combobox"
        autocomplete="off" 
        aria-expanded={isDropdownOpen}
        aria-autocomplete="list"
        aria-controls="omnibox-dropdown"
        bind:value={queryVal} 
        oninput={handleInput} 
        onkeydown={handleKeydownInput} 
        class="w-full pl-10 pr-12 bg-transparent text-slate-800 font-sans text-xl placeholder-slate-400 focus:outline-none tracking-wide" 
        placeholder={activeStudent || activeTeacher ? "Buch-Barcode (B-) scannen..." : "Schüler (S-), Lehrer (L-), Buch (B-) scannen..."} 
      />
      <div class="absolute left-8 top-1/2 -translate-y-1/2 text-slate-400"><svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg></div>
      <!-- Mobile camera button -->
      <button type="button" onclick={showCamera ? stopCamera : startCamera}
        title="Kamera-Scanner (Mobilgerät)"
        aria-label="Kamera-Barcode-Scanner ein- oder ausschalten"
        class="absolute right-5 top-1/2 -translate-y-1/2 p-1.5 rounded-xl transition-colors {showCamera ? 'bg-blue-100 text-blue-600' : 'text-slate-400 hover:text-blue-500 hover:bg-blue-50'}">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"/>
        </svg>
      </button>

      {#if isDropdownOpen && totalDropdownItems > 0}
        <div id="omnibox-dropdown" role="listbox" aria-label="Suchergebnisse" class="absolute top-full left-0 right-0 mt-4 bg-white/80 backdrop-blur-2xl border border-white/60 shadow-[0_12px_40px_rgb(0,0,0,0.12)] rounded-2xl z-50 overflow-hidden flex flex-col max-h-[60vh] animate-slide-up">
          <div class="overflow-y-auto overscroll-contain flex-1 p-3 space-y-4">
            {#if unifiedSearchResults.students.length > 0}
              <div>
                <div class="px-3 pb-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider">Schüler ({unifiedSearchResults.students.length})</div>
                <div class="space-y-1">
                  {#each unifiedSearchResults.students as student, i}
                    {@render studentDropdownRow(student, i)}
                  {/each}
                </div>
              </div>
            {/if}
            {#if unifiedSearchResults.books.length > 0}
              <div>
                <div class="px-3 pb-2 text-[10px] font-bold text-slate-400 uppercase tracking-wider">Bücher ({unifiedSearchResults.books.length})</div>
                <div class="space-y-1">
                  {#each unifiedSearchResults.books as book, j}
                    {@render bookDropdownRow(book, j + unifiedSearchResults.students.length)}
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </form>
  </div>

  <!-- HTML5 Kamera-Scanner (Mobile) -->
  {#if showCamera}
    <div class="w-full rounded-2xl overflow-hidden border border-blue-200 shadow-lg animate-slide-up bg-black relative">
      <div class="absolute top-3 right-3 z-10">
        <button onclick={stopCamera} class="p-1.5 rounded-full bg-white/80 text-slate-700 hover:bg-white shadow transition-colors" title="Kamera schließen">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
        </button>
      </div>
      <div class="px-4 pt-3 pb-1 text-xs text-blue-200 font-semibold text-center">Kamera auf Barcode richten</div>
      <div id="camera-scan-region" class="w-full min-h-[240px]"></div>
    </div>
  {/if}

  {#if activeStudent}
    {#if lastFremdrueckgabe}
      <div class="w-full max-w-xl p-3 rounded-xl bg-amber-50 border border-amber-100 text-amber-800 text-xs font-medium flex items-center space-x-2 animate-slide-up no-print mb-2">
        <span>⚠️</span>
        <span><strong>Fremdrückgabe:</strong> Buch wurde von <strong>{lastFremdrueckgabe.vorbesitzerName}</strong> zurückgegeben und für {activeStudent.vorname} verbucht.</span>
      </div>
    {/if}
    <StudentProfile bind:this={studentProfileComponent} student={activeStudent} onDeselect={() => { activeStudent = null; scannedBooks = []; lastFremdrueckgabe = null; }} />
  {:else if activeTeacher}
    {@render teacherCard(activeTeacher)}
  {/if}

  {#if (activeStudent || activeTeacher) && scannedBooks.length > 0}
    <div class="w-full max-w-xl rounded-2xl border border-slate-100 bg-white overflow-hidden animate-slide-up shadow-sm">
      <div class="px-5 py-3 border-b border-slate-100 text-xs text-slate-400 uppercase tracking-wider font-mono">Scans in dieser Sitzung</div>
      <div class="divide-y divide-slate-100 max-h-60 overflow-y-auto">
        {#each scannedBooks as entry, idx}
          {@render scannedBookRow(entry, idx)}
        {/each}
      </div>
    </div>
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

<!-- ── Snippets ── -->
{#snippet bookCover(/** @type {any} */ book)}
  {#if book.cover_url}
    <img src={book.cover_url} class="w-12 h-16 object-cover rounded-md shadow-sm border border-slate-100" alt="Cover" />
  {:else}
    <div class="w-12 h-16 rounded-md shadow-sm flex-none flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-600 text-sm border border-indigo-600/10">
      {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
    </div>
  {/if}
{/snippet}

{#snippet teacherCard(/** @type {any} */ teacher)}
  <div class="w-full max-w-xl p-5 rounded-2xl bg-blue-50 border border-blue-100 flex items-center justify-between shadow-sm animate-slide-up">
    <div class="flex items-center space-x-4">
      <div class="w-12 h-12 rounded-xl bg-blue-100/50 border border-blue-200/50 flex items-center justify-center text-blue-600"><svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" /></svg></div>
      <div>
        <h3 class="font-bold text-blue-800">{teacher.vorname} {teacher.nachname}</h3>
        <p class="text-xs text-blue-600/80 font-medium">Handapparat-Modus aktiv · <span class="underline font-semibold">Ausleihe erfolgt als dauerhafter Handapparat</span></p>
      </div>
    </div>
    <div class="flex items-center space-x-3">
      <span class="text-xs px-2.5 py-1 rounded-full bg-blue-100/80 border border-blue-200 text-blue-700 font-semibold tracking-wide uppercase">Handapparat</span>
      <button onclick={() => { activeTeacher = null; scannedBooks = []; lastFremdrueckgabe = null; }} class="p-1 text-blue-500 hover:text-blue-700 transition-colors cursor-pointer" title="Lehrer abwählen (ESC)"><svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg></button>
    </div>
  </div>
{/snippet}

{#snippet scannedBookRow(/** @type {any} */ entry, /** @type {number} */ idx)}
  <div class="p-4 flex items-center justify-between hover:bg-slate-50 transition-colors duration-200 {entry.defekt ? 'bg-rose-50/60' : ''}">
    <div class="flex items-center space-x-4">
      {@render bookCover(entry.book)}
      <div>
        <div class="flex items-center space-x-2 mb-1">
          <span class="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full font-bold border {entry.action === 'ausleihe' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : entry.defekt ? 'bg-rose-100 border-rose-200 text-rose-700' : 'bg-blue-50 border-blue-100 text-blue-700'}">
            {entry.defekt ? 'Defekt' : entry.action === 'ausleihe' ? 'Ausleihe' : 'Rückgabe'}
          </span>
        </div>
        <h4 class="font-semibold text-sm text-slate-800">{entry.book.titel}</h4>
        <p class="text-xs text-slate-400">{entry.book.autor} · Barcode: {entry.book.barcode_id}</p>
      </div>
    </div>
    <div class="flex items-center space-x-2">
      {#if entry.dueDate}
        <div class="text-right mr-2">
          <span class="text-[10px] text-slate-400">Frist:</span>
          <p class="text-xs font-mono text-emerald-600 font-bold">
            {activeTeacher ? 'Dauerhaft (Handapparat)' : new Date(entry.dueDate).toLocaleDateString("de-DE")}
          </p>
        </div>
      {/if}
      {#if entry.action === 'rueckgabe' && entry.loanId && !entry.defekt}
        <button onclick={() => undoReturn(entry.loanId, idx)} title="Rückgabe rückgängig machen"
          class="px-2 py-1 text-xs font-semibold rounded-lg bg-amber-50 border border-amber-200 text-amber-700 hover:bg-amber-100 transition-colors cursor-pointer flex items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/></svg>
          Undo
        </button>
        <button onclick={() => markDefekt(entry, idx)} title="Defekt/Schaden melden und Mahngebühr erheben"
          class="px-2 py-1 text-xs font-semibold rounded-lg bg-rose-50 border border-rose-200 text-rose-700 hover:bg-rose-100 transition-colors cursor-pointer flex items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
          Defekt
        </button>
      {/if}
    </div>
  </div>
{/snippet}

{#snippet studentDropdownRow(/** @type {any} */ student, /** @type {number} */ index)}
  <div id="dropdown-item-{index}"
       role="option"
       aria-selected={selectedDropdownIndex === index}
       aria-label="Schüler: {student.vorname} {student.nachname}, Klasse {student.klasse}, Barcode {student.barcode_id}"
       tabindex="-1"
       class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selectedDropdownIndex === index ? 'bg-blue-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
       onclick={() => selectDropdownItem(index)}
       onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectDropdownItem(index); } }}>
    <div class="flex items-center space-x-3">
      <div class="w-10 h-10 rounded-full flex items-center justify-center font-bold {selectedDropdownIndex === index ? 'bg-white/20 text-white' : 'bg-blue-100 text-blue-700'}" aria-hidden="true">
        {student.vorname.charAt(0)}{student.nachname.charAt(0)}
      </div>
      <div>
        <div class="font-bold {selectedDropdownIndex === index ? 'text-white' : 'text-slate-900'}">{student.vorname} {student.nachname}</div>
        <div class="text-xs {selectedDropdownIndex === index ? 'text-blue-100' : 'text-slate-500'}">{student.klasse} · {student.barcode_id}</div>
      </div>
    </div>
  </div>
{/snippet}

{#snippet bookDropdownRow(/** @type {any} */ book, /** @type {number} */ index)}
  <div id="dropdown-item-{index}"
       role="option"
       aria-selected={selectedDropdownIndex === index}
       aria-label="Buch: {book.titel} von {book.autor}, ISBN {book.isbn || 'Keine ISBN'}"
       tabindex="-1"
       class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selectedDropdownIndex === index ? 'bg-indigo-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
       onclick={() => selectDropdownItem(index)}
       onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectDropdownItem(index); } }}>
    <div class="flex items-center space-x-4">
      {#if book.cover_url}
        <img src={book.cover_url} class="w-10 h-14 object-cover rounded shadow-sm border {selectedDropdownIndex === index ? 'border-indigo-400' : 'border-slate-200'}" alt="Cover von {book.titel}" />
      {:else}
        <div class="w-10 h-14 rounded shadow-sm flex items-center justify-center font-bold text-xs {selectedDropdownIndex === index ? 'bg-indigo-500 text-white border border-indigo-400' : 'bg-slate-100 text-slate-400 border border-slate-200'}" aria-hidden="true">
          {book.titel ? book.titel.charAt(0).toUpperCase() : '?'}
        </div>
      {/if}
      <div>
        <div class="font-bold line-clamp-1 {selectedDropdownIndex === index ? 'text-white' : 'text-slate-900'}">{book.titel}</div>
        <div class="text-xs line-clamp-1 {selectedDropdownIndex === index ? 'text-indigo-100' : 'text-slate-500'}">{book.autor} · {book.isbn || 'Keine ISBN'}</div>
      </div>
    </div>
  </div>
{/snippet}

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
