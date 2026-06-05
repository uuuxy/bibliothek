<script>
  import { apiFetch } from "./apiFetch.js";
  import AntolinBadge from './AntolinBadge.svelte';
  import { appState } from "../inventur/lib/store.svelte.js";

  // Props
  let { title = { id: "1", titel: "LMF-Mathe 9", autor: "Dr. L. Müller", verlag: "Klett", erscheinungsjahr: 2023 } } = $props();

  // State Runes
  /** @type {any[]} */
  let copies = $state.raw([]);
  let loadingCopies = $state(false);
  let showModal = $state(false);
  /** @type {any} */
  let activeCopy = $state(null);
  let newNote = $state("");

  /** @type {any[]} */
  let borrowers = $state.raw([]);
  let loadingBorrowers = $state(false);

  let filterKlasse = $state("Alle");
  let filterName = $state("");

  let availableKlassen = $derived(
    ["Alle", ...Array.from(new Set(borrowers.map((/** @type {any} */ b) => b.klasse || 'Unbekannt'))).sort()]
  );

  let filteredBorrowers = $derived(
    borrowers.filter((/** @type {any} */ b) => {
      const matchKlasse = filterKlasse === "Alle" || (b.klasse || 'Unbekannt') === filterKlasse;
      const matchName = filterName === "" || `${b.schueler_name} ${b.schueler_nachname}`.toLowerCase().includes(filterName.toLowerCase());
      return matchKlasse && matchName;
    })
  );

  async function loadBorrowers() {
    if (!title || !title.id || title.id === "1") {
      borrowers = [];
      return;
    }
    loadingBorrowers = true;
    try {
      const res = await apiFetch(`/api/buecher/titel/${title.id}/ausleiher`);
      if (res.ok) {
        borrowers = await res.json() || [];
      } else {
        borrowers = [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Ausleiher:", err);
      borrowers = [];
    } finally {
      loadingBorrowers = false;
    }
  }

  async function loadCopies() {
    if (!title || !title.id || title.id === "1") {
      copies = [
        { id: "e1", barcode_id: "B-20031", zustand_notiz: "Leichte Kratzer", ist_ausleihbar: true },
        { id: "e2", barcode_id: "B-20032", zustand_notiz: "Eselsohren auf S. 12", ist_ausleihbar: true },
        { id: "e3", barcode_id: "B-20033", zustand_notiz: "Unleserlicher Barcode", ist_ausleihbar: true },
        { id: "e4", barcode_id: "B-20034", zustand_notiz: "Neuwertig", ist_ausleihbar: true }
      ];
      return;
    }

    loadingCopies = true;
    try {
      const res = await apiFetch(`/api/buecher/titel/${title.id}/exemplare`);
      if (res.ok) {
        copies = await res.json();
      } else {
        copies = [];
      }
    } catch (err) {
      console.error("Fehler beim Laden der Exemplare:", err);
      copies = [];
    } finally {
      loadingCopies = false;
    }
  }

  $effect(() => {
    loadCopies();
    loadBorrowers();
  });

  // Print barcode label immediately to native print dialog
  /**
   * @param {string} barcode
   */
  function printLabel(barcode) {
    const printWindow = window.open("", "_blank");
    if (!printWindow) return;

    printWindow.document.title = "Barcode Label - " + barcode;
    const body = printWindow.document.body;
    body.style.margin = "0";
    body.style.display = "flex";
    body.style.flexDirection = "column";
    body.style.alignItems = "center";
    body.style.justifyContent = "center";
    body.style.height = "100vh";
    body.style.fontFamily = "'Courier New', monospace";
    body.style.fontSize = "8px";
    body.style.color = "black";
    body.style.backgroundColor = "white";

    const styleEl = printWindow.document.createElement("style");
    styleEl.textContent = "@page { size: 50mm 30mm; margin: 0; } .barcode { font-size: 16px; font-weight: bold; letter-spacing: 2px; margin-bottom: 2px; } .label { font-weight: bold; }";
    printWindow.document.head.appendChild(styleEl);

    const barcodeDiv = printWindow.document.createElement("div");
    barcodeDiv.className = "barcode";
    barcodeDiv.textContent = "|||||| | |||| | |";
    body.appendChild(barcodeDiv);

    const labelDiv = printWindow.document.createElement("div");
    labelDiv.className = "label";
    labelDiv.textContent = barcode;
    body.appendChild(labelDiv);

    const titleDiv = printWindow.document.createElement("div");
    titleDiv.textContent = title.titel;
    body.appendChild(titleDiv);

    const scriptEl = printWindow.document.createElement("script");
    scriptEl.textContent = "window.print(); setTimeout(function() { window.close(); }, 500);";
    body.appendChild(scriptEl);
  }

  // Update physical condition notes
  async function updateNote() {
    if (!activeCopy) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${activeCopy.id}/schadensnotiz`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ note: newNote })
      });
      if (!res.ok) throw new Error("Fehler beim Speichern der Notiz");
      
      // Update local state
      const idx = copies.findIndex(c => c.id === activeCopy.id);
      if (idx !== -1) {
        copies[idx].zustand_notiz = newNote;
      }
      showModal = false;
      newNote = "";
      activeCopy = null;
    } catch (err) {
      const error = /** @type {any} */ (err);
      alert(error.message);
    }
  }

  // Delete physical copy from circulation
  /**
   * @param {string} copyId
   */
  async function deleteCopy(copyId) {
    if (!confirm("Möchtest du dieses Buchexemplar wirklich unwiderruflich aus dem Bestand löschen?")) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${copyId}`, {
        method: "DELETE"
      });
      if (!res.ok) throw new Error("Fehler beim Löschen");
      
      // Update local state
      copies = copies.filter(c => c.id !== copyId);
    } catch (err) {
      const error = /** @type {any} */ (err);
      alert(error.message);
    }
  }

  // Mark physical copy as decommissioned (ausgesondert)
  /**
   * @param {string} copyId
   */
  async function aussondernCopy(copyId) {
    if (!confirm("Möchtest du dieses Exemplar aussondern (Makulatur)? Es bleibt für Statistiken in der Datenbank, wird aber aus Katalog, Kiosk und Inventur ausgeblendet.")) return;
    try {
      const res = await apiFetch(`/api/buecher/exemplare/${copyId}/aussondern`, {
        method: "POST"
      });
      if (!res.ok) throw new Error("Fehler beim Aussondern");

      // Update local state to reflect decommissioned status
      const idx = copies.findIndex(c => c.id === copyId);
      if (idx !== -1) {
        copies[idx] = { ...copies[idx], ist_ausgesondert: true, ist_ausleihbar: false };
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      alert(error.message);
    }
  }
</script>

{#if !title}
  <div class="py-12 flex flex-col items-center justify-center text-slate-400 space-y-2">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
    <span class="text-xs font-semibold">Kein Buch ausgewählt. Bitte suche einen Buchtitel über die Ausleihe.</span>
  </div>
{:else}
  <div class="w-full space-y-6 text-slate-800">
    
    <!-- Title Info Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-100 pb-5">
      <div class="flex items-center space-x-4">
        {#if title.coverUrl}
          <img src={title.coverUrl} class="w-14 h-20 object-cover rounded-xl shadow-md border border-slate-100/50 shrink-0" alt="Cover" />
        {:else}
          <div class="w-14 h-20 rounded-xl shadow-md shrink-0 flex items-center justify-center font-bold text-white bg-linear-to-br from-indigo-500 to-purple-650 text-lg border border-indigo-600/10">
            {title.titel ? title.titel.charAt(0).toUpperCase() : '?'}
          </div>
        {/if}
        <div>
          <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Lehrmittelfreiheit (LMF) Klassensatz</span>
          <h2 class="text-2xl font-bold text-slate-900 leading-tight">{title.titel}</h2>
          <p class="text-xs text-slate-500">
            {title.medientyp === 'DVD' ? 'Regisseur' : 'Autor'}: {title.autor} · 
            {title.medientyp === 'CD' || title.medientyp === 'DVD' ? 'EAN' : 'ISBN'}: {title.isbn || '-'} · 
            Verlag: {title.verlag} ({title.erscheinungsjahr})
          </p>
          {#if title.isbn && title.medientyp !== 'CD' && title.medientyp !== 'DVD'}
            <AntolinBadge isbn={title.isbn} />
          {/if}
          {#if title.erweiterteEigenschaften?.standort}
            <div class="mt-1.5">
              <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-amber-50 text-amber-800 border border-amber-100">
                📍 Standort: {title.erweiterteEigenschaften.standort}
              </span>
            </div>
          {/if}
        </div>
      </div>
      <div class="text-sm bg-slate-50 border border-slate-100 rounded-2xl py-2 px-4 flex items-center gap-3">
        <span class="text-slate-400">Exemplare:</span>
        <span class="font-bold text-slate-700">{copies.length} im Bestand</span>
      </div>
    </div>

    <!-- Copies Grid -->
    {#if loadingCopies}
      <div class="py-12 flex justify-center items-center">
        <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
      </div>
    {:else if copies.length === 0}
      <div class="py-12 flex flex-col items-center justify-center text-center max-w-md mx-auto space-y-4">
        <div class="w-12 h-12 rounded-full bg-amber-50 border border-amber-250 flex items-center justify-center text-amber-650 shadow-xs">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <p class="text-sm font-semibold text-slate-700 leading-relaxed">
          Titel vorhanden, aber noch keine physischen Exemplare mit Barcodes angelegt. Bitte im Medienkatalog Exemplare hinzufügen.
        </p>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each copies as copy}
          <div class="p-4 rounded-2xl {copy.ist_ausgesondert ? 'bg-slate-100/60 border border-slate-200 opacity-60' : 'bg-slate-50/50 border border-slate-100 hover:border-slate-200'} flex flex-col justify-between transition-all duration-300">
            <div class="flex items-start justify-between">
              <div class="space-y-1">
                <span class="text-xs font-bold {copy.ist_ausgesondert ? 'text-slate-400 bg-slate-100 border-slate-200' : 'text-blue-700 bg-blue-50 border-blue-100/50'} border px-2 py-0.5 rounded">
                  {copy.barcode_id}
                </span>
                {#if copy.ist_ausgesondert}
                  <span class="block text-[10px] font-bold text-slate-400 uppercase tracking-wider pt-0.5">⛔ Ausgesondert</span>
                {/if}
                <p class="text-xs text-slate-650 pt-1.5"><strong class="text-slate-500 font-medium">Zustand:</strong> {copy.zustand_notiz || 'Neuwertig'}</p>
              </div>
              
              <!-- Copy quick Actions -->
              <div class="flex space-x-1">
                {#if !copy.ist_ausgesondert}
                  <button onclick={() => printLabel(copy.barcode_id)} class="p-1.5 text-slate-450 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors cursor-pointer" title="Schnelldruck Barcode-Etikett">
                    🖨️
                  </button>
                  <button onclick={() => { activeCopy = copy; newNote = copy.zustand_notiz; showModal = true; }} class="p-1.5 text-slate-450 hover:text-amber-600 hover:bg-amber-50 rounded-lg transition-colors cursor-pointer" title="Schadensnotiz bearbeiten">
                    ✏️
                  </button>
                  <button onclick={() => aussondernCopy(copy.id)} class="p-1.5 text-slate-450 hover:text-orange-600 hover:bg-orange-50 rounded-lg transition-colors cursor-pointer" title="Exemplar aussondern (Makulatur)">
                    ⛔
                  </button>
                {/if}
                <button onclick={() => deleteCopy(copy.id)} class="p-1.5 text-slate-450 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer" title="Exemplar löschen">
                  🗑️
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Active Borrowers List -->
    {#if borrowers && borrowers.length > 0}
      <div class="mt-8 border-t border-slate-100 pt-6">
        <h3 class="text-sm font-bold text-slate-700 mb-4 flex items-center gap-2 shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>
          Aktuell entliehen von:
        </h3>
        
        <!-- Filter Controls (Sticky) -->
        <div class="sticky -top-6 pt-4 pb-3 z-10 bg-white flex flex-col gap-3 border-b border-slate-100 mb-3 -mx-2 px-2">
          <div class="flex gap-2">
            <select 
              bind:value={filterKlasse} 
              class="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-indigo-500/50 cursor-pointer min-w-[100px]"
            >
              {#each availableKlassen as k}
                <option value={k}>{k}</option>
              {/each}
            </select>
            <div class="relative flex-1">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
              <input 
                type="text" 
                bind:value={filterName} 
                placeholder="Name filtern..." 
                class="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 placeholder:text-slate-400"
              />
            </div>
          </div>
        </div>

        <!-- Scrollable List -->
        <div class="pb-8">
          {#if filteredBorrowers.length === 0}
            <div class="text-center py-6 text-sm text-slate-400 bg-slate-50 rounded-xl">Keine Ausleihen entsprechen dem Filter.</div>
          {:else}
            <div class="bg-white rounded-2xl border border-slate-100 overflow-hidden shadow-xs">
              <ul class="divide-y divide-slate-50">
                {#each filteredBorrowers as b}
                  <li class="p-3 hover:bg-slate-50 transition-colors flex items-center justify-between group">
                    <div class="flex items-center gap-3 min-w-0">
                      <div class="w-8 h-8 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold text-xs shrink-0">
                        {b.schueler_name ? b.schueler_name.charAt(0) : ''}{b.schueler_nachname ? b.schueler_nachname.charAt(0) : ''}
                      </div>
                      <div class="min-w-0">
                        <button onclick={() => appState.triggerStudentScan = b.schueler_barcode} class="text-sm font-semibold text-slate-800 hover:text-indigo-600 text-left transition-colors cursor-pointer block truncate">
                          {b.schueler_name} {b.schueler_nachname} <span class="text-xs font-normal text-slate-500">({b.klasse || 'Unbekannt'})</span>
                        </button>
                        <div class="text-xs text-slate-400 mt-0.5 truncate">Exemplar: <span class="font-mono text-slate-500">{b.exemplar_barcode}</span></div>
                      </div>
                    </div>
                    <div class="text-right shrink-0 ml-2">
                      <div class="text-[10px] font-medium text-slate-400 uppercase tracking-wider mb-0.5">Rückgabe bis</div>
                      <div class="text-xs font-bold {new Date(b.rueckgabe_frist) < new Date() ? 'text-rose-600' : 'text-slate-700'}">
                        {new Date(b.rueckgabe_frist).toLocaleDateString('de-DE')}
                      </div>
                    </div>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>

  <!-- Glassmorphic Damage Note Modal -->
  {#if showModal}
    <div class="fixed inset-0 bg-slate-900/10 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="w-full max-w-md p-6 rounded-3xl bg-white border border-slate-100 shadow-2xl space-y-4 animate-scale-up">
        <h3 class="font-bold text-slate-800">Schadensnotiz aktualisieren</h3>
        <p class="text-xs text-slate-500">Verfasse einen kurzen Zustandsbericht für Exemplar <strong class="text-slate-700">{activeCopy?.barcode_id}</strong>.</p>
        
        <textarea bind:value={newNote} rows="3" class="w-full bg-slate-50 border border-slate-200 rounded-xl p-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-300 text-slate-800" placeholder="Schaden beschreiben..."></textarea>
        
        <div class="flex justify-end space-x-3 text-sm pt-2">
          <button onclick={() => { showModal = false; activeCopy = null; }} class="px-4 py-2 rounded-xl bg-slate-100 text-slate-650 hover:bg-slate-200 transition-colors cursor-pointer">Abbrechen</button>
          <button onclick={updateNote} class="px-4 py-2 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 transition-colors cursor-pointer">Speichern</button>
        </div>
      </div>
    </div>
  {/if}
{/if}
