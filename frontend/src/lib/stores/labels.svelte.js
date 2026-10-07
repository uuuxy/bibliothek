// stores/labels.svelte.js
// Status- und Logikverwaltung für den Etikettendruck (Svelte 5 Runes)

import { apiFetch } from '../apiFetch.js';
import { felderProBogen } from '../etikettformate.js';
import { printQueue, clearPrintQueue } from './printQueue.svelte.js';
import { toastStore } from './toastStore.svelte.js';
import { normalisiereScan } from '../scanEinordnen.js';
import { dekodiereLitteraEtikett } from '../litteraEtikett.js';
import { ohnePruefzeichen } from '../code39Pruefzeichen.js';

// Bewusst im Modul-Scope und nicht im Store: Die Funktion greift auf nichts aus
// dem Store zu — nur auf apiFetch und ihr Argument. Innen definiert wurde sie bei
// jedem createLabelStore() neu angelegt (sonarjs S7721).
/**
 * Bucht den Druck gegen, damit die Nachdruck-Liste die Exemplare loswird.
 *
 * Gemessen wird am erzeugten PDF, nicht am Papier — mehr weiss der Browser nicht. Wer
 * den Ausdruck abbricht, findet das Exemplar über die Suche im Nachdruck wieder.
 *
 * Scheitert der Vermerk, bleibt der Druck trotzdem gültig: Das PDF ist bereits offen,
 * und ein Fehler an dieser Stelle darf ihn nicht als fehlgeschlagen erscheinen lassen.
 * Der Preis ist ein Exemplar, das erneut auf der Liste steht — harmlos gegenüber einem
 * Etikett, das niemand mehr nachdruckt.
 * @param {Array<{barcode_id?: string}>} gedruckt
 */
async function vermerkeGedruckt(gedruckt) {
	const barcodes = gedruckt.map((l) => l.barcode_id).filter(Boolean);
	if (barcodes.length === 0) return;
	try {
		await apiFetch('/api/exemplare/etiketten-gedruckt', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ barcode_ids: barcodes })
		});
	} catch (err) {
		console.error('Etiketten konnten nicht als gedruckt vermerkt werden', err);
	}
}

