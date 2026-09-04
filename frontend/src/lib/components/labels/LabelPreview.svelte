<script>
	import { formatKurzname } from '../../etikettformate.js';
	import { labelStore } from '../../stores/labels.svelte.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
</script>

<!-- Vorschau: die kleinere Haelfte (5 von 12), nicht die groessere.
     Vorher lag es umgekehrt — 7 Spalten fuer die Vorschau, 5 fuer das Formular. Bei
     1600 px waren das 696 gegen 488 px: Die ARBEIT hatte die kleinere Haelfte, und im
     Leerzustand stand ein halber Bildschirm auf "Kein Buch ausgewaehlt".

     Auch nicht mehr gestrichelt und nicht mehr rounded-3xl. Gestrichelt heisst in
     dieser Anwendung "hier gehoert etwas hin" (Ablageflaeche, Leerzustand) — eine
     Vorschau ist aber kein Platzhalter, sondern die Ausgabe. Und 28 px Radius ist in
     der Formskala (theme-mass.css) der DIALOG-Wert; hier steht kein Dialog.

     min-h-125 (500 px feste Mindesthoehe) ist weg: Der Kasten war im Leerzustand einen
     halben Bildschirm hoch, ohne etwas zu zeigen. Er waechst jetzt mit seinem Inhalt. -->
<div
	class="lg:col-span-5 flex flex-col items-center justify-start rounded-xl bg-surface-container p-6"
>
	<span class="text-xs text-slate-400 font-medium mb-4"
		>A4 Etiketten-Vorschau · {formatKurzname(labelStore.formatId)}</span
	>

	{#if !labelStore.selectedTitle && (printQueue.copies?.length ?? 0) === 0}
		<div class="grow flex flex-col items-center justify-center text-slate-400 py-12">
			<span>Kein Buch ausgewählt</span>
			<span class="mt-1 text-xs text-slate-500"
				>Suche einen Titel links, um die Live-Vorschau zu aktivieren.</span
			>
		</div>
	{:else if labelStore.finalLabels.length === 0}
		<div class="grow flex flex-col items-center justify-center text-slate-400 py-12">
			<span>Keine Etiketten gewählt</span>
			<span class="mt-1 text-xs text-slate-500"
				>Wähle mindestens ein Exemplar oder erhöhe die Menge.</span
			>
		</div>
	{:else}
		<!-- A4 Page Mockup container — proportionally scaled to 2/3 A4 -->
		<div
			class="bg-white border border-slate-300 shadow-2xl relative flex flex-col items-start select-none"
			style="width: 140mm; min-height: 198mm; padding: 10.1mm 4.8mm 0; box-sizing: border-box;"
		>
			<div
				style="display: grid; grid-template-columns: repeat(3, 42.3mm); column-gap: 1.7mm; row-gap: 0; width: 100%;"
			>
				{#each labelStore.finalLabels as lbl, _i (_i)}
					{#if lbl.isBlank}
						<!-- Blank Label placeholder representation -->
						<div
							class="border border-dashed border-slate-200 bg-slate-50 flex items-center justify-center"
							style="width: 42.3mm; height: 25.4mm;"
						>
							<span class="text-[6px] text-slate-400 tracking-wider font-bold">LEER</span>
						</div>
					{:else}
						<div
							class="bg-white text-slate-800 text-left overflow-hidden flex flex-col justify-between {labelStore.labelBorder
								? 'border border-slate-300'
								: ''}"
							style="width: 42.3mm; height: 25.4mm; padding: 1.5mm; font-size: 5px; box-sizing: border-box;"
						>
							<div
								class="font-extrabold text-slate-900 title-clamp tracking-tight mb-0.5"
								style="font-size: 5.5px; line-height: 1.1;"
							>
								{lbl.titel}
							</div>
							<div class="text-slate-500 author-clamp" style="font-size: 5px; line-height: 1.1;">
								{lbl.autor || 'Unbekannt'}
							</div>
							<div class="flex flex-col items-center justify-center grow pt-0.5">
								<img
									src="/api/barcode?content={lbl.barcode_id}&qr={labelStore.barcodeType ===
										'qr'}&width=150&height=50"
									class="{labelStore.barcodeType === 'qr'
										? 'h-6 w-6'
										: 'h-4 w-full'} object-contain"
									alt="Barcode"
								/>
								<span
									class="mt-0.5 font-bold tracking-widest text-slate-600"
									style="font-size: 4.5px;">{lbl.barcode_id}</span
								>
							</div>
						</div>
					{/if}
				{/each}
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
