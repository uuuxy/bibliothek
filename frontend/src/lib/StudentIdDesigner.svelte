<script>
	import { apiFetch } from './apiFetch.js';
	/**
	 * @file StudentIdDesigner.svelte
	 * Canvas-based ID-card designer — top-level coordinator component.
	 */
	import { onMount } from 'svelte';
	import {
		idStore,
		applyDesign,
		serializeDesign,
		resetDesign,
		wendeSchulstammdatenAn
	} from './designer/idDesignerStore.svelte.js';
	import CanvasArea from './designer/CanvasArea.svelte';
	import PrintPreview from './designer/PrintPreview.svelte';
	import PropertiesPanel from './designer/PropertiesPanel.svelte';
	import Toolbar from './designer/Toolbar.svelte';

	let selectedId = $state(/** @type {string|null} */ (null));
	let side = $state(/** @type {"front"|"back"} */ ('front'));
	let printMode = $state(/** @type {"card"|"a4"} */ ('card'));
	let zoom = $state(150);
	// Cache-Buster für die Barcode-Bilder des Testdrucks.
	let timestamp = $state(Date.now());

	// Zentrale Persistenz: erst nach dem initialen Laden auto-speichern, sonst würden
	// die Store-Defaults den geladenen Stand überschreiben.
	let designLoaded = $state(false);
	/** @type {'idle'|'saving'|'saved'|'error'} */
	let saveState = $state('idle');
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
	const previewStudent = $derived({ ...PLATZHALTER_STUDENT });

	onMount(() => {
		loadDesign();
	});

	// Lädt das zentral gespeicherte Ausweis-Design. Leeres {} (Erststart) → Defaults.
	async function loadDesign() {
		try {
			const res = await apiFetch('/api/ausweis-layout');
			if (res.ok) applyDesign(await res.json());
		} catch (e) {
			console.error('Ausweis-Design konnte nicht geladen werden:', e);
		} finally {
			designLoaded = true;
		}
		// NACH applyDesign(): Sonst überschreibt das geladene Design (auch ein Design,
		// das den Platzhalter noch trägt) die geheilten Werte sofort wieder.
		await heileSchulstammdaten();
	}

	// /api/einstellungen verlangt manage_users — wer den Ausweis-Designer nur zum
	// Drucken öffnet (view_students reicht dafür), bekäme sonst ein sichtbares
	// Berechtigungs-Toast für eine reine Komfortfunktion. Deshalb roh über apiFetch
	// und bei jedem Fehler (auch 403) still nichts tun, wie bei loadClasses/loadStudents.
	async function heileSchulstammdaten() {
		try {
			const res = await apiFetch('/api/einstellungen');
			if (!res.ok) return;
			const data = await res.json();
			const adresse = [
				data.schule_strasse,
				[data.schule_plz, data.schule_ort].filter(Boolean).join(' ')
			]
				.filter(Boolean)
				.join(', ');
			wendeSchulstammdatenAn(data.schule_name ?? '', adresse);
		} catch {
			/* Komfortfunktion — Platzhalter bleibt stehen */
		}
	}

	/** @param {string} body */
	async function saveDesign(body) {
		try {
			const res = await apiFetch('/api/ausweis-layout', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body
			});
			saveState = res.ok ? 'saved' : 'error';
		} catch {
			saveState = 'error';
		}
	}

	// Auto-Save (debounced): jede Design-Änderung wird zentral gespeichert, damit der
	// Druck-Arbeitsplatz beim nächsten Öffnen exakt denselben Stand lädt.
	$effect(() => {
		const body = JSON.stringify(serializeDesign()); // liest reaktiven State → Dependency
		if (!designLoaded) return;
		clearTimeout(saveTimer);
		saveState = 'saving';
		saveTimer = setTimeout(() => saveDesign(body), 800);
		return () => clearTimeout(saveTimer);
	});

	/**
	 * Verwirft das aktuelle Design und stellt die Standardwerte her. Der Auto-Save-Effekt
	 * schreibt das Ergebnis anschließend zentral — die Rückfrage ist deshalb Pflicht:
	 * Der Schritt trifft ALLE Arbeitsplätze, nicht nur diesen Browser.
	 */
	async function zuruecksetzen() {
		const ok = window.confirm(
			'Ausweis-Design auf die Standardwerte zurücksetzen?\n\n' +
				'Alle eigenen Anpassungen an Vorder- und Rückseite gehen verloren — ' +
				'auch für die anderen Arbeitsplätze, da das Design zentral gespeichert wird.'
		);
		if (!ok) return;
		resetDesign();
		selectedId = null;
		// resetDesign() setzt den Kopf zurück auf PLATZHALTER_SCHULNAME. heileSchulstammdaten()
		// lief bisher nur einmal beim Laden — ohne diesen erneuten Aufruf hätte "Standardwerte
		// wiederherstellen" einen bereits geheilten echten Schulnamen wieder auf den
		// Platzhalter zurückgeworfen, ohne dass er sich von selbst erneut heilt.
		await heileSchulstammdaten();
	}

	function triggerPrint() {
		const style = document.createElement('style');
		if (printMode === 'a4') {
			style.textContent = '@media print { @page { size: A4; margin: 0; } }';
			document.body.setAttribute('data-print-mode', 'a4');
		} else {
			style.textContent = '@media print { @page { size: 85.6mm 53.98mm; margin: 0; } }';
			document.body.setAttribute('data-print-mode', 'card');
		}
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
		{#if saveState === 'saving'}
			<span class="text-slate-400">Speichert…</span>
		{:else if saveState === 'saved'}
			<span class="text-emerald-600">✓ Zentral gespeichert (alle Arbeitsplätze)</span>
		{:else if saveState === 'error'}
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
		{printMode}
		onPrintMode={(m) => {
			printMode = m;
		}}
		onPrint={triggerPrint}
		barcodeType={idStore.barcodeType}
		onBarcodeType={(t) => {
			idStore.barcodeType = t;
		}}
		{previewStudent}
	/>

	<div class="w-full flex flex-col lg:flex-row gap-5">
		<CanvasArea
			{side}
			{selectedId}
			onSelect={(id) => {
				selectedId = id;
			}}
			student={previewStudent}
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
<PrintPreview students={[previewStudent]} barcodeType={idStore.barcodeType} {timestamp} />
