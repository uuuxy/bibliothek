<script>
	/**
	 * @file AusweisDesigner.svelte
	 * Canvas-based ID-card designer — top-level coordinator component.
	 * Laden/Speichern des Designs steckt in designer/idDesignPersistenz.svelte.js.
	 */
	import { onDestroy, onMount } from 'svelte';
	import { idStore, serializeDesign } from './designer/idDesignerStore.svelte.js';
	import { erzeugeDesignAblage } from './designer/idDesignPersistenz.svelte.js';
	import { oeffneEtikettenbogen } from './schuelerEtiketten.js';
	import CanvasFlaeche from './designer/CanvasFlaeche.svelte';
	import Druckvorschau from './designer/Druckvorschau.svelte';
	import Eigenschaften from './designer/Eigenschaften.svelte';
	import Werkzeugleiste from './designer/Werkzeugleiste.svelte';

	let gewaehlteKennung = $state(/** @type {string|null} */ (null));
	let side = $state(/** @type {"front"|"back"} */ ('front'));
	let zoom = $state(150);
	// Cache-Buster für die Barcode-Bilder des Testdrucks.
	let timestamp = $state(Date.now());

	const ablage = erzeugeDesignAblage();
	/** @type {any} */
	let speicherZeitgeber = null;

	// Dieser Bildschirm lädt keine echten Schüler: Hier entsteht das Design, das zentral für
	// alle Arbeitsplätze gilt. Wer wessen Karte druckt, entscheidet die Schülerdatei
	// (markieren → „Ausweise drucken"), wo auch das Ablaufjahr je Schüler steht.
	//
	// Die Leinwand zeigt deshalb immer denselben Platzhalter, und das Gültigkeitsdatum ist ein
	// Musterwert: Der Bildschirm zeigt, wie die Karte aussieht, nicht was auf einer bestimmten
	// Karte steht. Das echte Datum kommt beim Drucken vom Server (ausweis_gueltig_bis, Regel in
	// internal/ausweis).
	const PLATZHALTER_LESER = {
		id: 'placeholder',
		barcode_id: 'DEMO-S-001',
		vorname: 'Max',
		nachname: 'Mustermann',
		klasse: '9R1',
		ausweis_gueltig_bis: new Date().getFullYear() + 1
	};
	const muster = $derived({ ...PLATZHALTER_LESER });

	onMount(() => {
		ablage.laden();
	});

	// Auto-Save (entprellt): Jede Design-Änderung wird zentral gespeichert, damit der
	// Druck-Arbeitsplatz beim nächsten Öffnen denselben Stand lädt. `ausstehend` hält den
	// letzten noch nicht gespeicherten Stand, denn der Abbau des Effekts löscht den Timer: Wer
	// innerhalb der 800 ms weiterklickt, verlöre sonst die letzte Änderung. onDestroy schickt
	// den ausstehenden Stand deshalb sofort ab.
	/** @type {string | null} */
	let ausstehend = null;
	$effect(() => {
		const body = JSON.stringify(serializeDesign()); // liest reaktiven State → Dependency
		if (!ablage.geladen) return;
		clearTimeout(speicherZeitgeber);
		ablage.beginneSpeichern();
		ausstehend = body;
		speicherZeitgeber = setTimeout(() => {
			ausstehend = null;
			ablage.speichern(body);
		}, 800);
		return () => clearTimeout(speicherZeitgeber);
	});
	onDestroy(() => {
		if (ausstehend !== null) ablage.speichern(ausstehend);
	});

	async function zuruecksetzen() {
		if (await ablage.zuruecksetzen()) gewaehlteKennung = null;
	}

	/** @param {string} kennung */
	async function vorlageAnwenden(kennung) {
		if (await ablage.vorlageAnwenden(kennung)) gewaehlteKennung = null;
	}

	// Etikettenbogen: dasselbe PDF wie der echte Bogen, nur mit einem Muster-Schüler.
	// Kartendrucker: die Druck-CSS des Browsers, wie bisher.
	async function drucke() {
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

<div class="w-full space-y-5 no-print text-on-surface animate-fade-in font-sans">
	{#if ablage.ladefehler}
		<!-- Ohne den zentralen Stand zeigt die Leinwand Vorgabewerte, und die Auto-Speicherung
		     ist aus: Sie schöbe diese Vorgabewerte sonst an alle Arbeitsplätze. Der Bildschirm
		     sagt beides. -->
		<div
			class="rounded-xl border border-error bg-error-container px-4 py-3 text-sm text-on-error-container"
			role="alert"
		>
			<p class="font-semibold">{ablage.ladefehler}</p>
			<button
				type="button"
				onclick={() => ablage.laden()}
				class="mt-1 font-semibold underline underline-offset-2 cursor-pointer"
			>
				Erneut laden
			</button>
		</div>
	{/if}

	<div class="flex items-center justify-end gap-3 text-sm font-semibold min-h-4">
		{#if ablage.zustand === 'saving'}
			<span class="text-on-surface-variant">Speichert…</span>
		{:else if ablage.zustand === 'saved'}
			<span class="text-success">✓ Zentral gespeichert (alle Arbeitsplätze)</span>
		{:else if ablage.zustand === 'error'}
			<span class="text-error">Speichern fehlgeschlagen</span>
		{/if}
		<button
			type="button"
			onclick={zuruecksetzen}
			class="text-on-surface-variant underline underline-offset-2 cursor-pointer"
		>
			Standardwerte wiederherstellen
		</button>
	</div>

	<!-- printMode zentral aus idStore, nicht lokal — Begruendung in AusweisStapeldruck.svelte -->
	<Werkzeugleiste
		{zoom}
		onZoom={(v) => {
			zoom = v;
		}}
		{side}
		onSide={(s) => {
			side = s;
			gewaehlteKennung = null;
		}}
		druckArt={idStore.printMode}
		onDruckArt={(m) => {
			idStore.printMode = m;
		}}
		onPrint={drucke}
		barcodeType={idStore.barcodeType}
		onBarcodeType={(t) => {
			idStore.barcodeType = t;
		}}
		onVorlage={vorlageAnwenden}
		{muster}
	/>

	<div class="w-full flex flex-col lg:flex-row gap-5">
		<CanvasFlaeche
			{side}
			{gewaehlteKennung}
			onSelect={(id) => {
				gewaehlteKennung = id;
			}}
			leser={muster}
			{zoom}
			barcodeType={idStore.barcodeType}
		/>

		<Eigenschaften {gewaehlteKennung} {side} />
	</div>
</div>

<!-- Testdruck: EINE Karte mit dem Platzhalter, damit sich das Layout auf dem echten
     Kartendrucker prüfen lässt, bevor ein Stapel echter Ausweise durchläuft.
     Echte Schüler druckt dieser Bildschirm nicht mehr — das tut die Schülerdatei
     (markieren → „Ausweise drucken"). -->
<Druckvorschau leserListe={[muster]} barcodeType={idStore.barcodeType} {timestamp} platzhalter />
