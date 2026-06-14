<script>
  import { appState, showToast } from "../inventur/lib/store.svelte.js";
  import { uiStore } from "./stores/uiStore.svelte.js";
  import { apiFetch, apiClient } from "./apiFetch.js";
  import BookBorrowersTab from "./BookBorrowersTab.svelte";
  import BookExemplareTab from "./BookExemplareTab.svelte";
  import BookHistoryTab from "./BookHistoryTab.svelte";
  import BookVormerkungenTab from "./BookVormerkungenTab.svelte";

  /** @type {{ bookId: string | null, onBack: () => void }} */
  let { bookId, onBack } = $props();

  /** @type {any} */
  let book = $state(null);
  /** @type {any[]} */
  let borrowers = $state([]);
  /** @type {any[]} */
  let exemplare = $state([]);
  /** @type {any[]} */
  let history = $state([]);
  /** @type {any[]} */
  let vormerkungen = $state([]);

  let activeTab = $state("ausleiher");
  let isLoading = $state(true);

  /** @type {string[]} */
  let coverCandidates = $state([]);
  let currentCandidateIndex = $state(0);
  let coverSrc = $derived(coverCandidates[currentCandidateIndex] || "");
  let coverFailed = $state(false);

  async function deleteTitle() {
    if (!book) return;
    if (!confirm(`Achtung: Dies löscht diesen Titel und ALLE ${exemplare.length} zugehörigen Exemplare unwiderruflich. Fortfahren?`)) return;
    try {
      const res = await apiFetch(`/api/buecher/titel/${book.id}`, { method: "DELETE", credentials: "include" });
      if (res.ok) {
        showToast("Titel erfolgreich gelöscht", "success");
        onBack();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Löschen des Titels.");
      }
    } catch (e) {
      alert("Netzwerkfehler beim Löschen des Titels.");
    }
  }

  function editTitle() {
    if (!book) return;
    appState.bookToEdit = book;
    appState.requestAdminView = true;
    uiStore.activeTab = "media_catalog";
    appState.activeBookId = null;
  }

  $effect(() => {
    if (!bookId) return;
    loadAll(bookId);
  });

  /** @param {string} id */
  async function loadAll(id) {
    isLoading = true;
    // Use book data from appState if available (avoids extra fetch)
    if (appState.selectedBook && appState.selectedBook.id === id) {
      book = appState.selectedBook;
    } else {
      // Fallback: fetch from /api/books/{id} via inventur API
      try {
        const res = await apiFetch(`/api/books/${id}`, { credentials: "include" });
        if (res.ok) book = await res.json();
      } catch { /* ignore */ }
    }

    // Build cover candidates
    const candidates = [];
    if (book?.coverUrl) candidates.push(book.coverUrl);
    if (book?.isbn) {
      const clean = book.isbn.replace(/[- ]/g, "");
      candidates.push(`https://books.google.com/books/content?id=&vid=ISBN:${clean}&printsec=frontcover&img=1&zoom=1`);
      candidates.push(`https://covers.openlibrary.org/b/isbn/${clean}-L.jpg`);
    }
    coverCandidates = candidates;
    currentCandidateIndex = 0;
    coverFailed = candidates.length === 0;

    // Parallel-fetch all detail data
    const [bRes, eRes, hRes, vRes] = await Promise.allSettled([
      apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: "include" }),
      apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: "include" }),
      apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: "include" }),
      apiFetch(`/api/vormerkungen?titel_id=${id}`, { credentials: "include" }),
    ]);

    borrowers = bRes.status === "fulfilled" && bRes.value.ok ? await bRes.value.json() : [];
    exemplare = eRes.status === "fulfilled" && eRes.value.ok ? await eRes.value.json() : [];
    history   = hRes.status === "fulfilled" && hRes.value.ok ? await hRes.value.json() : [];
    vormerkungen = vRes.status === "fulfilled" && vRes.value.ok ? await vRes.value.json() : [];
    isLoading = false;
  }

  function onCoverError() {
    if (currentCandidateIndex < coverCandidates.length - 1) {
      currentCandidateIndex++;
    } else {
      coverFailed = true;
    }
  }

  /** @param {Event} event */
  function onCoverLoad(event) {
    const image = /** @type {HTMLImageElement} */ (event.currentTarget);
    if (image.naturalWidth < 10 || image.naturalHeight < 10) onCoverError();
  }

  const subjectGradients = {
    math: "from-blue-600 via-indigo-600 to-blue-700",
    deu: "from-red-600 via-rose-600 to-red-700",
    eng: "from-violet-600 via-purple-600 to-violet-700",
    bio: "from-teal-600 via-emerald-600 to-teal-700",
    ges: "from-amber-600 via-orange-600 to-amber-700",
    mus: "from-pink-600 via-fuchsia-600 to-pink-700",
    inf: "from-slate-600 via-slate-700 to-slate-800",
  };

  /** @param {string} subject */
  function getGradient(subject) {
    const s = (subject || "").toLowerCase();
    if (s.includes("math")) return subjectGradients.math;
    if (s.includes("eng") || s.includes("fra") || s.includes("spa") || s.includes("lat")) return subjectGradients.eng;
    if (s.includes("bio") || s.includes("che") || s.includes("phy")) return subjectGradients.bio;
    if (s.includes("ges") || s.includes("pol") || s.includes("geo")) return subjectGradients.ges;
    if (s.includes("mus") || s.includes("kun")) return subjectGradients.mus;
    if (s.includes("inf")) return subjectGradients.inf;
    return "from-slate-500 via-slate-600 to-slate-700";
  }
