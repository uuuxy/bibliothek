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

	/**
	 * `platzhalter` zeichnet leere Bild-, Logo- und Passbildfelder als gestrichelten
	 * Rahmen statt sie wegzulassen. NUR fuer den Testdruck des Ausweis-Designers.
	 *
	 * Der Testdruck ist da, um das Layout auf dem echten Kartendrucker zu pruefen — und
	 * liess bis zum 24.08.2026 ausgerechnet die zwei flaechenkritischen Felder weg: Der
	 * Platzhalter-Schueler hat kein Foto, und solange kein Logo hochgeladen ist, hat auch
	 * das Logofeld keinen Inhalt. Auf der Leinwand standen dort gestrichelte Rahmen, auf
	 * dem Papier nichts. Beim echten Schueler stimmte es, weil der ein Foto hat — genau
	 * der gemeldete Unterschied "Testdruck falsch, Schuelerprofil tadellos".
	 *
	 * Vorgabe false: Ein ECHTER Ausweis darf niemals einen Rahmen mit der Aufschrift
	 * "PASSBILD" tragen, nur weil beim Schueler kein Foto hinterlegt ist.
	 *
	 * @type {{ side: 'front'|'back', student: any, barcodeType: 'code39'|'qr', timestamp?: number, platzhalter?: boolean }}
	 */
	const { side, student, barcodeType, timestamp = 0, platzhalter = false } = $props();

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
		!isBarcode && ['header', 'address', 'name', 'validity', 'text'].includes(el.type)}
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
			{:else if el.type === 'validity' && student}
				Gültig bis: 31.07.{student.ausweis_gueltig_bis ?? '–'}
			{:else}
				{el.content}
			{/if}
		</div>
	{:else if isImage && (el.content || platzhalter)}
		<div
			class="absolute overflow-hidden flex items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			{#if el.content}
				<img src={el.content} class="w-full h-full object-contain" alt="Bild" />
			{:else}
				{@render leerstelle(el.type === 'logo' ? 'LOGO' : 'BILD')}
			{/if}
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
			{:else if platzhalter}
				{@render leerstelle('PASSBILD')}
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

{#snippet leerstelle(/** @type {string} */ beschriftung)}
	<span
		class="border-outline text-on-surface-variant flex h-full w-full items-center justify-center rounded-xs border border-dashed text-[5px] font-bold tracking-wider"
	>
		{beschriftung}
	</span>
{/snippet}
