<script>
  import { onMount } from "svelte";
  import StudentProfile from "./StudentProfile.svelte";

  let { onSelectBook } = $props();

  let activeStudent = $state(/** @type {any} */ (null));
  let activeTeacher = $state(/** @type {any} */ (null));
  let queryVal = $state("");
  let searchResults = $state(/** @type {any[]} */ ([]));
  let scannedBooks = $state(/** @type {any[]} */ ([])); 
  let toast = $state(/** @type {any} */ (null)); 
  let flashBorder = $state(""); 
  let lastFremdrueckgabe = $state(/** @type {any} */ (null)); 
  let studentProfileComponent = $state(/** @type {any} */ (null));
  let isShaking = $state(false);

  let isActive = $derived(!!(activeStudent || activeTeacher || searchResults.length > 0));

  function triggerShake() {
    isShaking = true;
    setTimeout(() => { isShaking = false; }, 500);
  }

  onMount(() => {
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
    return () => source.close();
  });

  // Automatically focus input only when the omnibox is in centered idle state
  $effect(() => {
    if (!isActive) {
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
        searchResults = [];
        scannedBooks = [];
        lastFremdrueckgabe = null;
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
  
  /** @param {string} color */
  function triggerFlash(color) { flashBorder = color; setTimeout(() => { flashBorder = ""; }, 1000); }

  /** @param {Event} [e] */
  async function submitAction(e) {
    if (e) e.preventDefault();
    const q = queryVal.trim();
    if (!q) return;

    queryVal = "";
    searchResults = [];
    lastFremdrueckgabe = null;

    try {
      const res = await fetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query: q,
          active_student_id: activeStudent?.id,
          active_teacher_id: activeTeacher?.id
        })
      });

      if (!res.ok) throw new Error(await res.text() || "Aktion failed");
      const data = await res.json();
      
      if (data.type === "student") {
        activeStudent = data.student;
        activeTeacher = null;
        scannedBooks = [];
        triggerFlash("green");
      } else if (data.type === "teacher") {
        activeTeacher = data.teacher;
        activeStudent = null;
        scannedBooks = [];
        triggerFlash("green");
        showToast(`📋 Handapparat-Sitzung gestartet für Lehrer/in ${data.teacher.vorname} ${data.teacher.nachname}`);
      } else if (data.type === "ausleihe") {
        triggerFlash("green");
        scannedBooks = [{ book: data.book, action: "ausleihe", date: new Date(), dueDate: data.due_date }, ...scannedBooks];

        if (data.fremdrueckgabe) {
          triggerFlash("orange");
          const prevName = data.vorbesitzer ? `${data.vorbesitzer.vorname} ${data.vorbesitzer.nachname}` : `${data.vorbesitzer_user.vorname} ${data.vorbesitzer_user.nachname}`;
          lastFremdrueckgabe = { vorbesitzerName: prevName };
          showToast(`⚠️ Fremdrückgabe erfolgt (Vorbesitzer: ${prevName})`, "warning");
        } else {
          showToast(`📖 "${data.book.titel}" ausgeliehen an ${activeTeacher ? activeTeacher.vorname : activeStudent.vorname}.`);
        }
        studentProfileComponent?.reloadProfile();
      } else if (data.type === "rueckgabe") {
        triggerFlash("green");
        scannedBooks = [{ book: data.book, action: "rueckgabe", date: new Date() }, ...scannedBooks];
        showToast(`📥 "${data.book.titel}" erfolgreich zurückgegeben.`);
        studentProfileComponent?.reloadProfile();

        if (data.student && !activeStudent && !activeTeacher) {
          activeStudent = data.student;
          scannedBooks = [];
        } else if (data.teacher && !activeStudent && !activeTeacher) {
          activeTeacher = data.teacher;
          scannedBooks = [];
        }
      } else if (data.type === "search_results") {
        searchResults = data.search_results || [];
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      if (q.startsWith("B-") && !activeStudent && !activeTeacher) {
        triggerShake();
        showToast("Bitte zuerst Schüler scannen", "warning");
      } else {
        showToast(error.message || String(error), "error");
        triggerFlash("orange");
      }
    }
  }
</script>

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

