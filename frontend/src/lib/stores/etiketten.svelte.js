// stores/etiketten.svelte.js
// Status- und Logikverwaltung für den Etikettendruck (Svelte 5 Runes)

import { apiFetch } from '../apiFetch.js';
import { felderProBogen } from '../etikettformate.js';
import { druckWarteschlange, leereDruckwarteschlange } from './druckWarteschlange.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { normalisiereScan } from '../scanEinordnen.js';
import { dekodiereLitteraEtikett } from '../litteraEtikett.js';
import { ohnePruefzeichen } from '../code39Pruefzeichen.js';

// Bewusst im Modul-Scope und nicht im Store: Die Funktion greift auf nichts aus
// dem Store zu — nur auf apiFetch und ihr Argument. Innen definiert wurde sie bei
// jedem erzeugeEtikettenStore() neu angelegt (sonarjs S7721).
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

function erzeugeEtikettenStore() {
	let suche = $state('');
	let treffer = $state.raw(/** @type {any[]} */ ([]));
	let sucht = $state(false);

	let klassen = $state.raw(/** @type {any[]} */ ([]));
	let gewaehlteKlasse = $state('');
	let klassenTitel = $state.raw(/** @type {any[]} */ ([]));

	let gewaehlterTitel = $state(/** @type {any} */ (null));
	let barcodeArt = $state('code39'); // "code39" | "qr"
	let mitRahmen = $state(true);
	let startFeld = $state(1); // 1 to 21

	// Vorgabe = die des Servers (pdf/etikett_formate.go EtikettFormatVorgabe): das Papier des
	// Lieferantenwegs. Bis 24.08.2026 stand hier avery_3475 — zwei Vorgaben für ein Raster.
	let formatId = $state('zweckform_l4760');
	// Aus der gemeinsamen Formatliste, nicht aus einer eigenen Zahlenkette: Die drei
	// Zahlen standen hier als dritte Kopie derselben Raster (24.08.2026).
	let felderJeBogen = $derived(felderProBogen(formatId));
	let modus = $state('existing');
	// Tief reaktiv, nicht $state.raw: Das Kästchen in Schritt 2 schreibt `checked` an das
	// Exemplar selbst, und der Bogen (bogen) muss das sehen.
	let vorhandeneExemplare = $state(/** @type {any[]} */ ([]));
	// Ausgesonderte Exemplare stehen nicht mehr im Regal und deshalb nicht in der Liste; ihre
	// Zahl steht als Hinweis darunter. Dieselbe Regel wie repository.EtikettOffenBedingung.
	let ausgesondertAnzahl = $state(0);
	let laedtExemplare = $state(false);
	// Ein gescheiterter Abruf ist ein eigener Zustand: Eine leere Liste hieße „kein Exemplar"
	// und legte nahe, neue Barcodes zu erzeugen.
	let exemplareNichtGeladen = $state(false);
	// Zählt die Abrufe der Exemplare: Nur die Antwort des jüngsten zählt. Die langsamere
	// Antwort eines früher gewählten Titels stellte sonst dessen Nummern unter den neuen Titel.
	let exemplarLauf = 0;
	// Der Text im Nummernfeld über der Liste: zeigt nur die Exemplare, deren Nummer ihn enthält.
	let exemplarSuche = $state('');
	let sichtbareExemplare = $derived.by(() => {
		const text = exemplarSuche.trim().toLowerCase();
		if (!text) return vorhandeneExemplare;
		return vorhandeneExemplare.filter((c) => String(c.barcode_id).toLowerCase().includes(text));
	});
	let auswahl = $derived({
		gewaehlt: vorhandeneExemplare.filter((c) => c.checked).length,
		gesamt: vorhandeneExemplare.length
	});
	/** @type {'alle' | 'teil' | 'keine'} */
	let auswahlSichtbar = $derived.by(() => {
		const gewaehlt = sichtbareExemplare.filter((c) => c.checked).length;
		if (gewaehlt === 0) return 'keine';
		return gewaehlt === sichtbareExemplare.length ? 'alle' : 'teil';
	});
	let neueAnzahl = $state(9);
	let neueStartnummer = $state(20060);

	let suchVerzoegerung = /** @type {any} */ (null);
	// Zählt die Titelsuchen wie exemplarLauf die Abrufe der Exemplare.
	let suchLauf = 0;

	/** @type {Array<{isBlank?: boolean, barcode_id?: string, titel?: string, autor?: string}>} */
	let bogen = $derived.by(() => {
		if ((druckWarteschlange.copies?.length ?? 0) > 0) {
			const copies = /** @type {any[]} */ (druckWarteschlange.copies);
			const etikettenListe = copies.map((c) => ({
				barcode_id: c.barcode_id,
				titel: c.titel,
				autor: c.autor || ''
			}));
			const offsetCount = Math.max(0, startFeld - 1);
			const leereFelder = Array.from({ length: offsetCount }, () => ({ isBlank: true }));
			return [...leereFelder, ...etikettenListe];
		}

		if (!gewaehlterTitel) return [];

		let etikettenListe = [];
		if (modus === 'existing') {
			etikettenListe = vorhandeneExemplare
				.filter((c) => c.checked)
				.map((c) => ({
					barcode_id: c.barcode_id,
					titel: gewaehlterTitel.titel,
					autor: gewaehlterTitel.autor || ''
				}));
		} else {
			etikettenListe = Array.from({ length: Math.max(1, neueAnzahl) }, (_, i) => ({
				barcode_id: `B-${neueStartnummer + i}`,
				titel: gewaehlterTitel.titel,
				autor: gewaehlterTitel.autor || ''
			}));
		}

		const offsetCount = Math.max(0, startFeld - 1);
		const leereFelder = Array.from({ length: offsetCount }, () => ({ isBlank: true }));

		return [...leereFelder, ...etikettenListe];
	});

	async function ladeKlassen() {
		try {
			const res = await apiFetch('/api/class-books');
			if (res.ok) {
				const body = await res.json();
				if (body?.data) {
					klassen = body.data;
				}
			}
		} catch (err) {
			console.error('Fehler beim Laden der Klassengruppen:', err);
		}
	}

	function wechsleKlasse() {
		const group = klassen.find((g) => g.className === gewaehlteKlasse);
		if (group) {
			klassenTitel = group.books || [];
		} else {
			klassenTitel = [];
		}
		gewaehlterTitel = null;
		vorhandeneExemplare = [];
		ausgesondertAnzahl = 0;
		exemplareNichtGeladen = false;
		exemplarSuche = '';
		exemplarLauf++;
		laedtExemplare = false;
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
				? vorhandeneExemplare.find((c) => String(c.barcode_id).toLowerCase() === gesucht)
				: undefined;
			if (exemplar) {
				exemplar.checked = true;
				exemplarSuche = '';
				return true;
			}
		}
		return false;
	}

	function verarbeiteSuche() {
		if (suchVerzoegerung) clearTimeout(suchVerzoegerung);
		const lauf = ++suchLauf;
		if (!suche.trim()) {
			treffer = [];
			sucht = false;
			return;
		}
		sucht = true;
		suchVerzoegerung = setTimeout(async () => {
			try {
				// Titel-Tür (view_books) — nie POST /api/action (dort löst ein gescannter
				// B-Barcode ohne aktiven Schüler eine Rückgabe aus, Inventur 03.09.2026) und seit
				// 05.09.2026 auch nicht mehr die Theken-Suche /api/search: Die lieferte
				// Schüler-Kiosk-Daten mit, die dieser Bildschirm nie zeigt, und hing am
				// Theken-Recht perform_actions, das ein Druckbildschirm nicht braucht.
				const res = await apiFetch(
					`/api/buecher/titel/suche?q=${encodeURIComponent(suche.trim())}`
				);
				if (res.ok) {
					const body = await res.json();
					if (lauf !== suchLauf) return;
					treffer = body.books || [];
				} else if (lauf === suchLauf) {
					// Sweep „verschluckte Fehlantwort" (06.09.2026): Vorher blieben die
					// Treffer des VORIGEN Suchtextes stehen — man klickt auf eine Zeile,
					// die zu einer anderen Eingabe gehört, und druckt deren Etikett.
					treffer = [];
					snackbars.addToast('Titelsuche fehlgeschlagen — bitte erneut versuchen.', 'error');
				}
			} catch (err) {
				if (lauf !== suchLauf) return;
				treffer = [];
				console.error('Fehler bei Buchtitelsuche:', err);
			} finally {
				if (lauf === suchLauf) sucht = false;
			}
		}, 300);
	}

	/** @param {any} titleObj */
	async function waehleTitel(titleObj) {
		gewaehlterTitel = titleObj;
		if (suchVerzoegerung) clearTimeout(suchVerzoegerung);
		suchLauf++;
		sucht = false;
		treffer = [];
		suche = titleObj.titel;
		gewaehlteKlasse = '';
		klassenTitel = [];
		await ladeVorhandeneExemplare();
	}

	async function ladeVorhandeneExemplare() {
		if (!gewaehlterTitel) return;
		const lauf = ++exemplarLauf;
		laedtExemplare = true;
		exemplarSuche = '';
		ausgesondertAnzahl = 0;
		exemplareNichtGeladen = false;
		try {
			const res = await apiFetch(`/api/buecher/titel/${gewaehlterTitel.id}/exemplare`);
			if (res.ok) {
				/** @type {any[]} */
				const alle = (await res.json()) || [];
				if (lauf !== exemplarLauf) return;
				vorhandeneExemplare = alle
					.filter((c) => !c.ist_ausgesondert)
					.map((c) => ({ ...c, checked: true }));
				ausgesondertAnzahl = alle.length - vorhandeneExemplare.length;
			} else if (lauf === exemplarLauf) {
				vorhandeneExemplare = [];
				exemplareNichtGeladen = true;
			}
		} catch (err) {
			if (lauf !== exemplarLauf) return;
			console.error('Fehler beim Laden der Exemplare:', err);
			vorhandeneExemplare = [];
			exemplareNichtGeladen = true;
		} finally {
			if (lauf === exemplarLauf) laedtExemplare = false;
		}
	}

	async function drucke() {
		const zuDrucken = bogen.filter((l) => !l.isBlank);
		if (zuDrucken.length === 0) return;

		try {
			const res = await apiFetch('/api/print/labels', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					formatId: formatId,
					startPosition: startFeld,
					isQR: barcodeArt === 'qr',
					items: zuDrucken.map((l) => ({
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
				await vermerkeGedruckt(zuDrucken);
			} else {
				console.error('Fehler beim Erstellen des PDFs');
				snackbars.addToast('Fehler beim Erstellen des PDFs', 'error');
			}
		} catch (err) {
			console.error('Netzwerkfehler beim Drucken', err);
			snackbars.addToast('Fehler beim Senden der Daten', 'error');
		}
	}

	return /** @satisfies {object} */ ({
		get suche() {
			return suche;
		},
		set suche(v) {
			suche = v;
		},
		get treffer() {
			return treffer;
		},
		get sucht() {
			return sucht;
		},

		get klassen() {
			return klassen;
		},
		get gewaehlteKlasse() {
			return gewaehlteKlasse;
		},
		set gewaehlteKlasse(v) {
			gewaehlteKlasse = v;
		},
		get klassenTitel() {
			return klassenTitel;
		},

		get gewaehlterTitel() {
			return gewaehlterTitel;
		},
		get barcodeArt() {
			return barcodeArt;
		},
		set barcodeArt(v) {
			barcodeArt = v;
		},
		get mitRahmen() {
			return mitRahmen;
		},
		set mitRahmen(v) {
			mitRahmen = v;
		},
		get formatId() {
			return formatId;
		},
		set formatId(v) {
			formatId = v;
		},
		get startFeld() {
			return startFeld;
		},
		set startFeld(v) {
			startFeld = v;
		},

		get modus() {
			return modus;
		},
		set modus(v) {
			modus = v;
		},
		get vorhandeneExemplare() {
			return vorhandeneExemplare;
		},
		set vorhandeneExemplare(v) {
			vorhandeneExemplare = v;
		},
		get laedtExemplare() {
			return laedtExemplare;
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
		get neueAnzahl() {
			return neueAnzahl;
		},
		set neueAnzahl(v) {
			neueAnzahl = v;
		},
		get neueStartnummer() {
			return neueStartnummer;
		},
		set neueStartnummer(v) {
			neueStartnummer = v;
		},

		get bogen() {
			return bogen;
		},
		get felderJeBogen() {
			return felderJeBogen;
		},

		ladeKlassen,
		wechsleKlasse,
		verarbeiteSuche,
		waehleTitel,
		ladeExemplare: ladeVorhandeneExemplare,
		setzeSichtbare,
		uebernimmNummer,
		drucke,

		leereDruckauftrag() {
			leereDruckwarteschlange();
		}
	});
}

export const etikettenStore = erzeugeEtikettenStore();
