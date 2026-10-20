<script>
	import { strichcodeBildUrl, strichcodeBildOptionenFuerElement } from '../strichcodeBild.js';
	/**
	 * @file Kartenseite.svelte
	 * Rendert EINE Ausweisseite (front/back) aus dem zentralen Element-Modell für einen
	 * konkreten Leser — Schüler wie Kollegen. Dies ist die EINZIGE Render-Quelle sowohl für den Batch-Druck
	 * (Druckvorschau, DruckCenter) als auch den Einzeldruck (Ausweiskarte, Profil) —
	 * so fällt an jedem Klick-Pfad exakt derselbe Ausweis heraus (Single Source of Truth).
	 * Rein für Ausgabe/Vorschau, kein Editier-Chrome.
	 *
	 * `student` darf null sein (z. B. Rückseite ohne personenbezogene Elemente); dann
	 * werden nur statische Elemente (Header/Adresse/Text/Bild) gerendert. Der Dokumenttyp
	 * gehört nicht dazu: Ohne Leser ist die Vorgabe „Schülerausweis".
	 */
	import { idStore } from './idDesignerStore.svelte.js';
	import { ausweisTitel, istKollegium } from '../leserArt.js';
	import { NUMMER_FARBE } from './kartenFarben.js';

	/**
	 * `platzhalter` zeichnet leere Bild-, Logo- und Passbildfelder als gestrichelten
	 * Rahmen statt sie wegzulassen, nur für den Testdruck des Ausweis-Designers.
	 *
	 * Der Testdruck ist da, um das Layout auf dem echten Kartendrucker zu prüfen, und darf
	 * die zwei flächenkritischen Felder nicht weglassen: Der Platzhalter-Schüler hat kein
	 * Foto, und solange kein Logo hochgeladen ist, hat auch das Logofeld keinen Inhalt. Auf
	 * der Leinwand stehen dort gestrichelte Rahmen; ohne sie stünde auf dem Papier nichts.
	 *
	 * Vorgabe false: Ein echter Ausweis darf niemals einen Rahmen mit der Aufschrift
	 * "PASSBILD" tragen, nur weil beim Schüler kein Foto hinterlegt ist.
	 *
	 * @type {{ side: 'front'|'back', leser: any, barcodeType: 'code39'|'qr', timestamp?: number, platzhalter?: boolean }}
	 */
	const { side, leser, barcodeType, timestamp = 0, platzhalter = false } = $props();

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
	{@const isBox = el.type === 'box'}
	{@const isBarcode =
		el.type === 'barcode' || (typeof el.content === 'string' && el.content.includes('{{barcode}}'))}
	{@const isText =
		!isBarcode &&
		['header', 'address', 'name', 'validity', 'dokumenttyp', 'text'].includes(el.type)}
	{@const isImage = !isBarcode && (el.type === 'image' || el.type === 'logo')}
	{@const isPhoto = !isBarcode && el.type === 'photo'}

	{#if isBox}
		<!-- Farbfläche (Kopf-/Fußband, Streifen): druckt als Hintergrundfarbe — die
		     Kartendruck-CSS setzt print-color-adjust: exact, sonst fiele sie im Druck weg. -->
		<div
			class="absolute"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex}; background-color: {el
				.style?.color ?? '#000000'}; border-radius: {el.style?.radius ?? 0}mm;"
		></div>
	{:else if isText}
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
        text-transform: {el.style?.textTransform ?? 'none'};
        z-index: {el.zIndex};
      "
		>
			{#if el.type === 'name' && leser}
				{leser.vorname} {leser.nachname}
			{:else if el.type === 'dokumenttyp'}
				{ausweisTitel(leser?.art)}
			{:else if el.type === 'validity' && leser}
				<!-- Ein Kollege hat keine Klasse, aus der ein Ablaufjahr zu rechnen wäre, und sein
				     Ausweis läuft mit keinem Schuljahr ab. Ohne diese Weiche stünde auf seiner
				     Karte „Gültig bis: 31.07.–". -->
				{#if !istKollegium(leser)}
					Gültig bis: 31.07.{leser.ausweis_gueltig_bis ?? '–'}
				{/if}
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
	{:else if isPhoto && leser}
		<div
			class="absolute overflow-hidden flex items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			{#if leser.foto_url}
				<img
					src="{leser.foto_url}?t={timestamp}"
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
	{:else if isBarcode && leser}
		<div
			class="absolute flex flex-col items-center justify-center"
			style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
		>
			<!-- Größe aus dem Element, wie auf der Leinwand (CanvasElement): Mit einer festen
			     Höhe wäre ein im Designer höher gezogener Strichcode auf dem Papier nicht höher. -->
			<img
				src={strichcodeBildUrl(
					leser.barcode_id,
					strichcodeBildOptionenFuerElement(el, barcodeType)
				)}
				class="max-w-full max-h-full object-contain"
				alt="Barcode"
			/>
			<span class="font-bold mt-0.5 text-[6.5pt] tracking-widest" style="color: {NUMMER_FARBE};"
				>{leser.barcode_id}</span
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