{#snippet scannedBookRow(/** @type {any} */ entry)}
  <div class="p-4 flex items-center justify-between hover:bg-slate-50 transition-colors duration-200">
    <div class="flex items-center space-x-4">
      {@render bookCover(entry.book)}
      <div>
        <div class="flex items-center space-x-2 mb-1">
          <span class="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full font-bold border {entry.action === 'ausleihe' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : 'bg-blue-50 border-blue-100 text-blue-700'}">
            {entry.action === 'ausleihe' ? 'Ausleihe' : 'Rückgabe'}
          </span>
        </div>
        <h4 class="font-semibold text-sm text-slate-800">{entry.book.titel}</h4>
        <p class="text-xs text-slate-400">{entry.book.autor} · Barcode: {entry.book.barcode_id}</p>
      </div>
    </div>
    {#if entry.dueDate}
      <div class="text-right">
        <span class="text-[10px] text-slate-400">Frist:</span>
        <p class="text-xs font-mono text-emerald-600 font-bold">
          {activeTeacher ? 'Dauerhaft (Handapparat)' : new Date(entry.dueDate).toLocaleDateString("de-DE")}
        </p>
      </div>
    {/if}
  </div>
{/snippet}

{#snippet searchResultRow(/** @type {any} */ book)}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div onclick={() => { if (onSelectBook) onSelectBook(book); }} class="p-4 flex items-center justify-between hover:bg-slate-50 transition-colors cursor-pointer">
    <div class="flex items-center space-x-4">
      {@render bookCover(book)}
      <div>
        <h4 class="font-semibold text-sm text-slate-800">{book.titel}</h4>
        <p class="text-xs text-slate-400">{book.autor} · {book.verlag} ({book.erscheinungsjahr})</p>
      </div>
    </div>
    <span class="text-[10px] font-mono bg-slate-50 border border-slate-200 px-2 py-0.5 rounded text-slate-500">{book.isbn || 'Keine ISBN'}</span>
  </div>
{/snippet}

<div class="w-full transition-all duration-500 ease-in-out {isActive ? 'max-w-4xl pt-4 justify-start' : 'max-w-2xl min-h-[60vh] justify-center'} flex flex-col items-center space-y-6">
  <form onsubmit={submitAction} class="w-full relative bg-white py-5 px-8 rounded-3xl border border-slate-200 shadow-2xl no-print transition-all duration-500 focus-within:border-blue-500 focus-within:ring-4 focus-within:ring-blue-50 {isActive ? 'scale-100' : 'scale-105'} {isShaking ? 'animate-shake border-rose-400' : ''} {flashBorder === 'green' ? 'ring-4 ring-emerald-500/10 border-emerald-400' : flashBorder === 'orange' ? 'ring-4 ring-amber-500/10 border-amber-400' : ''}">
    <input id="omnibox-input" type="text" autocomplete="off" bind:value={queryVal} class="w-full pl-10 bg-transparent text-slate-800 font-sans text-xl placeholder-slate-400 focus:outline-none tracking-wide" placeholder={activeStudent || activeTeacher ? "Buch-Barcode (B-) scannen..." : "Schüler (S-), Lehrer (L-), Buch (B-) scannen..."} />
    <div class="absolute left-8 top-1/2 -translate-y-1/2 text-slate-400"><svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg></div>
  </form>

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
        {#each scannedBooks as entry}
          {@render scannedBookRow(entry)}
        {/each}
      </div>
    </div>
  {/if}

  {#if searchResults.length > 0}
    <div class="w-full rounded-2xl border border-slate-100 bg-white overflow-hidden animate-fade-in shadow-xl">
      <div class="px-5 py-3 border-b border-slate-100 text-xs text-slate-400 uppercase tracking-wider font-mono">Bücherkatalog Treffer</div>
      <div class="divide-y divide-slate-100 max-h-60 overflow-y-auto">
        {#each searchResults as book}
          {@render searchResultRow(book)}
        {/each}
      </div>
    </div>
  {/if}
</div>

<div class="fixed top-24 left-1/2 -translate-x-1/2 w-full max-w-lg z-40 space-y-3 px-4 pointer-events-none">
  {#if toast}
    <div class="p-4 rounded-xl shadow-xl flex items-center space-x-3 backdrop-blur-md animate-slide-down pointer-events-auto border {toast.type === 'success' ? 'bg-emerald-50 border-emerald-100/50 text-emerald-700' : 'bg-rose-50 border-rose-100/50 text-rose-700'}">
      <span class="text-sm font-semibold">{toast.message}</span>
    </div>
  {/if}
</div>

<style>
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
