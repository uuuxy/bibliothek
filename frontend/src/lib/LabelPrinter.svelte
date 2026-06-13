<script>
  import { onMount } from "svelte";
  import { labelStore } from "./stores/labels.svelte.js";
  import LabelSettings from "./components/labels/LabelSettings.svelte";
  import LabelPreview from "./components/labels/LabelPreview.svelte";

  onMount(() => {
    labelStore.loadClassGroups();
  });
</script>

<div class="w-full space-y-6 no-print text-slate-800 animate-fade-in">
  
  <!-- Header Info -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-end gap-4 border-b border-slate-100 pb-5">
    <button onclick={labelStore.triggerPrint} disabled={labelStore.finalLabels.filter(lbl => !lbl.isBlank).length === 0} class="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 disabled:bg-slate-200 disabled:text-slate-400 disabled:cursor-not-allowed text-white font-bold transition-all flex items-center gap-2 shadow-xs cursor-pointer">
      <span>🖨️ A4-Bogen drucken</span>
    </button>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
    <LabelSettings />
    <LabelPreview />
  </div>
</div>

<!-- Print Output (Invisible on screen, visible on print) -->
<div class="print-rendered-output a4_grid">
  <div class="print-labels-grid">
    {#each labelStore.finalLabels as lbl}
      {#if lbl.isBlank}
        <!-- Hidden blank box keeping grid positions correct -->
        <div class="print-label-box border-none opacity-0"></div>
      {:else}
        <div class="print-label-box {labelStore.labelBorder ? 'border border-black' : ''} p-2 text-black bg-white flex flex-col justify-between">
          <div class="font-extrabold text-zinc-950 title-clamp leading-tight text-[8pt]">{lbl.titel}</div>
          <div class="text-zinc-700 font-medium author-clamp leading-tight text-[7pt] mt-0.5">{lbl.autor || 'Unbekannt'}</div>
          <div class="flex flex-col items-center justify-center grow pt-1">
            <img src="/api/barcode?content={lbl.barcode_id}&qr={labelStore.barcodeType === 'qr'}&width=220&height=70" class="{labelStore.barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[7mm]'} object-contain" alt="Barcode" />
            <span class="font-bold mt-1 text-[6.5pt] tracking-widest text-zinc-800">{lbl.barcode_id}</span>
          </div>
        </div>
      {/if}
    {/each}
  </div>
</div>

<style>
  /* 
    LINE-CLAMPING LOGIK FÜR EXTREM LANGE BUCHTITEL & AUTOREN:
    - title-clamp: Begrenzt lange Buchtitel auf maximal 2 Zeilen.
      Schneidet den Text mit '...' ab, um den Barcode/QR-Code nicht zu verschieben.
    - author-clamp: Begrenzt Autorennamen auf maximal 1 Zeile.
  */
  .title-clamp {
    display: -webkit-box;
    -webkit-line-clamp: 2; /* Maximal 2 Zeilen anzeigen */
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    word-break: break-word;
  }

  .author-clamp {
    display: -webkit-box;
    -webkit-line-clamp: 1; /* Maximal 1 Zeile anzeigen */
    line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
    word-break: break-word;
  }
</style>
