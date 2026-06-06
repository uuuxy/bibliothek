<script>
  import { appState, showToast } from "../inventur/lib/store.svelte.js";
  import { apiFetch } from "./apiFetch.js";

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

  /** @type {string | null} */
  let editingExemplarId = $state(null);
  let editBarcodeValue = $state("");
  let barcodeError = $state("");

  /** @type {string | null} */
  let editingStatusId = $state(null);
  let editStatusType = $state("Verfügbar");
  let editStatusNote = $state("");
  let statusError = $state("");
  let selectedExemplare = $state(new Set());

  /** @param {any} ex */
  async function saveBarcode(ex) {
    if (!editBarcodeValue.trim()) return;
    if (editBarcodeValue.trim() === ex.barcode_id) {
      editingExemplarId = null;
      return;
    }
    barcodeError = "";
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${ex.id}/barcode`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ barcode: editBarcodeValue.trim() })
      });
      if (res.ok) {
        ex.barcode_id = editBarcodeValue.trim();
        editingExemplarId = null;
      } else {
        const errorData = await res.json().catch(() => ({}));
        barcodeError = errorData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      barcodeError = "Netzwerkfehler";
    }
  }

  /** @param {any} ex */
  function openStatusEdit(ex) {
    editingStatusId = ex.id;
    if (ex.ist_ausleihbar) {
      editStatusType = "Verfügbar";
    } else if (ex.ist_ausgesondert || (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes("verloren"))) {
      editStatusType = "Verloren";
    } else {
      editStatusType = "Gesperrt (Defekt/Reserviert)";
    }
    editStatusNote = ex.zustand_notiz || "";
    statusError = "";
  }

  /** @param {any} ex */
  async function saveStatus(ex) {
    statusError = "";
    try {
      const isAusleihbar = editStatusType === "Verfügbar";
      const isAusgesondert = editStatusType === "Verloren" ? true : false;
      const notiz = isAusleihbar ? "" : editStatusNote.trim();
      const res = await apiFetch(`/api/buecher/exemplare/${ex.id}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          ist_ausleihbar: isAusleihbar,
          ist_ausgesondert: isAusgesondert,
          zustand_notiz: notiz
        })
      });
      if (res.ok) {
        ex.ist_ausleihbar = isAusleihbar;
        ex.ist_ausgesondert = isAusgesondert;
        ex.zustand_notiz = notiz;
        editingStatusId = null;
        showToast("Status erfolgreich gespeichert", "success");
      } else {
        const errData = await res.json().catch(() => ({}));
        statusError = errData.error || "Fehler beim Speichern";
      }
    } catch (e) {
      statusError = "Netzwerkfehler";
    }
  }

  /** @param {any} ex */
  async function deleteCopy(ex) {
    if (!confirm(`Möchtest du das Exemplar ${ex.barcode_id} wirklich unwiderruflich löschen?`)) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${ex.id}`, { method: "DELETE", credentials: "include" });
      if (res.ok) {
        exemplare = exemplare.filter((e) => e.id !== ex.id);
        if (book) {
          book.gesamt = Math.max(0, (book.gesamt || 0) - 1);
          if (book.verfuegbar !== undefined && ex.ist_ausleihbar) {
             book.verfuegbar = Math.max(0, (book.verfuegbar || 0) - 1);
          }
        }
        showToast("Exemplar erfolgreich gelöscht", "success");
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || "Fehler beim Löschen des Exemplars.");
      }
    } catch (e) {
      alert("Netzwerkfehler beim Löschen.");
    }
  }

  async function deleteSelectedCopies() {
    if (selectedExemplare.size === 0) return;
    if (!confirm(`Möchtest du die ${selectedExemplare.size} ausgewählten Exemplare unwiderruflich löschen?`)) return;
    
    let successCount = 0;
    for (const id of selectedExemplare) {
      try {
        const res = await apiFetch(`/api/buecher/exemplare/${id}`, { method: "DELETE", credentials: "include" });
        if (res.ok) {
          exemplare = exemplare.filter((e) => e.id !== id);
          successCount++;
        }
      } catch (e) {
        console.error("Fehler beim Löschen:", e);
      }
    }
    selectedExemplare.clear();
    if (successCount > 0) {
      showToast(`${successCount} Exemplare erfolgreich gelöscht`, "success");
      if (bookId) loadAll(bookId);
    }
  }

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
    const [bRes, eRes, hRes] = await Promise.allSettled([
      apiFetch(`/api/buecher/titel/${id}/ausleiher`, { credentials: "include" }),
      apiFetch(`/api/buecher/titel/${id}/exemplare`, { credentials: "include" }),
      apiFetch(`/api/buecher/titel/${id}/historie`, { credentials: "include" }),
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

  function printAusleiher() {
    const printWindow = window.open('', '_blank', 'width=800,height=600');
    if (!printWindow) {
      alert("Bitte erlaube Popups, um die Liste zu drucken.");
      return;
    }
    
    const printDate = new Date().toLocaleDateString("de-DE");
    let html = `
      <!DOCTYPE html>
      <html>
      <head>
        <title>Mahnliste: ${book?.title || 'Buch'}</title>
        <style>
          body { font-family: system-ui, -apple-system, sans-serif; padding: 2rem; color: #1e293b; }
          h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
          p.meta { margin: 0 0 1.5rem 0; color: #64748b; font-size: 0.875rem; }
          table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
          th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #e2e8f0; }
          th { background: #f8fafc; font-weight: 600; font-size: 0.875rem; color: #475569; }
          .overdue { color: #e11d48; font-weight: bold; }
          @media print { @page { margin: 1cm; } }
        </style>
      </head>
      <body>
        <h1>Ausleiher-Liste: ${book?.title || 'Buch'}</h1>
        <p class="meta">Erstellt am: ${printDate} | Filter: Klasse ${filterKlasse}</p>
        <table>
          <thead>
            <tr>
              <th>Schüler/in</th>
              <th>Klasse</th>
              <th>Exemplar</th>
              <th>Ausgeliehen am</th>
              <th>Rückgabe bis</th>
            </tr>
          </thead>
          <tbody>
    `;

    for (const b of filteredBorrowers) {
      const isOverdue = new Date(b.rueckgabe_frist) < new Date();
      html += `
        <tr>
          <td>${b.schueler_name} ${b.schueler_nachname}</td>
          <td>${b.klasse || '-'}</td>
          <td style="font-family: monospace; font-size: 0.875rem;">${b.exemplar_barcode}</td>
          <td>${fmtDate(b.ausgeliehen_am)}</td>
          <td class="${isOverdue ? 'overdue' : ''}">${fmtDate(b.rueckgabe_frist)}</td>
        </tr>
      `;
    }

    html += `
          </tbody>
        </table>
        \x3Cscript>
          window.onload = () => { setTimeout(() => window.print(), 200); }
        \x3C/script>
      </body>
      </html>
    `;
    
    printWindow.document.open();
    printWindow.document.write(html);
    printWindow.document.close();
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
      <button
        onclick={deleteTitle}
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-rose-600 hover:text-rose-700 hover:bg-rose-50 transition-all text-xs font-semibold cursor-pointer border border-rose-200"
      >
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
        Gesamten Titel löschen
      </button>
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
            <div class="flex-1"></div>
            <button onclick={printAusleiher} class="flex items-center gap-1.5 px-3 py-2 bg-white border border-slate-200 rounded-xl text-sm font-semibold text-slate-600 hover:text-slate-800 hover:bg-slate-50 transition-colors shadow-sm cursor-pointer">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" /></svg>
              Mahnliste drucken
            </button>
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
                  <div class="text-right shrink-0 ml-4 flex gap-6 items-center">
                    <div class="text-right hidden sm:block">
                      <p class="text-[10px] font-medium text-slate-400 uppercase tracking-wider">Ausgeliehen</p>
                      <p class="text-sm font-semibold text-slate-600">
                        {fmtDate(b.ausgeliehen_am)}
                      </p>
                    </div>
                    <div class="text-right">
                      <p class="text-[10px] font-medium text-slate-400 uppercase tracking-wider">Rückgabe bis</p>
                      <p class="text-sm font-bold {new Date(b.rueckgabe_frist) < new Date() ? 'text-rose-600' : 'text-slate-700'}">
                        {fmtDate(b.rueckgabe_frist)}
                      </p>
                    </div>
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
          {#if selectedExemplare.size > 0 && appState.adminAuthenticated}
            <div class="mb-4 p-3 bg-rose-50 border border-rose-100 rounded-xl flex items-center justify-between animate-fade-in">
              <span class="text-sm font-semibold text-rose-800">{selectedExemplare.size} Exemplare ausgewählt</span>
              <button class="px-3 py-1.5 bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold rounded-lg shadow-sm transition-colors cursor-pointer" onclick={deleteSelectedCopies}>
                Ausgewählte löschen
              </button>
            </div>
          {/if}
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {#each exemplare as ex}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="bg-white rounded-xl border p-4 shadow-sm transition-colors cursor-pointer {selectedExemplare.has(ex.id) ? 'border-blue-500 bg-blue-50/30 ring-1 ring-blue-500' : 'border-slate-200 hover:border-slate-300'}"
                   onclick={() => {
                     if (!appState.adminAuthenticated) return;
                     if (selectedExemplare.has(ex.id)) {
                       const newSet = new Set(selectedExemplare);
                       newSet.delete(ex.id);
                       selectedExemplare = newSet;
                     } else {
                       selectedExemplare = new Set(selectedExemplare).add(ex.id);
                     }
                   }}>
                <div class="flex items-start justify-between mb-3">
                  {#if editingExemplarId === ex.id}
                    <div class="flex-1 mr-2 relative">
                      <!-- svelte-ignore a11y_autofocus -->
                      <input
                        type="text"
                        bind:value={editBarcodeValue}
                        autofocus
                        onfocus={(e) => e.currentTarget.select()}
                        class="w-full px-2 py-1 text-xs font-mono border {barcodeError ? 'border-rose-500 bg-rose-50 text-rose-700' : 'border-blue-300'} rounded focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                        onkeydown={(e) => {
                          if (e.key === 'Enter') saveBarcode(ex);
                          if (e.key === 'Escape') { editingExemplarId = null; barcodeError = ""; }
                        }}
                      />
                      {#if barcodeError}
                        <p class="text-[10px] text-rose-600 mt-1 absolute -bottom-4 left-0 truncate w-full" title={barcodeError}>{barcodeError}</p>
                      {/if}
                    </div>
                  {:else}
                    <div class="flex items-center gap-3">
                      {#if appState.adminAuthenticated}
                        <input type="checkbox"
                               checked={selectedExemplare.has(ex.id)}
                               class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 cursor-pointer pointer-events-none"
                        />
                      {/if}
                      <div class="flex items-center gap-2">
                        <span class="text-xs font-bold {ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-') ? 'text-amber-700 bg-amber-50 border-amber-100' : 'text-blue-700 bg-blue-50 border-blue-100'} border px-2 py-0.5 rounded font-mono">
                          {ex.barcode_id}
                        </span>
                        {#if appState.adminAuthenticated}
                          {#if ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-')}
                            <button
                              class="text-xs px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-800 font-semibold rounded shadow-sm transition-colors cursor-pointer flex items-center gap-1"
                              onclick={(e) => { 
                                e.stopPropagation(); 
                                editingExemplarId = ex.id; 
                                editBarcodeValue = ""; // Leer lassen für den Scanner
                                barcodeError = ""; 
                              }}
                            >
                              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
                              Barcode scannen
                            </button>
                          {:else}
                            <button
                              title="Barcode zuweisen/ändern"
                              class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer flex items-center gap-1"
                              onclick={(e) => { 
                                e.stopPropagation(); 
                                editingExemplarId = ex.id; 
                                editBarcodeValue = ex.barcode_id; 
                                barcodeError = ""; 
                              }}
                            >
                              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
                            </button>
                          {/if}
                        {/if}
                      </div>
                    </div>
                  {/if}
                  <div class="flex items-center gap-2">
                    <span class="text-[10px] font-bold px-2 py-0.5 rounded-full {!ex.ist_ausleihbar ? 'bg-rose-50 text-rose-700 border border-rose-100' : !ex.ist_verfuegbar ? 'bg-amber-50 text-amber-700 border border-amber-100' : 'bg-emerald-50 text-emerald-700 border border-emerald-100'}">
                      {!ex.ist_ausleihbar ? "Gesperrt" : !ex.ist_verfuegbar ? "Ausgeliehen" : "Verfügbar"}
                    </span>
                    {#if editingStatusId !== ex.id}
                      <button title="Status ändern" class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer" onclick={() => openStatusEdit(ex)}>
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
                      </button>
                      <button title="Exemplar löschen" class="text-slate-400 hover:text-rose-600 transition-colors cursor-pointer" onclick={(e) => { e.stopPropagation(); deleteCopy(ex); }}>
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                      </button>
                    {/if}
                  </div>
                </div>
                {#if editingStatusId === ex.id}
                  <div class="mt-2 bg-slate-50 p-3 rounded-lg border border-slate-200">
                    <div class="flex items-center gap-2 mb-2">
                      <select bind:value={editStatusType} class="text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30">
                        <option value="Verfügbar">Verfügbar</option>
                        <option value="Gesperrt (Defekt/Reserviert)">Gesperrt (Defekt/Reserviert)</option>
                        <option value="Verloren">Verloren</option>
                      </select>
                    </div>
                    {#if editStatusType !== "Verfügbar"}
                      <div class="mb-2">
                        <input type="text" bind:value={editStatusNote} placeholder="Notiz (optional)" class="w-full text-xs border border-slate-300 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/30" onkeydown={(e) => { if (e.key === 'Enter') saveStatus(ex); if (e.key === 'Escape') { editingStatusId = null; statusError = ''; } }} />
                      </div>
                    {/if}
                    <div class="flex items-center justify-between">
                      <button onclick={() => { editingStatusId = null; statusError = ''; }} class="text-[10px] text-slate-500 hover:text-slate-700 font-semibold cursor-pointer">Abbrechen</button>
                      <button onclick={() => saveStatus(ex)} class="text-[10px] bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded font-semibold cursor-pointer">Speichern</button>
                    </div>
                    {#if statusError}
                      <p class="text-[10px] text-rose-600 mt-1">{statusError}</p>
                    {/if}
                  </div>
                {:else if ex.zustand_notiz}
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