</script>

<div class="w-full max-w-6xl mx-auto space-y-6 animate-fade-in">
  <!-- Back Button + Breadcrumb -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <button
        onclick={onBack}
        class="flex items-center gap-2 px-3 py-2 rounded-xl text-slate-500 hover:text-slate-800 hover:bg-slate-100 transition-all text-sm font-semibold cursor-pointer"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        Zurück zum Katalog
      </button>
      <span class="text-slate-300">/</span>
      <span class="text-slate-500 text-sm truncate max-w-xs">{book?.title ?? "Lade..."}</span>
    </div>
    {#if !isLoading && book}
      <div class="flex items-center gap-2">
        <button
          onclick={editTitle}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-blue-600 hover:text-blue-700 hover:bg-blue-50 transition-all text-xs font-semibold cursor-pointer border border-blue-200"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
          Titel bearbeiten
        </button>
        <button
          onclick={deleteTitle}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-rose-600 hover:text-rose-700 hover:bg-rose-50 transition-all text-xs font-semibold cursor-pointer border border-rose-200"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
          Gesamten Titel löschen
        </button>
      </div>
    {/if}
  </div>

  {#if isLoading}
    <div class="flex justify-center items-center py-32">
      <div class="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if book}
    <!-- Header Card -->
    <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div class="flex flex-col sm:flex-row gap-0">
        <!-- Cover / Spine -->
        <div class="w-full sm:w-48 shrink-0 bg-linear-to-br {getGradient(book.subject)} flex items-center justify-center min-h-56 relative">
          {#if coverSrc && !coverFailed}
            <img
              src={coverSrc}
              alt={`Cover ${book.title}`}
              class="h-full w-full object-cover absolute inset-0"
              onerror={onCoverError}
              onload={onCoverLoad}
            />
          {:else}
            <div class="text-center p-6 z-10">
              <p class="text-xs font-extrabold text-white/60 uppercase tracking-widest mb-2">{book.subject}</p>
              <p class="text-sm font-bold text-white leading-snug line-clamp-4">{book.title}</p>
            </div>
          {/if}
        </div>

        <!-- Meta -->
        <div class="flex-1 p-6 sm:p-8 flex flex-col justify-between gap-4">
          <div>
            <div class="flex flex-wrap gap-2 mb-3">
              <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-blue-50 border border-blue-200 text-blue-700">{book.subject}</span>
              <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-slate-100 border border-slate-200 text-slate-600">Klasse {book.gradeLevel}</span>
              {#if book.jahrgangVon && book.jahrgangBis}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-slate-100 border border-slate-200 text-slate-600">Verwendbar: Kl. {book.jahrgangVon} - {book.jahrgangBis}</span>
              {/if}
              {#if book.track}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-cyan-50 border border-cyan-200 text-cyan-700">{book.track}</span>
              {/if}
              {#if book.medientyp && book.medientyp !== "Buch"}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-amber-50 border border-amber-200 text-amber-700">{book.medientyp}</span>
              {/if}
              {#if book.erweiterte_eigenschaften?.signatur}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-purple-50 border border-purple-200 text-purple-700">📚 {book.erweiterte_eigenschaften.signatur}</span>
              {/if}
              {#if book.erweiterte_eigenschaften?.standort}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-emerald-50 border border-emerald-200 text-emerald-700">📍 {book.erweiterte_eigenschaften.standort}</span>
              {/if}
            </div>
            <h1 class="text-2xl font-extrabold text-slate-900 leading-tight mb-1">{book.title}</h1>
            <p class="text-sm text-slate-500 font-medium">{book.author || "Unbekannter Autor"}</p>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <!-- Stock -->
            <div class="bg-slate-50 rounded-xl p-3 border border-slate-100">
              <p class="text-[10px] font-bold uppercase tracking-wider text-slate-400 mb-1">Verfügbar</p>
              <p class="text-2xl font-extrabold {(book.verfuegbar) === 0 ? 'text-rose-600' : (book.verfuegbar) < 5 ? 'text-amber-600' : 'text-emerald-600'}">
                {book.verfuegbar}
                <span class="text-sm font-medium text-slate-400">/ {book.gesamt}</span>
              </p>
            </div>
            <!-- Ausleiher -->
            <div class="bg-indigo-50 rounded-xl p-3 border border-indigo-100">
              <p class="text-[10px] font-bold uppercase tracking-wider text-indigo-400 mb-1">Ausleiher</p>
              <p class="text-2xl font-extrabold text-indigo-700">{borrowers.length}</p>
            </div>
            <!-- Exemplare -->
            <div class="bg-emerald-50 rounded-xl p-3 border border-emerald-100">
              <p class="text-[10px] font-bold uppercase tracking-wider text-emerald-400 mb-1">Exemplare</p>
              <p class="text-2xl font-extrabold text-emerald-700">{exemplare.length}</p>
            </div>
            <!-- ISBN -->
            <div class="bg-slate-50 rounded-xl p-3 border border-slate-100">
              <p class="text-[10px] font-bold uppercase tracking-wider text-slate-400 mb-1">ISBN / EAN</p>
              <p class="text-sm font-mono font-semibold text-slate-700 break-all">{book.isbn || "–"}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-slate-200">
      <nav class="flex gap-6 overflow-x-auto no-scrollbar" aria-label="Buch-Akte Tabs">
        {#each [
          ["ausleiher", `Ausleiher (${borrowers.length})`], 
          ["exemplare", `Exemplare (${exemplare.length})`], 
          ["vormerkungen", `Vormerkungen (${vormerkungen.length})`],
          ["historie", "Historie"]
        ] as [id, label]}
          <button
            onclick={() => activeTab = id}
            class="relative pb-3 text-sm font-semibold transition-colors cursor-pointer {activeTab === id ? 'text-blue-600' : 'text-slate-500 hover:text-slate-700'}"
            role="tab"
            aria-selected={activeTab === id}
          >
            {label}
            {#if activeTab === id}
              <span class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 rounded-full"></span>
            {/if}
          </button>
        {/each}
      </nav>
    </div>

    <!-- Tab Content -->
    <div class="w-full">
      {#if activeTab === "ausleiher"}
        <BookBorrowersTab {borrowers} {book} {onBack} />
      {:else if activeTab === "exemplare"}
        <BookExemplareTab bind:exemplare {book} {loadAll} />
      {:else if activeTab === "historie"}
        <BookHistoryTab {history} />
      {:else if activeTab === "vormerkungen"}
        <BookVormerkungenTab bind:vormerkungen {book} />
      {/if}
    </div>
  {:else}
    <div class="py-24 flex flex-col items-center text-slate-400 gap-3">
      <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <p class="font-semibold">Buch nicht gefunden.</p>
      <button onclick={onBack} class="text-blue-600 text-sm font-semibold hover:underline cursor-pointer">Zurück zum Katalog</button>
    </div>
  {/if}
</div>
