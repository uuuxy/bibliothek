<script>
  import { apiFetch, apiClient } from "./apiFetch.js";

  /**
   * @typedef {Object} Props
   * @property {any[]} copies
   * @property {any} title
   * @property {boolean} loadingCopies
   */
  /** @type {Props} */
  let { copies = $bindable([]), title, loadingCopies } = $props();

  let showModal = $state(false);
  /** @type {any} */
  let activeCopy = $state(null);
  let newNote = $state("");

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
    titleDiv.textContent = title?.titel || "";
    body.appendChild(titleDiv);

    const scriptEl = printWindow.document.createElement("script");
    scriptEl.textContent = "window.print(); setTimeout(function() { window.close(); }, 500);";
    body.appendChild(scriptEl);
  }

  // Update physical condition notes
  async function updateNote() {
    if (!activeCopy) return;
    try {
      const res = await apiClient.post(`/api/buecher/exemplare/${activeCopy.id}/schadensnotiz`, { note: newNote });
      if (!res.ok) throw new Error("Fehler beim Speichern der Notiz");
      
      // Update local state
      const idx = copies.findIndex((/** @type {any} */ c) => c.id === activeCopy.id);
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
      copies = copies.filter((/** @type {any} */ c) => c.id !== copyId);
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
      const idx = copies.findIndex((/** @type {any} */ c) => c.id === copyId);
      if (idx !== -1) {
        copies[idx] = { ...copies[idx], ist_ausgesondert: true, ist_ausleihbar: false };
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      alert(error.message);
    }
  }
</script>

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
              <button onclick={() => printLabel(copy.barcode_id)} aria-label="Schnelldruck Barcode-Etikett" class="p-1.5 text-slate-450 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors cursor-pointer" title="Schnelldruck Barcode-Etikett">
                <span aria-hidden="true">🖨️</span>
              </button>
              <button onclick={() => { activeCopy = copy; newNote = copy.zustand_notiz; showModal = true; }} aria-label="Schadensnotiz bearbeiten" class="p-1.5 text-slate-450 hover:text-amber-600 hover:bg-amber-50 rounded-lg transition-colors cursor-pointer" title="Schadensnotiz bearbeiten">
                <span aria-hidden="true">✏️</span>
              </button>
              <button onclick={() => aussondernCopy(copy.id)} aria-label="Exemplar aussondern (Makulatur)" class="p-1.5 text-slate-450 hover:text-orange-600 hover:bg-orange-50 rounded-lg transition-colors cursor-pointer" title="Exemplar aussondern (Makulatur)">
                <span aria-hidden="true">⛔</span>
              </button>
            {/if}
            <button onclick={() => deleteCopy(copy.id)} aria-label="Exemplar löschen" class="p-1.5 text-slate-450 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer" title="Exemplar löschen">
              <span aria-hidden="true">🗑️</span>
            </button>
          </div>
        </div>
      </div>
    {/each}
  </div>
{/if}

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
