<script>
	/**
	 * @file StudentIdDesigner.svelte
	 * Canvas-based ID-card designer — top-level coordinator component.
	 * Laden/Speichern des Designs steckt in designer/idDesignPersistenz.svelte.js.
	 */
	import { onMount } from 'svelte';
	import { idStore, serializeDesign } from './designer/idDesignerStore.svelte.js';
	import { erzeugeDesignAblage } from './designer/idDesignPersistenz.svelte.js';
	import { oeffneEtikettenbogen } from './schuelerEtiketten.js';
	import CanvasArea from './designer/CanvasArea.svelte';
	import PrintPreview from './designer/PrintPreview.svelte';
	import PropertiesPanel from './designer/PropertiesPanel.svelte';
	import Toolbar from './designer/Toolbar.svelte';

	let selectedId = $state(/** @type {string|null} */ (null));
	let side = $state(/** @type {"front"|"back"} */ ('front'));
	let zoom = $state(150);
	// Cache-Buster für die Barcode-Bilder des Testdrucks.
	let timestamp = $state(Date.now());

	const ablage = erzeugeDesignAblage();
	/** @type {any} */
	let saveTimer = null;

	// Dieser Bildschirm lädt KEINE echten Schüler mehr.
	//
	// Bis zum 06.08.2026 hing hier eine Klassenauswahl, die per /api/schueler die Schüler
	// der gewählten Klasse holte und daraus den Druckstapel baute. Das vermischte zwei
	// Dinge: Hier entsteht das Design, und das gilt zentral für ALLE Arbeitsplätze — wer
	// wessen Karte druckt, ist eine ganz andere Frage. Sie wird jetzt in der Schülerdatei
	// beantwortet (markieren → „Ausweise drucken"), wo auch das Ablaufjahr je Schüler
	// sichtbar und überschreibbar ist.
	//
	// Die Leinwand zeigt deshalb immer denselben Platzhalter. Das Gültigkeitsdatum ist
	// hier ein Musterwert und wird NICHT aus einer Klasse gerechnet: Der Bildschirm zeigt,
	// wie die Karte AUSSIEHT, nicht was auf einer bestimmten Karte steht. Das echte Datum
	// kommt beim Drucken vom Server (schueler.ausweis_gueltig_bis, Regel in
	// internal/ausweis).
	const PLATZHALTER_STUDENT = {
		id: 'placeholder',
		barcode_id: 'DEMO-S-001',
		vorname: 'Max',
		nachname: 'Mustermann',
		klasse: '9R1',
		ausweis_gueltig_bis: new Date().getFullYear() + 1
	};
	const muster = $derived({ ...PLATZHALTER_STUDENT });

	onMount(() => {
		ablage.laden();
	});

	// Auto-Save (entprellt): jede Design-Änderung wird zentral gespeichert, damit der
	// Druck-Arbeitsplatz beim nächsten Öffnen exakt denselben Stand lädt.
	$effect(() => {
		const body = JSON.stringify(serializeDesign()); // liest reaktiven State → Dependency
		if (!ablage.geladen) return;
		clearTimeout(saveTimer);
		ablage.beginneSpeichern();
		saveTimer = setTimeout(() => ablage.speichern(body), 800);
		return () => clearTimeout(saveTimer);
	});

	async function zuruecksetzen() {
		if (await ablage.zuruecksetzen()) selectedId = null;
	}

	// Etikettenbogen: dasselbe PDF wie der echte Bogen, nur mit einem Muster-Schüler.
	// Kartendrucker: die Druck-CSS des Browsers, wie bisher.
	async function triggerPrint() {
		if (idStore.printMode === 'etikett') {
			await oeffneEtikettenbogen({ formatId: idStore.etikettFormat, muster: true });
			return;
		}
		const style = document.createElement('style');
		style.textContent = '@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }';
		document.body.setAttribute('data-print-mode', 'card');
		document.body.setAttribute('data-print-side', side);
		document.head.appendChild(style);
		window.print();
		document.head.removeChild(style);
		document.body.removeAttribute('data-print-mode');
		document.body.removeAttribute('data-print-side');
	}
</script>

<div class="w-full space-y-5 no-print text-slate-800 animate-fade-in font-sans">
	<div class="flex items-center justify-end gap-3 text-xs font-semibold min-h-4">
		{#if ablage.zustand === 'saving'}
			<span class="text-slate-400">Speichert…</span>
		{:else if ablage.zustand === 'saved'}
			<span class="text-emerald-600">✓ Zentral gespeichert (alle Arbeitsplätze)</span>
		{:else if ablage.zustand === 'error'}
			<span class="text-rose-600">Speichern fehlgeschlagen</span>
		{/if}
		<button
			type="button"
			onclick={zuruecksetzen}
			class="text-slate-400 hover:text-slate-700 underline underline-offset-2 transition-colors cursor-pointer"
		>
			Standardwerte wiederherstellen
		</button>
	</div>

	<!-- printMode zentral aus idStore, nicht lokal — Begruendung in StudentBatchPrint.svelte -->
	<Toolbar
		{zoom}
		onZoom={(v) => {
			zoom = v;
		}}
		{side}
		onSide={(s) => {
			side = s;
			selectedId = null;
		}}
		printMode={idStore.printMode}
		onPrintMode={(m) => {
			idStore.printMode = m;
		}}
		onPrint={triggerPrint}
		barcodeType={idStore.barcodeType}
		onBarcodeType={(t) => {
			idStore.barcodeType = t;
		}}
		previewStudent={muster}
	/>

	<div class="w-full flex flex-col lg:flex-row gap-5">
		<CanvasArea
			{side}
			{selectedId}
			onSelect={(id) => {
				selectedId = id;
			}}
			student={muster}
			{zoom}
			barcodeType={idStore.barcodeType}
		/>

		<PropertiesPanel {selectedId} {side} />
	</div>
</div>

<!-- Testdruck: EINE Karte mit dem Platzhalter, damit sich das Layout auf dem echten
     Kartendrucker prüfen lässt, bevor ein Stapel echter Ausweise durchläuft.
     Echte Schüler druckt dieser Bildschirm nicht mehr — das tut die Schülerdatei
     (markieren → „Ausweise drucken"). -->
<PrintPreview students={[muster]} barcodeType={idStore.barcodeType} {timestamp} platzhalter />
