<script>
  import { appState } from "../inventur/lib/store.svelte.js";

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

  let activeTab = $state("ausleiher");
  let isLoading = $state(true);
  let filterKlasse = $state("Alle");
  let filterName = $state("");

  let availableKlassen = $derived(
    ["Alle", ...Array.from(new Set(borrowers.map((b) => b.klasse || "Unbekannt"))).sort()]
  );

  let filteredBorrowers = $derived(
    borrowers.filter((b) => {
      const matchKlasse = filterKlasse === "Alle" || (b.klasse || "Unbekannt") === filterKlasse;
      const matchName =
        filterName === "" ||
        `${b.schueler_name} ${b.schueler_nachname}`.toLowerCase().includes(filterName.toLowerCase());
      return matchKlasse && matchName;
    })
  );

  /** @type {string[]} */
  let coverCandidates = $state([]);
  let currentCandidateIndex = $state(0);
  let coverSrc = $derived(coverCandidates[currentCandidateIndex] || "");
  let coverFailed = $state(false);

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
        const res = await fetch(`/api/books/${id}`, { credentials: "include" });
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
    const [bRes, eRes, hRes] = await Promise.allSettled([
      fetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: "include" }),
      fetch(`/api/buecher/titel/${id}/exemplare`, { credentials: "include" }),
      fetch(`/api/buecher/titel/${id}/historie`, { credentials: "include" }),
    ]);

    borrowers = bRes.status === "fulfilled" && bRes.value.ok ? await bRes.value.json() : [];
    exemplare = eRes.status === "fulfilled" && eRes.value.ok ? await eRes.value.json() : [];
    history   = hRes.status === "fulfilled" && hRes.value.ok ? await hRes.value.json() : [];
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
    if (s.includes("deu")) return subjectGradients.deu;
    if (s.includes("eng") || s.includes("fra") || s.includes("spa") || s.includes("lat")) return subjectGradients.eng;
    if (s.includes("bio") || s.includes("che") || s.includes("phy")) return subjectGradients.bio;
    if (s.includes("ges") || s.includes("pol") || s.includes("geo")) return subjectGradients.ges;
    if (s.includes("mus") || s.includes("kun")) return subjectGradients.mus;
    if (s.includes("inf")) return subjectGradients.inf;
    return "from-slate-500 via-slate-600 to-slate-700";
  }

  /** @param {string} d */
  function fmtDate(d) {
    if (!d) return "-";
    try { return new Date(d).toLocaleDateString("de-DE"); } catch { return d; }
  }
</script>

