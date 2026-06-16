<script>
  /**
   * @file CanvasArea.svelte
   * Interactive canvas for the ID card designer.
   *
   * Runes in use:
   *   $props()      — receives `side`, `selectedId`, `onSelect`, `student`, `zoom`
   *   $derived      — computes sorted (by zIndex) elements from the store
   *   $effect       — not used here; all event cleanup is handled inline via onDestroy
   *
   * Drag & Resize:
   *   Both operations bind pointermove/pointerup on `window` to allow the pointer
   *   to travel outside the card boundary. `activeDragCleanup` stores the teardown
   *   function and is called by onDestroy to prevent listener leaks on unmount.
   */
  import { onDestroy } from 'svelte';
  import { idStore } from './idDesignerStore.svelte.js';

  /** @type {{ side: 'front'|'back', selectedId: string|null, onSelect: (id: string|null)=>void, student: any, zoom: number, barcodeType: string, timestamp: number, onWebcam: (student: any)=>void }} */
  const { side, selectedId, onSelect, student, zoom, barcodeType, timestamp, onWebcam } = $props();

  /** Elements for the active side, sorted ascending by zIndex so higher layers render on top. */
  const elements = $derived(
    (side === 'front' ? idStore.front.elements : idStore.back.elements)
      .filter(el => el.show)
      .slice()
      .sort((a, b) => a.zIndex - b.zIndex)
  );

  const theme = $derived(side === 'front' ? idStore.front.theme : idStore.back.theme);

  /** @type {(() => void) | null} */
  let activeDragCleanup = null;
  // Called on unmount — removes any stale window listeners from an in-progress drag.
  onDestroy(() => activeDragCleanup?.());

  /** @type {HTMLDivElement | undefined} */
  let cardEl = $state(undefined);

  /** Convert screen-space pixel delta to card-space millimetres. */
  function pxToMm() {
    if (!cardEl) return 1;
    return 85.6 / cardEl.getBoundingClientRect().width;
  }

  /**
   * Begin drag-to-move for an element.
   * @param {PointerEvent} e
   * @param {string} elId
   */
  function startDrag(e, elId) {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();

    const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
    const el = els.find(x => x.id === elId);
    if (!el) return;

    onSelect(elId);
    const scale = pxToMm();
    const startX = e.clientX, startY = e.clientY;
    const ix = el.x, iy = el.y;
    let hasMoved = false;

    /** @param {PointerEvent} mv */
    function onMove(mv) {
      const dx = (mv.clientX - startX) * scale;
      const dy = (mv.clientY - startY) * scale;
      if (Math.abs(dx) > 0.5 || Math.abs(dy) > 0.5) hasMoved = true;
      el.x = Math.max(0, Math.min(85.6 - el.width,  Math.round((ix + dx) * 10) / 10));
      el.y = Math.max(0, Math.min(53.98 - el.height, Math.round((iy + dy) * 10) / 10));
    }
    function onUp() {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup',   onUp);
      activeDragCleanup = null;
      // Photo tap (no movement): open webcam
      if (!hasMoved && el.type === 'photo' && student) onWebcam(student);
    }
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup',   onUp);
    activeDragCleanup = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup',   onUp);
    };
  }

  /**
   * Begin resize from a corner handle.
   * @param {PointerEvent} e
   * @param {string} elId
   * @param {'nw'|'ne'|'sw'|'se'} corner
   */
  function startResize(e, elId, corner) {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();

    const els = side === 'front' ? idStore.front.elements : idStore.back.elements;
    const el = els.find(x => x.id === elId);
    if (!el) return;

    const scale = pxToMm();
    const startX = e.clientX, startY = e.clientY;
    const ix = el.x, iy = el.y, iw = el.width, ih = el.height;
    const aspectRatio = iw / ih;

    /** @param {PointerEvent} mv */
    function onMove(mv) {
      let dx = (mv.clientX - startX) * scale;
      let dy = (mv.clientY - startY) * scale;

      if (corner === 'se') {
        const newW = Math.max(5, iw + dx);
        el.width  = newW;
        el.height = el.proportional ? newW / aspectRatio : Math.max(3, ih + dy);
      } else if (corner === 'sw') {
        const newW = Math.max(5, iw - dx);
        el.x      = Math.max(0, ix + (iw - newW));
        el.width  = newW;
        el.height = el.proportional ? newW / aspectRatio : Math.max(3, ih + dy);
      } else if (corner === 'ne') {
        const newW = Math.max(5, iw + dx);
        const newH = Math.max(3, ih - dy);
        el.y      = Math.max(0, iy + (ih - newH));
        el.width  = newW;
        el.height = el.proportional ? newW / aspectRatio : newH;
      } else { // nw
        const newW = Math.max(5, iw - dx);
        const newH = el.proportional ? newW / aspectRatio : Math.max(3, ih - dy);
        el.x      = Math.max(0, ix + (iw - newW));
        el.y      = Math.max(0, iy + (ih - newH));
        el.width  = newW;
        el.height = newH;
      }
    }
    function onUp() {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup',   onUp);
      activeDragCleanup = null;
    }
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup',   onUp);
    activeDragCleanup = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup',   onUp);
    };
  }

  /** Deselect on clicking empty canvas background. */
  function onCanvasClick() { onSelect(null); }
</script>

<div
  class="flex-1 flex flex-col items-center justify-center overflow-hidden bg-slate-100 border border-dashed border-slate-200 rounded-3xl min-h-[480px] relative p-6"
  role="presentation"
  onpointerdown={onCanvasClick}
