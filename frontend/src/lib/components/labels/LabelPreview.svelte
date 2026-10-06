<script>
	import { strichcodeBildUrl } from '../../strichcodeBild.js';
	import { formatKurzname } from '../../etikettformate.js';
	import { labelStore } from '../../stores/labels.svelte.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';

	// Das Blatt ist in echten Millimetern gezeichnet, zwei Drittel von A4. Ist die Spalte
	// schmaler, wird es als Ganzes verkleinert (wie designer/VorlageMiniatur); die Hülle nimmt
	// die verkleinerte Größe ein, damit das Blatt nichts neben sich verdeckt.
	const BLATT_BREITE_MM = 140;
	const PX_JE_MM = 96 / 25.4;
	let platz = $state(0);
	let blattHoehe = $state(0);
	const massstab = $derived(platz > 0 ? Math.min(1, platz / (BLATT_BREITE_MM * PX_JE_MM)) : 1);
</script>

<!-- Die Vorschau bekommt die kleinere Hälfte (5 von 12): Die Arbeit geschieht im Formular
     daneben. Eine getönte Fläche ohne gestrichelten Rand, denn gestrichelt heißt in dieser
     Anwendung „hier gehört etwas hin", und die Vorschau ist die Ausgabe. Sie wächst mit
     ihrem Inhalt. -->
<div
	class="lg:col-span-5 flex flex-col items-center justify-start rounded-xl bg-surface-container p-6"
>
	<span class="text-xs text-on-surface-variant font-medium mb-4"
		>A4 Etiketten-Vorschau · {formatKurzname(labelStore.formatId)}</span
	>

	{#if !labelStore.selectedTitle && (printQueue.copies?.length ?? 0) === 0}
		<div class="grow flex flex-col items-center justify-center text-on-surface-variant py-12">
			<span>Kein Buch ausgewählt</span>
			<span class="mt-1 text-xs text-on-surface-variant"
				>Suche einen Titel links, um die Live-Vorschau zu aktivieren.</span
			>
		</div>
	{:else if labelStore.finalLabels.length === 0}
		<div class="grow flex flex-col items-center justify-center text-on-surface-variant py-12">
			<span>Keine Etiketten gewählt</span>
			<span class="mt-1 text-xs text-on-surface-variant"
				>Wähle mindestens ein Exemplar oder erhöhe die Menge.</span
			>
		</div>
	{:else}
		<!-- Die Zeile misst den Platz der Spalte, die Hülle darin ist so groß wie das verkleinerte
		     Blatt. -->
		<div class="flex w-full justify-center" bind:clientWidth={platz}>
			<div style="width: {BLATT_BREITE_MM * massstab}mm; height: {blattHoehe * massstab}px;">
				<div
					bind:clientHeight={blattHoehe}
					data-testid="etiketten-blatt"
					class="bg-surface-container-lowest border border-outline-variant shadow-2xl relative flex origin-top-left flex-col items-start select-none"
					style="width: {BLATT_BREITE_MM}mm; min-height: 198mm; padding: 10.1mm 4.8mm 0; box-sizing: border-box; transform: scale({massstab});"
				>
					<div
						style="display: grid; grid-template-columns: repeat(3, 42.3mm); column-gap: 1.7mm; row-gap: 0; width: 100%;"
					>
						{#each labelStore.finalLabels as lbl, _i (_i)}
							{#if lbl.isBlank}
								<!-- Blank Label placeholder representation -->
								<div
									class="border border-dashed border-outline-variant bg-surface flex items-center justify-center"
									style="width: 42.3mm; height: 25.4mm;"
								>
									<span class="text-[6px] text-on-surface-variant tracking-wider font-bold"
										>LEER</span
									>
								</div>
							{:else}
								<div
									class="bg-surface-container-lowest text-on-surface text-left overflow-hidden flex flex-col justify-between {labelStore.labelBorder
										? 'border border-outline-variant'
										: ''}"
									style="width: 42.3mm; height: 25.4mm; padding: 1.5mm; font-size: 5px; box-sizing: border-box;"
								>
									<div
										class="font-extrabold text-on-surface title-clamp tracking-tight mb-0.5"
										style="font-size: 5.5px; line-height: 1.1;"
									>
										{lbl.titel}
									</div>
									<div
										class="text-on-surface-variant author-clamp"
										style="font-size: 5px; line-height: 1.1;"
									>
										{lbl.autor || 'Unbekannt'}
									</div>
									<div class="flex flex-col items-center justify-center grow pt-0.5">
										<img
											src={strichcodeBildUrl(lbl.barcode_id, {
												qr: labelStore.barcodeType === 'qr',
												width: 150,
												height: 50
											})}
											class="{labelStore.barcodeType === 'qr'
												? 'h-6 w-6'
												: 'h-4 w-full'} object-contain"
											alt="Barcode"
										/>
										<span
											class="mt-0.5 font-bold tracking-widest text-on-surface-variant"
											style="font-size: 4.5px;">{lbl.barcode_id}</span
										>
									</div>
								</div>
							{/if}
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
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
