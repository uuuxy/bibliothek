<script>
	/**
	 * @file CardFace.svelte
	 * Rendert EINE Ausweisseite (front/back) aus dem zentralen Element-Modell für einen
	 * konkreten Schüler. Dies ist die EINZIGE Render-Quelle sowohl für den Batch-Druck
	 * (PrintPreview, DruckCenter) als auch den Einzeldruck (StudentPrintCard, Profil) —
	 * so fällt an jedem Klick-Pfad exakt derselbe Ausweis heraus (Single Source of Truth).
	 * Rein für Ausgabe/Vorschau, kein Editier-Chrome.
	 *
	 * `student` darf null sein (z. B. Rückseite ohne personenbezogene Elemente); dann
	 * werden nur statische Elemente (Header/Adresse/Text/Bild) gerendert.
	 */
	import { idStore } from './idDesignerStore.svelte.js';

	/** @type {{ side: 'front'|'back', student: any, barcodeType: 'code39'|'qr', timestamp?: number }} */
	const { side, student, barcodeType, timestamp = 0 } = $props();

	/** Sichtbare Elemente der Seite, aufsteigend nach zIndex (höhere Ebenen zuletzt). */
	const elements = $derived(
		(side === 'front' ? idStore.front.elements : idStore.back.elements)
			.filter((/** @type {any} */ e) => e.show)
			.slice()
			.sort((/** @type {any} */ a, /** @type {any} */ b) => a.zIndex - b.zIndex)
	);
</script>

{#each elements as el (el.id)}
	{@render cardElement(el)}
{/each}

{#snippet cardElement(/** @type {any} */ el)}
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
				src="/api/barcode?content={student.barcode_id}&qr={barcodeType === 'qr'}&width={barcodeType ===
				'qr'
					? 80
					: 200}&height={barcodeType === 'qr' ? 80 : 50}"
				class="{barcodeType === 'qr' ? 'h-[11mm] w-[11mm]' : 'h-[8mm]'} object-contain"
				alt="Barcode"
			/>
			<span class="font-bold mt-0.5 text-[6.5pt] tracking-widest text-zinc-800">{student.barcode_id}</span
			>
		</div>
	{/if}
{/snippet}