>
  <div style="transform: scale({zoom / 100}); transform-origin: center; transition: transform 0.05s ease-out;" class="shrink-0">
    <div
      bind:this={cardEl}
      class="card-container shadow-2xl relative border border-slate-200 rounded-[8px] overflow-visible select-none"
      style="width: 85.6mm; height: 53.98mm; background: white;"
    >
      <div class="w-full h-full relative rounded-[8px] overflow-hidden {theme}">
        {#each elements as el (el.id)}
          {@render canvasElement(el)}
        {/each}
      </div>
    </div>
  </div>

  <span class="absolute bottom-4 left-1/2 -translate-x-1/2 text-[9px] uppercase tracking-wider text-slate-400 font-bold pointer-events-none">
    {side === 'front' ? 'Vorderseite' : 'Rückseite'} · Drag &amp; Drop zum Verschieben · Ecken zum Skalieren
  </span>
</div>

{#snippet canvasElement(el)}
  {@const isSelected = selectedId === el.id}
  {@const isText = ['header','address','name','details','validity','text'].includes(el.type)}
  {@const isImage = el.type === 'image' || el.type === 'logo'}
  {@const isPhoto = el.type === 'photo'}
  {@const isBarcode = el.type === 'barcode'}

  <div
    role="presentation"
    onpointerdown={(e) => startDrag(e, el.id)}
    style="
      position: absolute;
      left: {el.x}mm; top: {el.y}mm;
      width: {el.width}mm; height: {el.height}mm;
      z-index: {el.zIndex};
      cursor: move;
    "
    class="{isSelected ? 'ring-2 ring-blue-500 ring-offset-0' : 'hover:ring-1 hover:ring-slate-400 hover:ring-dashed'} rounded-xs"
  >
    {#if isText}
      <div
        class="w-full h-full overflow-hidden leading-tight whitespace-pre-wrap"
        style="
          font-size: {el.style?.fontSize ?? 7}pt;
          color: {el.style?.color ?? 'inherit'};
          font-weight: {el.style?.fontWeight ?? 'normal'};
          text-align: {el.style?.textAlign ?? 'left'};
          font-family: {el.style?.fontFamily ?? 'inherit'};
        "
      >
        {#if el.type === 'name'}
          {student ? `${student.vorname} ${student.nachname}` : 'Maximilian Schmidt'}
        {:else if el.type === 'details'}
          {student ? `Klasse: ${student.klasse}` : 'Klasse: 9a'}
        {:else if el.type === 'validity'}
          {`Gültig bis: 31.07.${student?.abgaenger_jahr ?? '–'}`}
        {:else}
          {el.content}
        {/if}
      </div>
    {:else if isImage}
      <div class="w-full h-full border border-dashed border-slate-300 bg-slate-50/50 flex items-center justify-center overflow-hidden rounded-xs">
        {#if el.content}
          <img src={el.content} class="w-full h-full object-contain pointer-events-none" alt="Bild" />
        {:else}
          <span class="text-[5px] text-slate-400 font-bold pointer-events-none">{el.type === 'logo' ? 'LOGO' : 'BILD'}</span>
        {/if}
      </div>
    {:else if isPhoto}
      <div class="w-full h-full border border-dashed border-slate-300 bg-slate-50 flex items-center justify-center overflow-hidden rounded-sm group">
        {#if student && student.foto_url}
          <img src="{student.foto_url}?t={timestamp}" onerror={(e) => { /** @type {any} */ (e.currentTarget).style.display='none'; }} class="w-full h-full object-cover pointer-events-none" alt="Passbild" />
        {/if}
        <div class="absolute inset-0 bg-slate-900/50 text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none text-[5px] font-bold tracking-wider uppercase">📸 Ändern</div>
      </div>
    {:else if isBarcode}
      <div class="w-full h-full flex flex-col items-center justify-center">
        {#if student}
          <img
            src="/api/barcode?content={student.barcode_id}&qr={barcodeType === 'qr'}&width={barcodeType === 'qr' ? 80 : 200}&height={barcodeType === 'qr' ? 80 : 50}"
            class="max-w-full max-h-full object-contain pointer-events-none"
            alt="Barcode"
          />
          <span class="font-bold text-[6.5pt] tracking-widest text-slate-700 pointer-events-none">{student.barcode_id}</span>
        {:else}
          <div class="text-[5px] text-slate-400 font-bold">BARCODE</div>
        {/if}
      </div>
    {/if}

    {#if isSelected}
      {@render resizeHandle(el, 'nw', 'top-0 left-0 cursor-nw-resize -translate-x-1/2 -translate-y-1/2')}
      {@render resizeHandle(el, 'ne', 'top-0 right-0 cursor-ne-resize translate-x-1/2 -translate-y-1/2')}
      {@render resizeHandle(el, 'sw', 'bottom-0 left-0 cursor-sw-resize -translate-x-1/2 translate-y-1/2')}
      {@render resizeHandle(el, 'se', 'bottom-0 right-0 cursor-se-resize translate-x-1/2 translate-y-1/2')}
    {/if}
  </div>
{/snippet}

{#snippet resizeHandle(el, corner, posClass)}
  <div
    role="presentation"
    onpointerdown={(e) => startResize(e, el.id, corner)}
    class="absolute w-3 h-3 bg-white border-2 border-blue-500 rounded-full z-50 {posClass}"
  ></div>
{/snippet}
