<script>
	/**
	 * @file CanvasElement.svelte
	 * EIN Element auf dem Reißbrett des Ausweis-Designers: Darstellung je Elementart
	 * plus Auswahl-Ring und Skalier-Griffe. Aus CanvasArea.svelte herausgelöst
	 * (01.09.2026), als das Flächen-Element ('box') dazukam — CanvasArea stand im
	 * eingefrorenen Größen-Bestand, und die Element-Darstellung ist der Teil, der mit
	 * jeder Elementart weiterwächst. Drag-/Resize-MECHANIK bleibt drüben: Die
	 * Fensterlistener und ihr Aufräumen gehören an eine Stelle.
	 */
	import { User } from '@lucide/svelte';

	/**
	 * @type {{
	 *   el: any, isSelected: boolean, student: any, barcodeType: string,
	 *   onStartDrag: (e: PointerEvent, id: string) => void,
	 *   onStartResize: (e: PointerEvent, id: string, corner: 'nw'|'ne'|'sw'|'se') => void
	 * }}
	 */
	const { el, isSelected, student, barcodeType, onStartDrag, onStartResize } = $props();

	const isBox = $derived(el.type === 'box');
	const isBarcode = $derived(
		el.type === 'barcode' || (typeof el.content === 'string' && el.content.includes('{{barcode}}'))
	);
	const isText = $derived(
		!isBarcode && ['header', 'address', 'name', 'validity', 'text'].includes(el.type)
	);
	const isImage = $derived(!isBarcode && (el.type === 'image' || el.type === 'logo'));
	const isPhoto = $derived(!isBarcode && el.type === 'photo');
</script>

<div
	role="presentation"
	onpointerdown={(e) => onStartDrag(e, el.id)}
	style="
      position: absolute;
      left: {el.x}mm; top: {el.y}mm;
      width: {el.width}mm; height: {el.height}mm;
      z-index: {el.zIndex};
      cursor: move;
    "
	class="{isSelected
		? 'ring-2 ring-blue-500 ring-offset-0'
		: 'hover:ring-1 hover:ring-slate-400 hover:ring-dashed'} rounded-xs"
>
	{#if isBox}
		<!-- Farbfläche: füllt ihre Box exakt (kein object-contain, kein Bild-Chrome) —
		     die SVG-Band-Bilder davor ließen am Kartenrand weiße Streifen stehen. -->
		<div
			class="w-full h-full"
			style="background-color: {el.style?.color ?? '#000000'}; border-radius: {el.style?.radius ??
				0}mm;"
		></div>
	{:else if isText}
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
				{student ? `${student.vorname} ${student.nachname}` : 'Max Mustermann'}
			{:else if el.type === 'validity'}
				{`Gültig bis: 31.07.${student?.ausweis_gueltig_bis ?? '–'}`}
			{:else}
				{el.content}
			{/if}
		</div>
	{:else if isImage}
		<div
			class="w-full h-full border border-dashed border-slate-300 bg-slate-50/50 flex items-center justify-center overflow-hidden rounded-xs"
		>
			{#if el.content}
				<img src={el.content} class="w-full h-full object-contain pointer-events-none" alt="Bild" />
			{:else}
				<span class="text-[5px] text-slate-400 font-bold pointer-events-none"
					>{el.type === 'logo' ? 'LOGO' : 'BILD'}</span
				>
			{/if}
		</div>
	{:else if isPhoto}
		<div
			class="w-full h-full border border-dashed border-slate-300 bg-slate-50 flex flex-col items-center justify-center overflow-hidden rounded-sm text-slate-400"
		>
			<User
				class="w-1/2 h-1/2 max-h-12 max-w-12 mb-1 opacity-40 pointer-events-none"
				aria-hidden="true"
			/>
			<span class="text-[5px] font-medium pointer-events-none">PASSBILD</span>
		</div>
	{:else if isBarcode}
		<div class="w-full h-full flex flex-col items-center justify-center">
			{#if student}
				<img
					src="/api/barcode?content={student.barcode_id}&qr={barcodeType ===
						'qr'}&width={barcodeType === 'qr' ? 80 : 200}&height={barcodeType === 'qr' ? 80 : 50}"
					class="max-w-full max-h-full object-contain pointer-events-none"
					alt="Barcode"
				/>
				<span class="font-bold text-[6.5pt] tracking-widest text-slate-700 pointer-events-none"
					>{student.barcode_id}</span
				>
			{:else}
				<div class="text-[5px] text-slate-400 font-bold">BARCODE</div>
			{/if}
		</div>
	{/if}

	{#if isSelected}
		{@render resizeHandle('nw', 'top-0 left-0 cursor-nw-resize -translate-x-1/2 -translate-y-1/2')}
		{@render resizeHandle('ne', 'top-0 right-0 cursor-ne-resize translate-x-1/2 -translate-y-1/2')}
		{@render resizeHandle(
			'sw',
			'bottom-0 left-0 cursor-sw-resize -translate-x-1/2 translate-y-1/2'
		)}
		{@render resizeHandle(
			'se',
			'bottom-0 right-0 cursor-se-resize translate-x-1/2 translate-y-1/2'
		)}
	{/if}
</div>

{#snippet resizeHandle(/** @type {'nw'|'ne'|'sw'|'se'} */ corner, /** @type {string} */ posClass)}
	<div
		role="presentation"
		onpointerdown={(e) => onStartResize(e, el.id, corner)}
		class="absolute w-3 h-3 bg-white border-2 border-blue-500 rounded-full z-50 {posClass}"
	></div>
{/snippet}
