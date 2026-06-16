<script>
  import { apiFetch, apiClient } from "./apiFetch.js";
  /**
   * @file StudentIdDesigner.svelte
   * Canvas-based ID-card designer — top-level coordinator component.
   */
  import { onMount } from "svelte";
  import { idStore } from "./designer/idDesignerStore.svelte.js";
  import CanvasArea from "./designer/CanvasArea.svelte";
  import PropertiesPanel from "./designer/PropertiesPanel.svelte";
  import Toolbar from "./designer/Toolbar.svelte";
  import PrintPreview from "./designer/PrintPreview.svelte";

  let selectedId = $state(/** @type {string|null} */ (null));
  let side = $state(/** @type {"front"|"back"} */ ("front"));
  let printMode = $state(/** @type {"card"|"a4"} */ ("card"));
  let zoom = $state(150);
  let classesList = $state.raw(/** @type {string[]} */ ([]));
  let selectedKlasse = $state("");
  let previewStudents = $state.raw(/** @type {any[]} */ ([]));
  let loadingStudents = $state(false);
  let timestamp = $state(Date.now());

  const mockStudents = [
    { id: "s1", barcode_id: "S-10041", vorname: "Maximilian", nachname: "Schmidt", klasse: "9a" },
    { id: "s2", barcode_id: "S-10042", vorname: "Sophie",     nachname: "Fischer", klasse: "9a" },
  ];

  const previewStudent = $derived(previewStudents[0] ?? mockStudents[0]);

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

  <div class="w-full flex flex-col lg:flex-row gap-5">
    <CanvasArea
      {side}
      {selectedId}
      onSelect={(id) => { selectedId = id; }}
      student={previewStudent}
      {zoom}
      barcodeType={idStore.barcodeType}
      {timestamp}
    />

    <PropertiesPanel
      {selectedId}
      {side}
    />
  </div>
</div>

<PrintPreview
  students={previewStudents.length > 0 ? previewStudents : mockStudents}
  barcodeType={idStore.barcodeType}
  {timestamp}
/>
