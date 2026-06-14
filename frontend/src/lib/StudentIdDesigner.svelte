<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  /**
   * @file StudentIdDesigner.svelte
   * Canvas-based ID-card designer — top-level coordinator component.
   *
   * Runes in use:
   *   $state        — UI flags: selectedId, side, printMode, printSide, zoom,
   *                   classesList, selectedKlasse, previewStudents,
   *                   loadingStudents, activeWebcamStudent, timestamp
   *   $state.raw    — classesList, previewStudents (wholesale-replaced on fetch;
   *                   never mutated in-place → $state.raw avoids deep-proxy cost)
   *   $derived      — previewStudent (first student or mock fallback)
   *   onMount       — one-shot initial data load (semantically correct for fetch)
   *
   * Sub-components:
   *   Toolbar.svelte         — print / zoom / class / add-element controls
   *   CanvasArea.svelte      — interactive draggable / resizable card canvas
   *   PropertiesPanel.svelte — dynamic properties inspector for selected element
   *   PrintPreview.svelte    — hidden print-output DOM sections
   *   WebcamCapture.svelte   — webcam photo capture overlay
   */
  import { onMount } from "svelte";
  import { idStore } from "./designer/idDesignerStore.svelte.js";
  import CanvasArea from "./designer/CanvasArea.svelte";
  import PropertiesPanel from "./designer/PropertiesPanel.svelte";
  import Toolbar from "./designer/Toolbar.svelte";
  import PrintPreview from "./designer/PrintPreview.svelte";
  import WebcamCapture from "./WebcamCapture.svelte";

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------

  /** Currently selected element ID on the canvas (null = none). */
  let selectedId = $state(/** @type {string|null} */ (null));

  /** Active design side shown in the canvas. */
  let side = $state(/** @type {"front"|"back"} */ ("front"));

  /** Print-mode selector value — card printer vs A4 sheet. */
  let printMode = $state(/** @type {"card"|"a4"} */ ("card"));

  /** Canvas zoom percentage (80–300). */
  let zoom = $state(150);

  /** List of class names returned by /api/klassen. */
  let classesList = $state.raw(/** @type {string[]} */ ([]));

  /** Currently selected class name. */
  let selectedKlasse = $state("");

  /** Students belonging to selectedKlasse (or mock fallback). */
  let previewStudents = $state.raw(/** @type {any[]} */ ([]));

  /** Whether a student-list fetch is in flight. */
  let loadingStudents = $state(false);

  /** Student whose webcam overlay is currently open (null = closed). */
  let activeWebcamStudent = $state(/** @type {any} */ (null));

  /** Cache-busting timestamp for photo <img> src attributes. */
  let timestamp = $state(Date.now());

  // ---------------------------------------------------------------------------
  // Derived
  // ---------------------------------------------------------------------------

  const mockStudents = [
    { id: "s1", barcode_id: "S-10041", vorname: "Maximilian", nachname: "Schmidt", klasse: "9a" },
    { id: "s2", barcode_id: "S-10042", vorname: "Sophie",     nachname: "Fischer", klasse: "9a" },
  ];

  /** First student in the list, or a mock for layout preview. */
  const previewStudent = $derived(previewStudents[0] ?? mockStudents[0]);

  // ---------------------------------------------------------------------------
  // Data loading
  // ---------------------------------------------------------------------------

  async function loadClasses() {
    try {
      const res = await apiFetch("/api/klassen");
      if (res.ok) {
        classesList = await res.json();
        if (classesList.length > 0) {
          selectedKlasse = classesList[0];
          await loadStudents(selectedKlasse);
          return;
        }
      }
    } catch { /* network error — fall through to mocks */ }
    previewStudents = mockStudents;
  }

  /** @param {string} klasse */
  async function loadStudents(klasse) {
    if (!klasse) return;
    loadingStudents = true;
    try {
      const res = await apiFetch(`/api/schueler?klasse=${encodeURIComponent(klasse)}`);
      if (res.ok) {
        const data = await res.json();
        previewStudents = data.length > 0 ? data : mockStudents;
      } else {
        previewStudents = mockStudents;
      }
    } catch {
      previewStudents = mockStudents;
    } finally {
      loadingStudents = false;
    }
  }

  onMount(() => { loadClasses(); });

  // ---------------------------------------------------------------------------
  // Print
  // ---------------------------------------------------------------------------

  function triggerPrint() {
    const style = document.createElement("style");
    if (printMode === "a4") {
      style.textContent = "@media print { @page { size: A4; margin: 0; } }";
      document.body.setAttribute("data-print-mode", "a4");
    } else {
      style.textContent = "@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }";
      document.body.setAttribute("data-print-mode", "card");
    }
    document.body.setAttribute("data-print-side", side);
    document.head.appendChild(style);
    window.print();
    document.head.removeChild(style);
    document.body.removeAttribute("data-print-mode");
    document.body.removeAttribute("data-print-side");
  }
</script>

<div class="w-full space-y-5 no-print text-slate-800 animate-fade-in font-sans">
  <!-- Toolbar: print controls, class picker, zoom, add-element buttons -->
  <Toolbar
    {zoom}           onZoom={(v) => { zoom = v; }}
    {side}           onSide={(s) => { side = s; selectedId = null; }}
    {printMode}      onPrintMode={(m) => { printMode = m; }}
    onPrint={triggerPrint}
    {classesList}
    {selectedKlasse} onKlasse={(k) => { selectedKlasse = k; loadStudents(k); }}
    barcodeType={idStore.barcodeType}
    onBarcodeType={(t) => { idStore.barcodeType = t; }}
    {loadingStudents}
    {previewStudent}
  />

  <!-- Main workspace: canvas + properties panel -->
  <div class="w-full flex flex-col lg:flex-row gap-5">
    <CanvasArea
      {side}
      {selectedId}
      onSelect={(id) => { selectedId = id; }}
      student={previewStudent}
      {zoom}
      barcodeType={idStore.barcodeType}
      {timestamp}
      onWebcam={(student) => { activeWebcamStudent = student; }}
    />

    <PropertiesPanel
      {selectedId}
      {side}
    />
  </div>
</div>

<!-- Hidden print-output sections (rendered by PrintPreview) -->
<PrintPreview
  students={previewStudents.length > 0 ? previewStudents : mockStudents}
  barcodeType={idStore.barcodeType}
  {timestamp}
/>

<!-- Webcam overlay (only mounted when a student is selected for photo capture) -->
{#if activeWebcamStudent}
  <WebcamCapture
    studentId={activeWebcamStudent.id}
    onCapture={() => { timestamp = Date.now(); activeWebcamStudent = null; loadStudents(selectedKlasse); }}
    onClose={() => { activeWebcamStudent = null; }}
  />
{/if}
