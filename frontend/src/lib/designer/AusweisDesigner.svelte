<script>
	// Der Ausweis-Designer: ordnet Leinwand, Werkzeugleiste und Eigenschaften an. Laden und Speichern
	// des Designs stehen in idDesignPersistenz.svelte.js.
	import { onDestroy, onMount } from 'svelte';
	import { idStore, serializeDesign } from './idDesignerStore.svelte.js';
	import { erzeugeDesignAblage } from './idDesignPersistenz.svelte.js';
	import { oeffneEtikettenbogen } from '../components/etiketten/schuelerEtiketten.js';
	import { authStore } from '../stores/authStore.svelte.js';
	import { hatRecht } from '../menue.js';
	import Ladekreis from '../components/ui/Ladekreis.svelte';
	import CanvasFlaeche from './CanvasFlaeche.svelte';
	import Druckvorschau from './Druckvorschau.svelte';
	import Eigenschaften from './Eigenschaften.svelte';
	import Werkzeugleiste from './Werkzeugleiste.svelte';

	let gewaehlteKennung = $state(/** @type {string|null} */ (null));
	let side = $state(/** @type {"front"|"back"} */ ('front'));
	let zoom = $state(150);
	// Cache-Buster für die Barcode-Bilder des Testdrucks.
	let timestamp = $state(Date.now());

	const ablage = erzeugeDesignAblage();
	/** @type {any} */
	let speicherZeitgeber = null;

	// Ansehen und testdrucken darf, wer Ausweise druckt. Das Design gilt für alle Arbeitsplätze;
	// ändern darf es nur, wer die Einstellungen verwaltet (api/routen_system.go).
	const gesperrt = $derived(!hatRecht(authStore.currentUser, 'manage_settings'));

	// Hier entsteht das Design, wer wessen Karte druckt, entscheidet die Schülerdatei. Die Leinwand
	// zeigt deshalb immer denselben Platzhalter, und das Gültigkeitsdatum ist ein Musterwert. Das
	// echte kommt beim Drucken vom Server (ausweis_gueltig_bis, Regel in internal/ausweis).
	const muster = {
		id: 'placeholder',
		barcode_id: 'DEMO-S-001',
		vorname: 'Max',
		nachname: 'Mustermann',
		klasse: '9R1',
		ausweis_gueltig_bis: new Date().getFullYear() + 1
	};

	onMount(() => ablage.laden());

	// Auto-Save (entprellt): Jede Design-Änderung wird zentral gespeichert, damit der
	// Druck-Arbeitsplatz beim nächsten Öffnen denselben Stand lädt. `ausstehend` hält den
	// letzten noch nicht gespeicherten Stand, denn der Abbau des Effekts löscht den Timer: Wer
	// innerhalb der 800 ms weiterklickt, verlöre sonst die letzte Änderung. onDestroy schickt
	// den ausstehenden Stand deshalb sofort ab.
	/** @type {string | null} */
	let ausstehend = null;
	// Der Stand, den der Effekt zuletzt gesehen hat. Der erste nach dem Laden ist der geladene:
	// Den Bildschirm zu öffnen, schreibt nichts.
	/** @type {string | null} */
	let gesehen = null;
	$effect(() => {
		const body = JSON.stringify(serializeDesign()); // liest reaktiven State → Dependency
		if (!ablage.geladen) return;
		if (gesperrt) return;
		const unveraendert = gesehen === null || body === gesehen;
		gesehen = body;
		if (unveraendert) return;
		clearTimeout(speicherZeitgeber);
		ablage.beginneSpeichern();
		ausstehend = body;
		speicherZeitgeber = setTimeout(() => {
			ausstehend = null;
			ablage.speichern(body);
		}, 800);
	});
	onDestroy(() => {
		clearTimeout(speicherZeitgeber);
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

<div
	class="w-full space-y-5 no-print text-on-surface animate-fade-in font-sans"
	inert={ablage.ladend}
>
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
		{#if ablage.ladekreis}
			<Ladekreis size="sm" etikett="Ausweis-Design lädt" />
		{:else if ablage.zustand === 'saving'}
			<span class="text-on-surface-variant">Speichert…</span>
		{:else if ablage.zustand === 'saved'}
			<span class="text-success">✓ Zentral gespeichert (alle Arbeitsplätze)</span>
		{:else if ablage.zustand === 'error'}
			<span role="alert" class="text-error">Speichern fehlgeschlagen: {ablage.speicherfehler}</span>
			<button
				type="button"
				onclick={() => ablage.erneutSpeichern()}
				class="text-on-surface-variant underline underline-offset-2 cursor-pointer"
			>
				Erneut speichern
			</button>
		{/if}
		<button
			type="button"
			onclick={zuruecksetzen}
			disabled={gesperrt}
			class="text-on-surface-variant underline underline-offset-2 cursor-pointer disabled:cursor-not-allowed disabled:text-on-surface/38"
		>
			Standardwerte wiederherstellen
		</button>
	</div>

	<!-- printMode zentral aus idStore, nicht lokal — Begruendung in AusweisStapeldruck.svelte -->
	<Werkzeugleiste
		{gesperrt}
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
			{gesperrt}
			{side}
			{gewaehlteKennung}
			onSelect={(id) => {
				gewaehlteKennung = id;
			}}
			leser={muster}
			{zoom}
			barcodeType={idStore.barcodeType}
		/>

		<Eigenschaften {gewaehlteKennung} {side} {gesperrt} />
	</div>
</div>

<!-- Testdruck: EINE Karte mit dem Platzhalter, damit sich das Layout auf dem echten
     Kartendrucker prüfen lässt, bevor ein Stapel echter Ausweise durchläuft.
     Echte Schüler druckt dieser Bildschirm nicht mehr — das tut die Schülerdatei
     (markieren → „Ausweise drucken"). -->
<Druckvorschau leserListe={[muster]} barcodeType={idStore.barcodeType} {timestamp} platzhalter />