<div class="w-full max-w-6xl mx-auto space-y-6 animate-fade-in">
  <!-- Back Button + Breadcrumb -->
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

  {#if isLoading}
    <div class="flex justify-center items-center py-32">
      <div class="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if book}
    <!-- Header Card -->
    <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div class="flex flex-col sm:flex-row gap-0">
        <!-- Cover / Spine -->
        <div class="w-full sm:w-48 shrink-0 bg-linear-to-br {getGradient(book.subject)} flex items-center justify-center min-h-[14rem] relative">
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
              {#if book.track}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-cyan-50 border border-cyan-200 text-cyan-700">{book.track}</span>
              {/if}
              {#if book.medientyp && book.medientyp !== "Buch"}
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-amber-50 border border-amber-200 text-amber-700">{book.medientyp}</span>
              {/if}
            </div>
            <h1 class="text-2xl font-extrabold text-slate-900 leading-tight mb-1">{book.title}</h1>
            <p class="text-sm text-slate-500 font-medium">{book.author || "Unbekannter Autor"}</p>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <!-- Stock -->
            <div class="bg-slate-50 rounded-xl p-3 border border-slate-100">
              <p class="text-[10px] font-bold uppercase tracking-wider text-slate-400 mb-1">Verfügbar</p>
              <p class="text-2xl font-extrabold {(book.verfuegbar ?? book.stock) === 0 ? 'text-rose-600' : (book.verfuegbar ?? book.stock) < 5 ? 'text-amber-600' : 'text-emerald-600'}">
                {book.verfuegbar ?? book.stock}
                <span class="text-sm font-medium text-slate-400">/ {book.gesamt ?? book.stock}</span>
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
      <nav class="flex gap-6" aria-label="Buch-Akte Tabs">
        {#each [["ausleiher", `Ausleiher (${borrowers.length})`], ["exemplare", `Exemplare (${exemplare.length})`], ["historie", "Historie"]] as [id, label]}
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

      <!-- AUSLEIHER TAB -->
      {#if activeTab === "ausleiher"}
        {#if borrowers.length === 0}
          <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
            <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <p class="font-semibold text-sm">Aktuell niemand hat dieses Buch ausgeliehen.</p>
          </div>
        {:else}
          <!-- Filters -->
          <div class="flex gap-3 mb-4">
            <select bind:value={filterKlasse} class="px-3 py-2 bg-white border border-slate-200 rounded-xl text-sm font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/30 cursor-pointer">
              {#each availableKlassen as k}<option value={k}>{k}</option>{/each}
            </select>
            <div class="relative flex-1 max-w-xs">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
              <input type="text" bind:value={filterName} placeholder="Name filtern..." class="w-full pl-9 pr-3 py-2 bg-white border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/30 placeholder:text-slate-400" />
            </div>
            {#if filteredBorrowers.length !== borrowers.length}
              <span class="text-xs text-slate-400 self-center">{filteredBorrowers.length} von {borrowers.length}</span>
            {/if}
          </div>

          <!-- List -->
          <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <ul class="divide-y divide-slate-50">
              {#each filteredBorrowers as b}
                <li class="px-5 py-3.5 hover:bg-slate-50 transition-colors flex items-center justify-between group">
                  <div class="flex items-center gap-3 min-w-0">
                    <div class="w-9 h-9 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold text-xs shrink-0">
                      {b.schueler_name?.[0] ?? ""}{b.schueler_nachname?.[0] ?? ""}
                    </div>
                    <div class="min-w-0">
                      <button
                        onclick={() => { appState.triggerStudentScan = b.schueler_barcode; onBack(); }}
                        class="text-sm font-semibold text-slate-800 hover:text-indigo-600 text-left cursor-pointer truncate block"
                      >
                        {b.schueler_name} {b.schueler_nachname}
                        <span class="text-xs font-normal text-slate-400 ml-1">({b.klasse || "Unbekannt"})</span>
                      </button>
                      <p class="text-xs text-slate-400 font-mono mt-0.5">Exemplar: {b.exemplar_barcode}</p>
                    </div>
                  </div>
                  <div class="text-right shrink-0 ml-4">
                    <p class="text-[10px] font-medium text-slate-400 uppercase tracking-wider">Rückgabe bis</p>
                    <p class="text-sm font-bold {new Date(b.rueckgabe_frist) < new Date() ? 'text-rose-600' : 'text-slate-700'}">
                      {fmtDate(b.rueckgabe_frist)}
                    </p>
                  </div>
                </li>
              {/each}
            </ul>
            {#if filteredBorrowers.length === 0}
              <div class="py-8 text-center text-sm text-slate-400">Keine Ausleihen entsprechen dem Filter.</div>
            {/if}
          </div>
        {/if}

      <!-- EXEMPLARE TAB -->
      {:else if activeTab === "exemplare"}
        {#if exemplare.length === 0}
          <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
            <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            <p class="font-semibold text-sm">Keine physischen Exemplare mit Barcodes angelegt.</p>
          </div>
        {:else}
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {#each exemplare as ex}
              <div class="bg-white rounded-xl border border-slate-200 p-4 shadow-sm hover:border-slate-300 transition-colors">
                <div class="flex items-start justify-between mb-3">
                  <span class="text-xs font-bold text-blue-700 bg-blue-50 border border-blue-100 px-2 py-0.5 rounded font-mono">{ex.barcode_id}</span>
                  <span class="text-[10px] font-bold px-2 py-0.5 rounded-full {!ex.ist_ausleihbar ? 'bg-rose-50 text-rose-700 border border-rose-100' : !ex.ist_verfuegbar ? 'bg-amber-50 text-amber-700 border border-amber-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
                    {!ex.ist_ausleihbar ? "Gesperrt" : !ex.ist_verfuegbar ? "Ausgeliehen" : "Verfügbar"}
                  </span>
                </div>
                {#if ex.zustand_notiz}
                  <p class="text-xs text-slate-500"><span class="font-semibold text-slate-400">Zustand:</span> {ex.zustand_notiz}</p>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

      <!-- HISTORIE TAB -->
      {:else if activeTab === "historie"}
        {#if history.length === 0}
          <div class="py-16 flex flex-col items-center text-slate-400 gap-3">
            <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            <p class="font-semibold text-sm">Noch keine Ausleihen in der Datenbank vorhanden.</p>
          </div>
        {:else}
          <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div class="px-5 py-3 border-b border-slate-100 flex items-center justify-between">
              <p class="text-xs font-bold text-slate-500 uppercase tracking-wider">Letzte {history.length} Ausleihen</p>
            </div>
            <ul class="divide-y divide-slate-50">
              {#each history as h}
                <li class="px-5 py-3 flex items-center justify-between hover:bg-slate-50 transition-colors">
                  <div class="flex items-center gap-3 min-w-0">
                    <div class="w-8 h-8 rounded-full bg-slate-100 text-slate-500 flex items-center justify-center font-bold text-xs shrink-0">
                      {h.schueler_name?.[0] ?? ""}{h.schueler_nachname?.[0] ?? ""}
                    </div>
                    <div class="min-w-0">
                      <p class="text-sm font-semibold text-slate-800 truncate">{h.schueler_name} {h.schueler_nachname} <span class="text-xs font-normal text-slate-400">({h.klasse})</span></p>
                      <p class="text-xs text-slate-400 font-mono">Exemplar: {h.exemplar_barcode}</p>
                    </div>
                  </div>
                  <div class="text-right shrink-0 ml-4 space-y-0.5">
                    <p class="text-xs text-slate-500">
                      <span class="font-medium text-slate-400">Von</span> {fmtDate(h.ausgeliehen_am)}
                    </p>
                    <p class="text-xs {h.rueckgabe_am ? 'text-emerald-600' : 'text-amber-600'} font-semibold">
                      {h.rueckgabe_am ? `Zurück ${fmtDate(h.rueckgabe_am)}` : "Noch ausgeliehen"}
                    </p>
                  </div>
                </li>
              {/each}
            </ul>
          </div>
        {/if}
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
