<script>
  import { onMount } from "svelte";
  import WebcamCapture from "./WebcamCapture.svelte";

  /** @type {string[]} */
  let classesList = $state([]);
  let selectedKlasse = $state("");
  /** @type {any[]} */
  let previewStudents = $state([]);
  let loadingStudents = $state(false);
  let barcodeType = $state("code39");
  let cardTheme = $state("bg-white text-black border-slate-200");
  /** @type {any} */
  let activeWebcamStudent = $state(null);
  let timestamp = $state(Date.now());

  /** @type {Record<string, any>} */
  let layout = $state({
    header: { x: 5, y: 4, scale: 1.0, show: true, text: "STÄDTISCHES GYMNASIUM MUSTERSTADT" },
    photo: { x: 5, y: 12, scale: 1.0, show: true },
    name: { x: 30, y: 14, scale: 1.1, show: true },
    details: { x: 30, y: 21, scale: 0.9, show: true },
    validity: { x: 30, y: 27, scale: 0.85, show: true },
    barcode: { x: 30, y: 34, scale: 0.8, show: true },
    logo: { x: 68, y: 12, scale: 1.0, show: true, url: "" },
    address: { x: 30, y: 8, scale: 0.8, show: true, text: "Musterstraße 12, 12345 Musterstadt" }
  });

  let activeElement = $state("name");

  const themes = [
    { value: "bg-white text-black border-slate-200", name: "Standard Weiß" },
    { value: "bg-slate-50 text-slate-900 border-slate-200", name: "Dezentes Grau" },
    { value: "bg-linear-to-tr from-emerald-50/40 to-teal-50/40 text-zinc-900 border-emerald-100", name: "Smaragd" },
    { value: "bg-linear-to-tr from-blue-50/40 to-indigo-50/40 text-zinc-900 border-blue-100", name: "Klassik" }
  ];

  const elementsList = [
    { key: "header", name: "Kopfzeile", maxX: 80, maxY: 50 },
    { key: "logo", name: "Schullogo", maxX: 80, maxY: 50 },
    { key: "address", name: "Schuladresse", maxX: 80, maxY: 50 },
    { key: "photo", name: "Passbild", maxX: 80, maxY: 50 },
    { key: "name", name: "Name", maxX: 80, maxY: 50 },
    { key: "details", name: "Klasse/Details", maxX: 80, maxY: 50 },
    { key: "validity", name: "Gültigkeit", maxX: 80, maxY: 50 },
    { key: "barcode", name: "Barcode", maxX: 80, maxY: 50 }
  ];

  const mockStudents = [
    { id: "s1", barcode_id: "S-10041", vorname: "Maximilian", nachname: "Schmidt", klasse: "9a" },
    { id: "s2", barcode_id: "S-10042", vorname: "Sophie", nachname: "Fischer", klasse: "9a" }
  ];

  async function loadClasses() {
    try {
      const res = await fetch("/api/klassen");
      if (res.ok) {
        classesList = await res.json();
        if (classesList.length > 0) {
          selectedKlasse = classesList[0];
          await loadStudents(selectedKlasse);
          return;
        }
      }
      previewStudents = mockStudents;
    } catch {
      previewStudents = mockStudents;
    }
  }

  /** @param {string} klasse */
  async function loadStudents(klasse) {
    if (!klasse) return;
    loadingStudents = true;
    try {
      const res = await fetch(`/api/schueler?klasse=${encodeURIComponent(klasse)}`);
      if (res.ok) {
        const data = await res.json();
        previewStudents = data.length > 0 ? data : mockStudents;
      }
    } catch {
      previewStudents = mockStudents;
    } finally {
      loadingStudents = false;
    }
  }

  onMount(() => { loadClasses(); });

  /** @param {any} e */
  function handleLogoUpload(e) {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        if (event.target && typeof event.target.result === "string") {
          layout.logo.url = event.target.result;
        }
      };
      reader.readAsDataURL(file);
    }
  }

  /**
   * @param {string} key
   * @returns {any}
   */
  function getLayoutEl(key) {
    switch (key) {
      case "header": return layout.header;
      case "photo": return layout.photo;
      case "name": return layout.name;
      case "details": return layout.details;
      case "validity": return layout.validity;
      case "barcode": return layout.barcode;
      case "logo": return layout.logo;
      case "address": return layout.address;
      default: return {};
    }
  }


  /**
   * @param {any} e
   * @param {string} elementKey
   */
  function startDrag(e, elementKey) {
    if (e.button !== 0) return;
    e.preventDefault();
    const targetEl = /** @type {any} */ (e.currentTarget);
    const cardEl = targetEl ? targetEl.closest(".card-container") : null;
    if (!cardEl) return;
    const rect = cardEl.getBoundingClientRect();
    const pxToMm = 85.6 / rect.width;
    const startX = e.clientX, startY = e.clientY;
    const elLayout = getLayoutEl(elementKey);
    const initialX = elLayout.x, initialY = elLayout.y;
    let hasMoved = false;
    activeElement = elementKey;

    /** @param {any} moveEvent */
    function onPointerMove(moveEvent) {
      const dx = moveEvent.clientX - startX;
      const dy = moveEvent.clientY - startY;
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasMoved = true;
      elLayout.x = Math.max(0, Math.min(80, Math.round((initialX + dx * pxToMm) * 10) / 10));
      elLayout.y = Math.max(0, Math.min(50, Math.round((initialY + dy * pxToMm) * 10) / 10));
    }

    function onPointerUp() {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
      if (!hasMoved && elementKey === "photo" && targetEl) {
        const studentId = targetEl.getAttribute("data-student-id");
        activeWebcamStudent = previewStudents.find(s => s.id === studentId) || null;
      }
    }
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
  }
