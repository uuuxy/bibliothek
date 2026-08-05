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
	import PropertiesPanel from './designer/PropertiesPanel.svelte';
	import Toolbar from './designer/Toolbar.svelte';
	import PrintPreview from './designer/PrintPreview.svelte';

	let selectedId = $state(/** @type {string|null} */ (null));
	let side = $state(/** @type {"front"|"back"} */ ('front'));
	let printMode = $state(/** @type {"card"|"a4"} */ ('card'));
	let zoom = $state(150);
	let classesList = $state.raw(/** @type {string[]} */ ([]));
	let selectedKlasse = $state('');
	let previewStudents = $state.raw(/** @type {any[]} */ ([]));
	let loadingStudents = $state(false);
	let timestamp = $state(Date.now());

	// Zentrale Persistenz: erst nach dem initialen Laden auto-speichern, sonst würden
	// die Store-Defaults den geladenen Stand überschreiben.
	let designLoaded = $state(false);
	/** @type {'idle'|'saving'|'saved'|'error'} */
	let saveState = $state('idle');
	/** @type {any} */
	let saveTimer = null;

	// Fallback für den Druck-Stapel (PrintPreview), falls die gewählte Klasse leer ist oder
	// /api/schueler fehlschlägt — Max Mustermann statt eines beliebig klingenden Namens,
	// passend zu PLATZHALTER_SCHULNAME ("Musterstadt").
	const mockStudents = [
		{ id: 's1', barcode_id: 'S-10041', vorname: 'Max', nachname: 'Mustermann', klasse: '9a' },
		{ id: 's2', barcode_id: 'S-10042', vorname: 'Erika', nachname: 'Musterfrau', klasse: '9a' }
	];

	// Die Design-Leinwand (CanvasArea) zeigt IMMER diesen Platzhalter, unabhängig davon,
	// welche Klasse gerade im Dropdown gewählt ist. Vorher zeigte sie previewStudents[0] —
	// den ersten echten Schüler der gewählten Klasse. Das hieß: Um im Designer einen
	// hübschen Platzhalternamen zu sehen, musste man einen echten Schüler-Datensatz
	// umbenennen. Die Klassenauswahl bleibt trotzdem nötig — sie steuert, welche echten
	// Schüler in PrintPreview/"Vorderseiten drucken" landen, das ist der tatsächliche
	// Druck-Stapel, nicht die Bearbeitungsvorschau.
	const PLATZHALTER_STUDENT = {
		id: 'placeholder',
		barcode_id: 'DEMO-S-001',
		vorname: 'Max',
		nachname: 'Mustermann',
		klasse: '9a'
	};
	const previewStudent = $derived({
		...PLATZHALTER_STUDENT,
		klasse: selectedKlasse || PLATZHALTER_STUDENT.klasse
	});

	async function loadClasses() {
		try {
			const res = await apiFetch('/api/klassen');
			if (res.ok) {
				classesList = await res.json();
				if (classesList.length > 0) {
					selectedKlasse = classesList[0];
					await loadStudents(selectedKlasse);
					return;
				}
			}
		} catch {
			/* network error — fall through to mocks */
		}
		previewStudents = mockStudents;
	}

	/** @param {string} klasse */
	async function loadStudents(klasse) {
		if (!klasse) return;
		loadingStudents = true;
		try {
			const res = await apiFetch(`/api/schueler?klasse=${encodeURIComponent(klasse)}`);
			if (res.ok) {
				const data = await res.json();
				previewStudents = data.length > 0 ? data : mockStudents;
			} else {
				previewStudents = mockStudents;
			}
		} catch {
			previewStudents = mockStudents;
		} finally {
			loadingStudents = false;
		}
	}

	onMount(() => {
		loadClasses();
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
			const adresse = [data.schule_strasse, [data.schule_plz, data.schule_ort].filter(Boolean).join(' ')]
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
		{classesList}
		{selectedKlasse}
		onKlasse={(k) => {
			selectedKlasse = k;
			loadStudents(k);
		}}
		barcodeType={idStore.barcodeType}
		onBarcodeType={(t) => {
			idStore.barcodeType = t;
		}}
		{loadingStudents}
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

<PrintPreview
	students={previewStudents.length > 0 ? previewStudents : mockStudents}
	barcodeType={idStore.barcodeType}
	{timestamp}
/>
