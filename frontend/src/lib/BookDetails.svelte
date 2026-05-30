<script>
  import AntolinBadge from './AntolinBadge.svelte';

  // Props
  let { title = { id: "1", titel: "LMF-Mathe 9", autor: "Dr. L. Müller", verlag: "Klett", erscheinungsjahr: 2023 } } = $props();

  // State Runes
  /** @type {any[]} */
  let copies = $state([]);
  let loadingCopies = $state(false);
  let showModal = $state(false);
  /** @type {any} */
  let activeCopy = $state(null);
  let newNote = $state("");

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
      const res = await fetch(`/api/buecher/titel/${title.id}/exemplare`);
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
      const res = await fetch(`/api/buecher/exemplare/${activeCopy.id}/schadensnotiz`, {
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
      const res = await fetch(`/api/buecher/exemplare/${copyId}`, {
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
        </div>
      </div>
      <div class="text-sm bg-slate-50 border border-slate-100 rounded-2xl py-2 px-4 flex items-center gap-3">
        <span class="text-slate-400 font-mono">Exemplare:</span>
        <span class="font-bold text-slate-700">{copies.length} im Bestand</span>
      </div>
    </div>

    <!-- Copies Grid -->
    {#if loadingCopies}
      <div class="py-12 flex justify-center items-center">
        <div class="w-8 h-8 border-4 border-slate-800 border-t-transparent rounded-full animate-spin"></div>
      </div>
    {:else if copies.length === 0}
      <div class="py-12 flex flex-col items-center justify-center text-slate-400 space-y-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-slate-350" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
        <span class="text-xs font-semibold">Keine Exemplare im Bestand gefunden.</span>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each copies as copy}
          <div class="p-4 rounded-2xl bg-slate-50/50 border border-slate-100 flex flex-col justify-between hover:border-slate-200 transition-all duration-300">
            <div class="flex items-start justify-between">
              <div class="space-y-1">
                <span class="text-xs font-mono font-bold text-blue-700 bg-blue-50 border border-blue-100/50 px-2 py-0.5 rounded">
                  {copy.barcode_id}
                </span>
                <p class="text-xs text-slate-650 pt-1.5"><strong class="text-slate-500 font-medium">Zustand:</strong> {copy.zustand_notiz || 'Neuwertig'}</p>
              </div>
              
              <!-- Copy quick Actions -->
              <div class="flex space-x-1">
                <button onclick={() => printLabel(copy.barcode_id)} class="p-1.5 text-slate-450 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors cursor-pointer" title="Schnelldruck Barcode-Etikett">
                  🖨️
                </button>
                <button onclick={() => { activeCopy = copy; newNote = copy.zustand_notiz; showModal = true; }} class="p-1.5 text-slate-450 hover:text-amber-600 hover:bg-amber-50 rounded-lg transition-colors cursor-pointer" title="Schadensnotiz bearbeiten">
                  ✏️
                </button>
                <button onclick={() => deleteCopy(copy.id)} class="p-1.5 text-slate-450 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer" title="Exemplar löschen">
                  🗑️
                </button>
              </div>
            </div>
          </div>
        {/each}
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