</script>

<div class="w-full space-y-6 no-print text-slate-800 animate-fade-in font-sans">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-100 pb-5">
    <div>
      <span class="text-xs font-semibold text-slate-400 tracking-wider uppercase">Ausweis-Designer</span>
      <h2 class="text-2xl font-bold text-slate-900">Schülerausweise drucken (ID-1)</h2>
      <p class="text-xs text-slate-500 font-medium">Gestalte das Kartenlayout live und nimm Fotos direkt über die Kamera auf.</p>
    </div>
    <button onclick={() => window.print()} class="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold transition-all flex items-center gap-2 shadow-xs cursor-pointer">
      <span>🖨️ Ausweise drucken</span>
    </button>
  </div>

  <div class="p-5 rounded-2xl bg-slate-50 border border-slate-100 grid grid-cols-1 md:grid-cols-2 gap-4 items-end font-sans">
    <div class="space-y-1.5 text-left">
      <span class="text-[10px] uppercase font-bold text-slate-450 font-mono">Klasse auswählen</span>
      {#if classesList.length > 0}
        <select bind:value={selectedKlasse} onchange={() => loadStudents(selectedKlasse)} class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-705 focus:outline-none">
          {#each classesList as kl}
            <option value={kl}>Klasse {kl}</option>
          {/each}
        </select>
      {:else}
        <div class="text-xs text-slate-400 font-medium py-2">Lade Klassen...</div>
      {/if}
    </div>
    <div class="space-y-1.5 text-left">
      <span class="text-[10px] uppercase font-bold text-slate-450 font-mono">Barcode-Typ</span>
      <select bind:value={barcodeType} class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-705 focus:outline-none">
        <option value="code39">Code39 (Standard 1D)</option>
        <option value="qr">QR-Code (2D)</option>
      </select>
    </div>
  </div>

  <div class="w-full flex flex-col lg:flex-row gap-6 font-sans">
    <div class="flex-1 flex flex-col items-center justify-start p-6 bg-slate-50/50 border border-dashed border-slate-200 rounded-3xl min-h-[450px]">
      <span class="text-[10px] uppercase tracking-widest text-slate-400 font-bold font-mono mb-4">Live-Arbeitsfläche (Ziehe Elemente per Drag&Drop)</span>
      {#if loadingStudents}
        <div class="grow flex items-center justify-center py-12">
          <div class="w-8 h-8 border-4 border-t-blue-600 border-slate-200 rounded-full animate-spin"></div>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 justify-center w-full">
          {#each previewStudents as student}
            <div class="card-container shadow-lg relative border border-slate-200 rounded-[8px] overflow-hidden select-none bg-white shrink-0" style="width: 85.6mm; height: 53.98mm;">
              <div class="w-full h-full relative {cardTheme}">
                {#if layout.header.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "header")} class="absolute font-black text-center tracking-tight leading-none truncate text-slate-800 cursor-move p-0.5 rounded-xs {activeElement === 'header' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.header.x}mm; top: {layout.header.y}mm; transform: scale({layout.header.scale}); transform-origin: top left; font-size: 7.5pt; width: {85.6 - layout.header.x * 2}mm;">{layout.header.text}</div>
                {/if}
                {#if layout.logo.show}
                  <button onpointerdown={(e) => startDrag(e, "logo")} class="absolute border border-dashed border-slate-300 bg-slate-50/50 flex items-center justify-center overflow-hidden cursor-move rounded-xs {activeElement === 'logo' ? 'ring-1 ring-blue-500' : 'hover:ring-1 hover:ring-slate-350'}" style="left: {layout.logo.x}mm; top: {layout.logo.y}mm; width: {12 * layout.logo.scale}mm; height: {12 * layout.logo.scale}mm;">
                    {#if layout.logo.url}
                      <img src={layout.logo.url} class="w-full h-full object-contain pointer-events-none" alt="Logo" />
                    {:else}
                      <span class="text-[5px] text-slate-400 font-bold">LOGO</span>
                    {/if}
                  </button>
                {/if}
                {#if layout.address.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "address")} class="absolute font-semibold tracking-tight opacity-75 leading-none text-slate-650 cursor-move p-0.5 rounded-xs {activeElement === 'address' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.address.x}mm; top: {layout.address.y}mm; transform: scale({layout.address.scale}); transform-origin: top left; font-size: 6.5pt; width: {85.6 - layout.address.x - 2}mm; max-height: 12mm; overflow: hidden;">{layout.address.text}</div>
                {/if}
                {#if layout.photo.show}
                  <button onpointerdown={(e) => startDrag(e, "photo")} data-student-id={student.id} class="absolute border border-dashed border-slate-350 bg-slate-50 hover:bg-slate-100 hover:border-blue-550 flex items-center justify-center font-bold text-slate-400 rounded-sm leading-none overflow-hidden group cursor-move transition-all duration-200 {activeElement === 'photo' ? 'ring-1 ring-blue-500' : ''}" style="left: {layout.photo.x}mm; top: {layout.photo.y}mm; width: {22 * layout.photo.scale}mm; height: {28 * layout.photo.scale}mm; font-size: 6pt;">
                    <img src="/uploads/fotos/{student.barcode_id}.jpg?t={timestamp}" onerror={(e) => { const img = /** @type {any} */ (e.currentTarget); img.style.display = 'none'; const sib = img.nextElementSibling; if (sib) { /** @type {any} */ (sib).style.display = 'flex'; } }} class="w-full h-full object-cover pointer-events-none" alt="Passbild" />
                    <div class="absolute inset-0 flex flex-col items-center justify-center bg-slate-50 text-[5px] text-slate-450 group-hover:text-blue-600 transition-colors pointer-events-none" style="display: none;">
                      <span>KEIN FOTO</span>
                      <span class="text-[4.5px] text-slate-500 mt-1">📸 NEUES BILD</span>
                    </div>
                    <div class="absolute inset-0 bg-slate-900/60 text-white flex flex-col items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4.5 w-4.5 text-white/90" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                      <span class="text-[4px] uppercase tracking-wider font-bold mt-0.5">Ändern</span>
                    </div>
                  </button>
                {/if}
                {#if layout.name.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "name")} class="absolute font-extrabold tracking-tight leading-none text-slate-900 cursor-move p-0.5 rounded-xs {activeElement === 'name' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.name.x}mm; top: {layout.name.y}mm; transform: scale({layout.name.scale}); transform-origin: top left; font-size: 9pt;">{student.vorname} {student.nachname}</div>
                {/if}
                {#if layout.details.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "details")} class="absolute font-semibold tracking-tight opacity-75 leading-none text-slate-650 cursor-move p-0.5 rounded-xs {activeElement === 'details' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.details.x}mm; top: {layout.details.y}mm; transform: scale({layout.details.scale}); transform-origin: top left; font-size: 7.5pt;">Klasse: {student.klasse}</div>
                {/if}
                {#if layout.validity.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "validity")} class="absolute font-semibold tracking-tight opacity-75 leading-none text-slate-650 cursor-move p-0.5 rounded-xs {activeElement === 'validity' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.validity.x}mm; top: {layout.validity.y}mm; transform: scale({layout.validity.scale}); transform-origin: top left; font-size: 7pt;">Gültig bis: 31.07.2027</div>
                {/if}
                {#if layout.barcode.show}
                  <div role="presentation" onpointerdown={(e) => startDrag(e, "barcode")} class="absolute flex flex-col items-center leading-none cursor-move p-0.5 rounded-xs {activeElement === 'barcode' ? 'ring-1 ring-blue-500 bg-blue-50/20' : 'hover:ring-1 hover:ring-slate-300 hover:ring-dashed'}" style="left: {layout.barcode.x}mm; top: {layout.barcode.y}mm; transform: scale({layout.barcode.scale}); transform-origin: top left;">
                    <img src="/api/barcode?content={student.barcode_id}&qr={barcodeType === 'qr'}&width={barcodeType === 'qr' ? 80 : 200}&height={barcodeType === 'qr' ? 80 : 50}" class="{barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain pointer-events-none" alt="Barcode" />
                    <span class="font-mono font-bold mt-1 text-[6.5pt] tracking-widest text-slate-700 pointer-events-none">{student.barcode_id}</span>
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="w-full lg:w-80 bg-white border border-slate-100 p-5 rounded-2xl shadow-xl space-y-5 shrink-0 text-left">
      <div>
        <h3 class="text-xs font-semibold text-slate-505 uppercase tracking-wider font-mono">Layout-Steuerung</h3>
        <p class="text-[10px] text-slate-400">Passe Felder live per Slider oder Drag&Drop an.</p>
      </div>
      <div class="space-y-1.5">
        <span class="text-[10px] uppercase font-bold text-slate-450 font-mono">Karten-Hintergrund</span>
        <select bind:value={cardTheme} class="w-full bg-white border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-705 focus:outline-none">
          {#each themes as t}
            <option value={t.value}>{t.name}</option>
          {/each}
        </select>
      </div>
      <div class="space-y-3.5 pt-3 border-t border-slate-100">
        <span class="text-[10px] uppercase font-bold text-slate-450 font-mono">Element-Optionen</span>
        <div class="space-y-3 divide-y divide-slate-100">
          {#each elementsList as el}
            <div class="pt-3 first:pt-0 space-y-2 font-sans">
              <div class="flex items-center justify-between">
                <button onclick={() => activeElement = el.key} class="text-xs font-bold hover:text-blue-600 transition-colors text-left {activeElement === el.key ? 'text-blue-600' : 'text-slate-650'}">
                  {el.name}
                </button>
                <label class="relative inline-flex items-center cursor-pointer select-none">
                  <input type="checkbox" checked={getLayoutEl(el.key).show} onchange={(e) => { getLayoutEl(el.key).show = e.currentTarget.checked; }} class="sr-only peer" />
                  <div class="w-7 h-4 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>
              {#if activeElement === el.key && getLayoutEl(el.key).show}
                <div class="space-y-2 pl-2 border-l-2 border-blue-500 py-1 transition-all">
                  {#if el.key === 'header'}
                    <div class="space-y-1">
                      <span class="text-[9px] text-slate-450 font-bold uppercase block">Text</span>
                      <input type="text" bind:value={layout.header.text} class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1 text-xs text-slate-705 focus:outline-none" />
                    </div>
                  {:else if el.key === 'address'}
                    <div class="space-y-1">
                      <span class="text-[9px] text-slate-450 font-bold uppercase block">Schuladresse</span>
                      <input type="text" bind:value={layout.address.text} class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1 text-xs text-slate-705 focus:outline-none" />
                    </div>
                  {:else if el.key === 'logo'}
                    <div class="space-y-1">
                      <span class="text-[9px] text-slate-450 font-bold uppercase block">Logo Hochladen</span>
                      <input type="file" accept="image/*" onchange={handleLogoUpload} class="w-full text-xs text-slate-500 file:mr-2 file:py-1 file:px-2 file:rounded-md file:border-0 file:text-[10px] file:font-semibold file:bg-slate-100 file:text-slate-700 hover:file:bg-slate-200 cursor-pointer" />
                    </div>
                  {/if}
                  <div class="space-y-1">
                    <div class="flex justify-between text-[10px] text-slate-400">
                      <span>X-Position</span>
                      <span class="font-mono font-bold text-blue-600">{getLayoutEl(el.key).x} mm</span>
                    </div>
                    <input type="range" min="0" max={el.maxX} step="0.5" value={getLayoutEl(el.key).x} oninput={(e) => { getLayoutEl(el.key).x = parseFloat(e.currentTarget.value); }} class="w-full accent-blue-600 h-1 bg-slate-150 rounded-lg cursor-pointer" />
                  </div>
                  <div class="space-y-1">
                    <div class="flex justify-between text-[10px] text-slate-400">
                      <span>Y-Position</span>
                      <span class="font-mono font-bold text-blue-600">{getLayoutEl(el.key).y} mm</span>
                    </div>
                    <input type="range" min="0" max={el.maxY} step="0.5" value={getLayoutEl(el.key).y} oninput={(e) => { getLayoutEl(el.key).y = parseFloat(e.currentTarget.value); }} class="w-full accent-blue-600 h-1 bg-slate-150 rounded-lg cursor-pointer" />
                  </div>
                  <div class="space-y-1">
                    <div class="flex justify-between text-[10px] text-slate-400">
                      <span>Größe</span>
                      <span class="font-mono font-bold text-blue-600">{getLayoutEl(el.key).scale.toFixed(2)}x</span>
                    </div>
                    <input type="range" min="0.4" max="2.2" step="0.05" value={getLayoutEl(el.key).scale} oninput={(e) => { getLayoutEl(el.key).scale = parseFloat(e.currentTarget.value); }} class="w-full accent-blue-600 h-1 bg-slate-150 rounded-lg cursor-pointer" />
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
</div>

<div class="print-rendered-output hidden print:block">
  {#each previewStudents as student}
    <div class="print-card-box {cardTheme}">
      {#if layout.header.show}
        <div class="absolute font-black text-center tracking-tight leading-none truncate text-black" style="left: {layout.header.x}mm; top: {layout.header.y}mm; transform: scale({layout.header.scale}); transform-origin: top left; font-size: 7.5pt; width: {85.6 - layout.header.x * 2}mm;">{layout.header.text}</div>
      {/if}
      {#if layout.logo.show && layout.logo.url}
        <div class="absolute overflow-hidden flex items-center justify-center" style="left: {layout.logo.x}mm; top: {layout.logo.y}mm; width: {12 * layout.logo.scale}mm; height: {12 * layout.logo.scale}mm;">
          <img src={layout.logo.url} class="w-full h-full object-contain" alt="Logo" />
        </div>
      {/if}
      {#if layout.address.show}
        <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800" style="left: {layout.address.x}mm; top: {layout.address.y}mm; transform: scale({layout.address.scale}); transform-origin: top left; font-size: 6.5pt; width: {85.6 - layout.address.x - 2}mm; max-height: 12mm; overflow: hidden;">{layout.address.text}</div>
      {/if}
      {#if layout.photo.show}
        <div class="absolute border border-solid border-zinc-300 bg-zinc-55 flex items-center justify-center overflow-hidden rounded-xs" style="left: {layout.photo.x}mm; top: {layout.photo.y}mm; width: {22 * layout.photo.scale}mm; height: {28 * layout.photo.scale}mm;">
          <img src="/uploads/fotos/{student.barcode_id}.jpg?t={timestamp}" onerror={(e) => { (/** @type {any} */ (e.currentTarget)).style.display = 'none'; }} class="w-full h-full object-cover" alt="Passbild" />
        </div>
      {/if}
      {#if layout.name.show}
        <div class="absolute font-extrabold tracking-tight leading-none text-black" style="left: {layout.name.x}mm; top: {layout.name.y}mm; transform: scale({layout.name.scale}); transform-origin: top left; font-size: 9pt;">{student.vorname} {student.nachname}</div>
      {/if}
      {#if layout.details.show}
        <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800" style="left: {layout.details.x}mm; top: {layout.details.y}mm; transform: scale({layout.details.scale}); transform-origin: top left; font-size: 7.5pt;">Klasse: {student.klasse}</div>
      {/if}
      {#if layout.validity.show}
        <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800" style="left: {layout.validity.x}mm; top: {layout.validity.y}mm; transform: scale({layout.validity.scale}); transform-origin: top left; font-size: 7pt;">Gültig bis: 31.07.2027</div>
      {/if}
      {#if layout.barcode.show}
        <div class="absolute flex flex-col items-center leading-none" style="left: {layout.barcode.x}mm; top: {layout.barcode.y}mm; transform: scale({layout.barcode.scale}); transform-origin: top left;">
          <img src="/api/barcode?content={student.barcode_id}&qr={barcodeType === 'qr'}&width={barcodeType === 'qr' ? 80 : 200}&height={barcodeType === 'qr' ? 80 : 50}" class="{barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain" alt="Barcode" />
          <span class="font-mono font-bold mt-1 text-[6.5pt] tracking-widest text-zinc-800">{student.barcode_id}</span>
        </div>
      {/if}
    </div>
  {/each}
</div>

{#if activeWebcamStudent}
  <WebcamCapture studentId={(/** @type {any} */ (activeWebcamStudent)).id} onCapture={() => { timestamp = Date.now(); activeWebcamStudent = null; }} onClose={() => activeWebcamStudent = null} />
{/if}

<style>
  @media print {
    @page {
      size: 85.6mm 53.98mm;
      margin: 0;
    }
    :global(html, body) {
      margin: 0 !important;
      padding: 0 !important;
      width: 85.6mm !important;
      height: 53.98mm !important;
      background: white !important;
      overflow: hidden !important;
    }
    :global(main, .min-h-screen, .flex) {
      margin: 0 !important;
      padding: 0 !important;
      display: block !important;
      width: 85.6mm !important;
      height: 53.98mm !important;
      background: white !important;
      border: none !important;
      box-shadow: none !important;
    }
    :global(.no-print) {
      display: none !important;
    }
    .print-rendered-output {
      display: block !important;
      position: fixed !important;
      left: 0 !important;
      top: 0 !important;
      width: 85.6mm !important;
      height: 53.98mm !important;
      z-index: 99999 !important;
      background: white !important;
    }
    .print-card-box {
      width: 85.6mm !important;
      height: 53.98mm !important;
      page-break-after: always !important;
      page-break-inside: avoid !important;
      position: relative !important;
      overflow: hidden !important;
      box-sizing: border-box !important;
      -webkit-print-color-adjust: exact;
      print-color-adjust: exact;
    }
  }
</style>