function createLabelStore() {
	let searchVal = $state('');
	let searchResults = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);

	let classGroups = $state.raw(/** @type {any[]} */ ([]));
	let selectedClass = $state('');
	let classBooks = $state.raw(/** @type {any[]} */ ([]));

	let selectedTitle = $state(/** @type {any} */ (null));
	let barcodeType = $state('code39'); // "code39" | "qr"
	let labelBorder = $state(true);
	let startPosition = $state(1); // 1 to 21

	// Vorgabe = die des Servers (api/label_formats.go StandardLabelFormat): das Papier des
	// Lieferantenwegs. Bis 24.08.2026 stand hier avery_3475 — zwei Vorgaben für ein Raster.
	let formatId = $state('zweckform_l4760');
	// Aus der gemeinsamen Formatliste, nicht aus einer eigenen Zahlenkette: Die drei
	// Zahlen standen hier als dritte Kopie derselben Raster (24.08.2026).
	let maxPositions = $derived(felderProBogen(formatId));
	let generationMode = $state('existing');
	// Tief reaktiv, nicht $state.raw: Das Kästchen in Schritt 2 schreibt `checked` an das
	// Exemplar selbst, und der Bogen (finalLabels) muss das sehen.
	let existingCopies = $state(/** @type {any[]} */ ([]));
	// Ausgesonderte Exemplare stehen nicht mehr im Regal und deshalb nicht in der Liste; ihre
	// Zahl steht als Hinweis darunter. Dieselbe Regel wie repository.EtikettOffenBedingung.
	let ausgesondertAnzahl = $state(0);
	let loadingCopies = $state(false);
	// Ein gescheiterter Abruf ist ein eigener Zustand: Eine leere Liste hieße „kein Exemplar"
	// und legte nahe, neue Barcodes zu erzeugen.
	let exemplareNichtGeladen = $state(false);
	// Der Text im Nummernfeld über der Liste: zeigt nur die Exemplare, deren Nummer ihn enthält.
	let exemplarSuche = $state('');
	let sichtbareExemplare = $derived.by(() => {
		const text = exemplarSuche.trim().toLowerCase();
		if (!text) return existingCopies;
		return existingCopies.filter((c) => String(c.barcode_id).toLowerCase().includes(text));
	});
	let auswahl = $derived({
		gewaehlt: existingCopies.filter((c) => c.checked).length,
		gesamt: existingCopies.length
	});
	/** @type {'alle' | 'teil' | 'keine'} */
	let auswahlSichtbar = $derived.by(() => {
		const gewaehlt = sichtbareExemplare.filter((c) => c.checked).length;
		if (gewaehlt === 0) return 'keine';
		return gewaehlt === sichtbareExemplare.length ? 'alle' : 'teil';
	});
	let newQuantity = $state(9);
	let newStartNum = $state(20060);

	let searchTimeout = /** @type {any} */ (null);

	/** @type {Array<{isBlank?: boolean, barcode_id?: string, titel?: string, autor?: string}>} */
	let finalLabels = $derived.by(() => {
		if ((printQueue.copies?.length ?? 0) > 0) {
			const copies = /** @type {any[]} */ (printQueue.copies);
			const rawList = copies.map((c) => ({
				barcode_id: c.barcode_id,
				titel: c.titel,
				autor: c.autor || ''
			}));
			const offsetCount = Math.max(0, startPosition - 1);
			const offsetLabels = Array.from({ length: offsetCount }, () => ({ isBlank: true }));
			return [...offsetLabels, ...rawList];
		}

		if (!selectedTitle) return [];

		let rawList = [];
		if (generationMode === 'existing') {
			rawList = existingCopies
				.filter((c) => c.checked)
				.map((c) => ({
					barcode_id: c.barcode_id,
					titel: selectedTitle.titel,
					autor: selectedTitle.autor || ''
				}));
		} else {
			rawList = Array.from({ length: Math.max(1, newQuantity) }, (_, i) => ({
				barcode_id: `B-${newStartNum + i}`,
				titel: selectedTitle.titel,
				autor: selectedTitle.autor || ''
			}));
		}

		const offsetCount = Math.max(0, startPosition - 1);
		const offsetLabels = Array.from({ length: offsetCount }, () => ({ isBlank: true }));

		return [...offsetLabels, ...rawList];
	});

	async function loadClassGroups() {
		try {
			const res = await apiFetch('/api/class-books');
			if (res.ok) {
				const body = await res.json();
				if (body?.data) {
					classGroups = body.data;
				}
			}
		} catch (err) {
			console.error('Fehler beim Laden der Klassengruppen:', err);
		}
	}

	function handleClassChange() {
		const group = classGroups.find((g) => g.className === selectedClass);
		if (group) {
			classBooks = group.books || [];
		} else {
			classBooks = [];
		}
		selectedTitle = null;
		existingCopies = [];
		ausgesondertAnzahl = 0;
		exemplareNichtGeladen = false;
		exemplarSuche = '';
	}

	/**
	 * Das Kästchen über der Liste: wählt oder entfernt, was gerade zu sehen ist. Was das
	 * Nummernfeld ausblendet, bleibt, wie es war.
	 * @param {boolean} gewaehlt
	 */
	function setzeSichtbare(gewaehlt) {
		for (const exemplar of sichtbareExemplare) exemplar.checked = gewaehlt;
	}

	/**
	 * Die Eingabetaste im Nummernfeld, wie sie ein Handscanner schickt: setzt das Exemplar mit
	 * genau dieser Nummer auf den Bogen und leert das Feld. Gelesen wird wie an der Theke
	 * (scanEinordnen.js): die Nummer selbst, dann das Littera-Etikett, zuletzt ein Etikett mit
	 * Prüfzeichen. Ein Teil einer Nummer wählt nichts; er zeigt nur die passenden Zeilen.
	 * Groß und klein zählen nicht: Das Feld sucht nur Nummern dieses Titels, kein Wort.
	 * @returns {boolean} ob ein Exemplar dieses Titels gefunden wurde
	 */
	function uebernimmNummer() {
		const scan = normalisiereScan(exemplarSuche);
		if (!scan) return false;
		for (const nummer of [scan, dekodiereLitteraEtikett(scan), ohnePruefzeichen(scan)]) {
			const gesucht = nummer?.toLowerCase();
			const exemplar = gesucht
				? existingCopies.find((c) => String(c.barcode_id).toLowerCase() === gesucht)
				: undefined;
			if (exemplar) {
				exemplar.checked = true;
				exemplarSuche = '';
				return true;
			}
		}
		return false;
	}

	function handleSearchInput() {
		if (searchTimeout) clearTimeout(searchTimeout);
		if (!searchVal.trim()) {
			searchResults = [];
			return;
		}
		isSearching = true;
		searchTimeout = setTimeout(async () => {
			try {
				// Titel-Tür (view_books) — nie POST /api/action (dort löst ein gescannter
				// B-Barcode ohne aktiven Schüler eine Rückgabe aus, Inventur 03.09.2026) und seit
				// 05.09.2026 auch nicht mehr die Theken-Suche /api/search: Die lieferte
				// Schüler-Kiosk-Daten mit, die dieser Bildschirm nie zeigt, und hing am
				// Theken-Recht perform_actions, das ein Druckbildschirm nicht braucht.
				const res = await apiFetch(
					`/api/buecher/titel/suche?q=${encodeURIComponent(searchVal.trim())}`
				);
				if (res.ok) {
					const body = await res.json();
					searchResults = body.books || [];
				} else {
					// Sweep „verschluckte Fehlantwort" (06.09.2026): Vorher blieben die
					// Treffer des VORIGEN Suchtextes stehen — man klickt auf eine Zeile,
					// die zu einer anderen Eingabe gehört, und druckt deren Etikett.
					searchResults = [];
					toastStore.addToast('Titelsuche fehlgeschlagen — bitte erneut versuchen.', 'error');
				}
			} catch (err) {
				searchResults = [];
				console.error('Fehler bei Buchtitelsuche:', err);
			} finally {
				isSearching = false;
			}
		}, 300);
	}

	/** @param {any} titleObj */
	async function selectBookTitle(titleObj) {
		selectedTitle = titleObj;
		searchResults = [];
		searchVal = titleObj.titel;
		selectedClass = '';
		classBooks = [];
		await loadExistingCopies();
	}

	async function loadExistingCopies() {
		if (!selectedTitle) return;
		loadingCopies = true;
		exemplarSuche = '';
		ausgesondertAnzahl = 0;
		exemplareNichtGeladen = false;
		try {
			const res = await apiFetch(`/api/buecher/titel/${selectedTitle.id}/exemplare`);
			if (res.ok) {
				/** @type {any[]} */
				const alle = (await res.json()) || [];
				existingCopies = alle
					.filter((c) => !c.ist_ausgesondert)
					.map((c) => ({ ...c, checked: true }));
				ausgesondertAnzahl = alle.length - existingCopies.length;
			} else {
				existingCopies = [];
				exemplareNichtGeladen = true;
			}
		} catch (err) {
			console.error('Fehler beim Laden der Exemplare:', err);
			existingCopies = [];
			exemplareNichtGeladen = true;
		} finally {
			loadingCopies = false;
		}
	}

	async function triggerPrint() {
		const itemsToPrint = finalLabels.filter((l) => !l.isBlank);
		if (itemsToPrint.length === 0) return;

		try {
			const res = await apiFetch('/api/print/labels', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					formatId: formatId,
					startPosition: startPosition,
					isQR: barcodeType === 'qr',
					items: itemsToPrint.map((l) => ({
						BarcodeID: l.barcode_id,
						Titel: l.titel,
						Autor: l.autor
					}))
				})
			});

			if (res.ok) {
				const blob = await res.blob();
				const url = window.URL.createObjectURL(blob);
				window.open(url, '_blank');
				await vermerkeGedruckt(itemsToPrint);
			} else {
				console.error('Fehler beim Erstellen des PDFs');
				toastStore.addToast('Fehler beim Erstellen des PDFs', 'error');
			}
		} catch (err) {
			console.error('Netzwerkfehler beim Drucken', err);
			toastStore.addToast('Fehler beim Senden der Daten', 'error');
		}
	}

	return {
		get searchVal() {
			return searchVal;
		},
		set searchVal(v) {
			searchVal = v;
		},
		get searchResults() {
			return searchResults;
		},
		get isSearching() {
			return isSearching;
		},

		get classGroups() {
			return classGroups;
		},
		get selectedClass() {
			return selectedClass;
		},
		set selectedClass(v) {
			selectedClass = v;
		},
		get classBooks() {
			return classBooks;
		},

		get selectedTitle() {
			return selectedTitle;
		},
		get barcodeType() {
			return barcodeType;
		},
		set barcodeType(v) {
			barcodeType = v;
		},
		get labelBorder() {
			return labelBorder;
		},
		set labelBorder(v) {
			labelBorder = v;
		},
		get formatId() {
			return formatId;
		},
		set formatId(v) {
			formatId = v;
		},
		get startPosition() {
			return startPosition;
		},
		set startPosition(v) {
			startPosition = v;
		},

		get generationMode() {
			return generationMode;
		},
		set generationMode(v) {
			generationMode = v;
		},
		get existingCopies() {
			return existingCopies;
		},
		set existingCopies(v) {
			existingCopies = v;
		},
		get loadingCopies() {
			return loadingCopies;
		},
		get exemplareNichtGeladen() {
			return exemplareNichtGeladen;
		},
		get ausgesondertAnzahl() {
			return ausgesondertAnzahl;
		},
		get exemplarSuche() {
			return exemplarSuche;
		},
		set exemplarSuche(v) {
			exemplarSuche = v;
		},
		get sichtbareExemplare() {
			return sichtbareExemplare;
		},
		get auswahl() {
			return auswahl;
		},
		get auswahlSichtbar() {
			return auswahlSichtbar;
		},
		get newQuantity() {
			return newQuantity;
		},
		set newQuantity(v) {
			newQuantity = v;
		},
		get newStartNum() {
			return newStartNum;
		},
		set newStartNum(v) {
			newStartNum = v;
		},

		get finalLabels() {
			return finalLabels;
		},
		get maxPositions() {
			return maxPositions;
		},

		loadClassGroups,
		handleClassChange,
		handleSearchInput,
		selectBookTitle,
		ladeExemplare: loadExistingCopies,
		setzeSichtbare,
		uebernimmNummer,
		triggerPrint,

		resetPendingCopies() {
			clearPrintQueue();
		}
	};
}

export const labelStore = createLabelStore();
