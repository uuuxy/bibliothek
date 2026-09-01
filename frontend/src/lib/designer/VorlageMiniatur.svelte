<!-- @component VorlageMiniatur — die Vorderseite einer Design-Vorlage in Briefmarkengröße
     für die Vorlagen-Galerie. Zeichnet die VORLAGENDATEN (vorlage(kennung)), nicht den
     Store: Die Galerie zeigt, was man bekäme, nicht was man hat.

     Bewusst kein CardFace: Das liest den Store und rendert echte Barcode-Bilder vom
     Server — sieben Miniaturen hießen sieben Barcode-Requests bei jedem Öffnen. Hier
     genügt die Gestalt: Flächen in Farbe, Texte mit Musterinhalt, Foto/Logo als graue
     Blöcke, der Barcode als Streifenmuster. Die Karte wird in echten mm gezeichnet und
     per transform verkleinert, damit die Proportionen exakt die der Leinwand sind. -->
<script>
	import { vorlage } from './ausweisVorlagen.js';

	/** @type {{ kennung: string, massstab?: number }} */
	let { kennung, massstab = 0.32 } = $props();

	const BREITE = 85.6;
	const HOEHE = 53.98;

	const seite = $derived(vorlage(kennung)?.front);
	const elements = $derived(
		(seite?.elements ?? [])
			.slice()
			.sort((/** @type {any} */ a, /** @type {any} */ b) => a.zIndex - b.zIndex)
	);

	/** @param {any} el */
	function textVon(el) {
		if (el.type === 'name') return 'Max Mustermann';
		if (el.type === 'validity') return 'Gültig bis: 31.07.2027';
		return el.content;
	}

	const BARCODE_MUSTER =
		'repeating-linear-gradient(90deg, #1f1f1f 0 0.3mm, transparent 0.3mm 0.7mm, #1f1f1f 0.7mm 1.3mm, transparent 1.3mm 1.6mm)';
</script>

<div
	class="overflow-hidden rounded-xs"
	style="width: {BREITE * massstab}mm; height: {HOEHE * massstab}mm;"
>
	<div
		class="relative origin-top-left overflow-hidden rounded-sm {seite?.theme ?? ''}"
		style="width: {BREITE}mm; height: {HOEHE}mm; transform: scale({massstab});"
	>
		{#each elements as el (el.id)}
			<div
				class="absolute"
				data-miniatur-element
				style="left: {el.x}mm; top: {el.y}mm; width: {el.width}mm; height: {el.height}mm; z-index: {el.zIndex};"
			>
				{#if el.type === 'box'}
					<div
						class="h-full w-full"
						style="background-color: {el.style?.color}; border-radius: {el.style?.radius ?? 0}mm;"
					></div>
				{:else if el.type === 'barcode'}
					<div class="mx-auto h-[8mm] w-[26mm]" style="background: {BARCODE_MUSTER};"></div>
				{:else if el.type === 'image' && el.content}
					<img src={el.content} class="h-full w-full object-contain" alt="" />
				{:else if ['photo', 'logo', 'image'].includes(el.type)}
					<div class="h-full w-full rounded-xs" style="background-color: rgb(0 0 0 / 0.12);"></div>
				{:else}
					<div
						class="h-full w-full overflow-hidden leading-tight whitespace-pre-wrap"
						style="font-size: {el.style?.fontSize ?? 7}pt; color: {el.style?.color ??
							'inherit'}; font-weight: {el.style?.fontWeight ?? 'normal'}; text-align: {el.style
							?.textAlign ?? 'left'};"
					>
						{textVon(el)}
					</div>
				{/if}
			</div>
		{/each}
	</div>
</div>
