<script>
	/**
	 * @file PrintPreview.svelte
	 * Hidden print-output sections rendered by the browser's print engine.
	 *
	 * Runes in use:
	 *   $props()  — receives `students`, `barcodeType`, `timestamp`
	 *
	 * Print architecture:
	 *   Four hidden <div> containers are placed in the DOM. The triggerPrint()
	 *   function in the coordinator sets `data-print-mode` and `data-print-side`
	 *   on <body>, and CSS rules in this file (plus app.css) selectively show the
	 *   correct container for the active mode/side combination.
	 *
	 *   ┌──────────────────────────────────────────────────────────────┐
	 *   │ print-section-card      → card printer, front side           │
	 *   │ print-section-a4        → A4 sheet,     front side           │
	 *   │ print-section-back-card → card printer, back  side           │
	 *   │ print-section-back-a4   → A4 sheet,     back  side           │
	 *   └──────────────────────────────────────────────────────────────┘
	 *
	 * Each section iterates over `students` and renders one card per student
	 * using the `{@snippet}` helpers below.
	 */
	import { idStore } from './idDesignerStore.svelte.js';

	/** @type {{ students: any[], barcodeType: 'code39'|'qr', timestamp: number }} */
	const { students, barcodeType, timestamp } = $props();

	/** Elements visible on the front side, sorted by zIndex ascending. */
	const frontEls = $derived(
		idStore.front.elements
			.filter((e) => e.show)
			.slice()
			.sort((a, b) => a.zIndex - b.zIndex)
	);
	/** Elements visible on the back side, sorted by zIndex ascending. */
	const backEls = $derived(
		idStore.back.elements
			.filter((e) => e.show)
			.slice()
			.sort((a, b) => a.zIndex - b.zIndex)
	);
</script>

<!-- Card printer: front -->
<div class="print-rendered-output print-section-card hidden print:block">
	{#each students as student (student.id)}
		<div class="print-card-box {idStore.front.theme}">
			{@render printFrontCard(student)}
		</div>
	{/each}
</div>

<!-- A4 sheet: front -->
<div class="print-rendered-output print-section-a4 a4_sheet hidden print:block">
	<div class="print-cards-grid">
		{#each students as student (student.id)}
			<div class="print-card-box {idStore.front.theme}">
				{@render printFrontCard(student)}
			</div>
		{/each}
	</div>
</div>

<!-- Card printer: back -->
<div class="print-rendered-output print-section-back-card hidden">
	{#each students as _student (_student.id)}
		<div class="print-card-box {idStore.back.theme}">
			{@render printBackCard()}
		</div>
	{/each}
</div>

<!-- A4 sheet: back -->
<div class="print-rendered-output print-section-back-a4 a4_sheet hidden">
	<div class="print-cards-grid">
		{#each students as _student (_student.id)}
			<div class="print-card-box {idStore.back.theme}">
				{@render printBackCard()}
			</div>
		{/each}
	</div>
</div>

{#snippet printFrontCard(student)}
	{#each frontEls as el (el.id)}
		{@render printElement(el, student)}
	{/each}
{/snippet}

{#snippet printBackCard()}
	{#each backEls as el (el.id)}
		{@render printElement(el, null)}
	{/each}
{/snippet}

{#snippet printElement(el, student)}
	{@const isBarcode =
		el.type === 'barcode' || (typeof el.content === 'string' && el.content.includes('{{barcode}}'))}
	{@const isText =
		!isBarcode && ['header', 'address', 'name', 'details', 'validity', 'text'].includes(el.type)}
	{@const isImage = !isBarcode && (el.type === 'image' || el.type === 'logo')}
	{@const isPhoto = !isBarcode && el.type === 'photo'}

	{#if isText}
		<div
			class="absolute leading-tight whitespace-pre-wrap overflow-hidden"
			style="
        left: {el.x}mm; top: {el.y}mm;
        width: {el.width}mm; height: {el.height}mm;
        font-size: {el.style?.fontSize ?? 7}pt;
        color: {el.style?.color ?? 'black'};
        font-weight: {el.style?.fontWeight ?? 'normal'};
        text-align: {el.style?.textAlign ?? 'left'};
        font-family: {el.style?.fontFamily ?? 'inherit'};
        z-index: {el.zIndex};
      "
		>
			{#if el.type === 'name' && student}
				{student.vorname} {student.nachname}
			{:else if el.type === 'details' && student}
				Klasse: {student.klasse}
			{:else if el.type === 'validity' && student}
				Gültig bis: 31.07.{student.abgaenger_jahr ?? '–'}
			{:else}
				{el.content}
			{/if}
		</div>
	{:else if isImage && el.content}
		<div
			class="absolute overflow-hidden flex items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			<img src={el.content} class="w-full h-full object-contain" alt="Bild" />
		</div>
	{:else if isPhoto && student}
		<div
			class="absolute overflow-hidden flex items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			{#if student.foto_url}
				<img
					src="{student.foto_url}?t={timestamp}"
					onerror={(e) => {
						/** @type {any} */ (e.currentTarget).style.display = 'none';
					}}
					class="w-full h-full object-cover"
					alt="Passbild"
				/>
			{/if}
		</div>
	{:else if isBarcode && student}
		<div
			class="absolute flex flex-col items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			<img
				src="/api/barcode?content={student.barcode_id}&qr={barcodeType ===
					'qr'}&width={barcodeType === 'qr' ? 80 : 200}&height={barcodeType === 'qr' ? 80 : 50}"
				class="{barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain"
				alt="Barcode"
			/>
			<span class="font-bold mt-0.5 text-[6.5pt] tracking-widest text-zinc-800"
				>{student.barcode_id}</span
			>
		</div>
	{/if}
{/snippet}

<style>
	@media print {
		:global(html, body) {
			margin: 0 !important;
			padding: 0 !important;
			background: white !important;
			overflow: hidden !important;
		}
		:global(main, .min-h-screen, .flex) {
			margin: 0 !important;
			padding: 0 !important;
			display: block !important;
			background: white !important;
			border: none !important;
			box-shadow: none !important;
		}
		:global(.no-print) {
			display: none !important;
		}
		:global(body[data-print-mode='card']) .print-section-card {
			display: block !important;
		}
		:global(body[data-print-mode='card'][data-print-side='back']) .print-section-card {
			display: none !important;
		}
		:global(body[data-print-mode='a4'][data-print-side='back']) .print-section-a4 {
			display: none !important;
		}
		:global(body[data-print-mode='card'][data-print-side='back']) .print-section-back-card {
			display: block !important;
		}
		:global(body[data-print-mode='a4'][data-print-side='back']) .print-section-back-a4 {
			display: block !important;
		}
	}
</style>
