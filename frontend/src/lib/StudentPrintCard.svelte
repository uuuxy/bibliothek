<script>
  import { idStore } from "./idLayoutStore.svelte.js";

  /** @type {{ profile: any, timestamp: number }} */
  let { profile, timestamp } = $props();
</script>

<!--
  Single-card print section.
  Hidden on screen (display:none inline), shown via @media print when
  body[data-print-mode="card-single"] is set by printCard().
  Rendered outside the .no-print wrapper so it survives print suppression.
  Uses idStore so it always reflects the latest Ausweis-Designer settings.
-->
<div class="single-card-print-section" style="display:none" aria-hidden="true">
  <div class="print-card-box {idStore.cardTheme}">
    {#if idStore.layout?.header?.show}
      <div class="absolute font-black text-center tracking-tight leading-none truncate text-black"
        style="left: {idStore.layout.header.x}mm; top: {idStore.layout.header.y}mm; transform: scale({idStore.layout.header.scale}); transform-origin: top left; font-size: 7.5pt; width: {85.6 - idStore.layout.header.x * 2}mm;">
        {idStore.layout.header.text}
      </div>
    {/if}
    {#if idStore.layout?.logo?.show && idStore.layout.logo.url}
      <div class="absolute overflow-hidden flex items-center justify-center"
        style="left: {idStore.layout.logo.x}mm; top: {idStore.layout.logo.y}mm; width: {12 * idStore.layout.logo.scale}mm; height: {12 * idStore.layout.logo.scale}mm;">
        <img src={idStore.layout.logo.url} class="w-full h-full object-contain" alt="Logo" />
      </div>
    {/if}
    {#if idStore.layout?.address?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.address.x}mm; top: {idStore.layout.address.y}mm; transform: scale({idStore.layout.address.scale}); transform-origin: top left; font-size: 6.5pt; width: {85.6 - idStore.layout.address.x - 2}mm; max-height: 12mm; overflow: hidden;">
        {idStore.layout.address.text}
      </div>
    {/if}
    {#if idStore.layout?.photo?.show}
      <div class="absolute border border-solid border-zinc-300 bg-zinc-50 flex items-center justify-center overflow-hidden rounded-sm"
        style="left: {idStore.layout.photo.x}mm; top: {idStore.layout.photo.y}mm; width: {22 * idStore.layout.photo.scale}mm; height: {28 * idStore.layout.photo.scale}mm;">
        <img src="/api/schueler/{profile.barcode_id}/photo?t={timestamp}"
          onerror={(e) => { (/** @type {any} */ (e.currentTarget)).style.display = 'none'; }}
          class="w-full h-full object-cover" alt="Passbild" />
      </div>
    {/if}
    {#if idStore.layout?.name?.show}
      <div class="absolute font-extrabold tracking-tight leading-none text-black"
        style="left: {idStore.layout.name.x}mm; top: {idStore.layout.name.y}mm; transform: scale({idStore.layout.name.scale}); transform-origin: top left; font-size: 9pt;">
        {profile.vorname} {profile.nachname}
      </div>
    {/if}
    {#if idStore.layout?.details?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.details.x}mm; top: {idStore.layout.details.y}mm; transform: scale({idStore.layout.details.scale}); transform-origin: top left; font-size: 7.5pt;">
        Klasse: {profile.klasse}
      </div>
    {/if}
    {#if idStore.layout?.validity?.show}
      <div class="absolute font-semibold tracking-tight opacity-75 leading-none text-zinc-800"
        style="left: {idStore.layout.validity.x}mm; top: {idStore.layout.validity.y}mm; transform: scale({idStore.layout.validity.scale}); transform-origin: top left; font-size: 7pt;">
        Gültig bis: 31.07.{profile.abgaenger_jahr ?? '–'}
      </div>
    {/if}
    {#if idStore.layout?.barcode?.show}
      <div class="absolute flex flex-col items-center leading-none"
        style="left: {idStore.layout.barcode.x}mm; top: {idStore.layout.barcode.y}mm; transform: scale({idStore.layout.barcode.scale}); transform-origin: top left;">
        <img src="/api/barcode?content={profile.barcode_id}&qr={idStore.barcodeType === 'qr'}&width={idStore.barcodeType === 'qr' ? 80 : 200}&height={idStore.barcodeType === 'qr' ? 80 : 50}"
          class="{idStore.barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain" alt="Barcode" />
        <span class="font-bold mt-1 text-[6.5pt] tracking-widest text-zinc-800">{profile.barcode_id}</span>
      </div>
    {/if}
  </div>
</div>
