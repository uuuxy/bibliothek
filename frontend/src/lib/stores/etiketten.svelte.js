// stores/etiketten.svelte.js
// Status- und Logikverwaltung für den Etikettendruck (Svelte 5 Runes)

import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { felderProBogen } from '../components/etiketten/etikettformate.js';
import { druckWarteschlange, leereDruckwarteschlange } from './druckWarteschlange.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import { normalisiereScan } from '../utils/scanEinordnen.js';
import { dekodiereLitteraEtikett } from '../utils/litteraEtikett.js';
import { ohnePruefzeichen } from '../utils/code39Pruefzeichen.js';
import { druckeBogen } from '../components/etiketten/etikettenDruck.js';

function erzeugeEtikettenStore() {
	let suche = $state('');
	let treffer = $state.raw(/** @type {any[]} */ ([]));
	let sucht = $state(false);

	let klassen = $state.raw(/** @type {any[]} */ ([]));
	let gewaehlteKlasse = $state('');
	let klassenTitel = $state.raw(/** @type {any[]} */ ([]));

	let gewaehlterTitel = $state(/** @type {any} */ (null));
	let barcodeArt = $state('code39'); // "code39" | "qr"
	let startFeld = $state(1); // 1 to 21

	// Vorgabe = die des Servers (pdf/etikett_formate.go EtikettFormatVorgabe): das Papier des
	// Lieferantenwegs, eine Vorgabe für beide.
	let formatId = $state('zweckform_l4760');
	// Aus der gemeinsamen Formatliste, nicht aus einer eigenen Zahlenkette, die eine weitere
	// Kopie derselben Raster wäre.
	let felderJeBogen = $derived(felderProBogen(formatId));
	// Tief reaktiv, nicht $state.raw: Das Kästchen in Schritt 2 schreibt `checked` an das
	// Exemplar selbst, und der Bogen (bogen) muss das sehen.
	let vorhandeneExemplare = $state(/** @type {any[]} */ ([]));
	// Ausgesonderte Exemplare stehen nicht mehr im Regal und deshalb nicht in der Liste; ihre
	// Zahl steht als Hinweis darunter. Dieselbe Regel wie repository.EtikettOffenBedingung.
	let ausgesondertAnzahl = $state(0);
	let laedtExemplare = $state(false);
	// Ein gescheiterter Abruf ist ein eigener Zustand mit seinem Grund: Eine leere Liste hieße
	// „kein Exemplar", und der Bogen bliebe leer, ohne dass jemand den Grund sieht.
	let exemplarFehler = $state('');
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

	let suchVerzoegerung = /** @type {any} */ (null);
	// Zählt die Titelsuchen wie exemplarLauf die Abrufe der Exemplare.
	let suchLauf = 0;

	/** @type {Array<{isBlank?: boolean, barcode_id?: string, titel?: string, autor?: string}>} */
	let bogen = $derived.by(() => {
		const offsetCount = Math.max(0, startFeld - 1);
		const leereFelder = Array.from({ length: offsetCount }, () => ({ isBlank: true }));
		if ((druckWarteschlange.copies?.length ?? 0) > 0) {
			const copies = /** @type {any[]} */ (druckWarteschlange.copies);
			const etikettenListe = copies.map((c) => ({
				barcode_id: c.barcode_id,
				titel: c.titel,
				autor: c.autor || ''
			}));
			return [...leereFelder, ...etikettenListe];
		}

		if (!gewaehlterTitel) return [];

		const etikettenListe = vorhandeneExemplare
			.filter((c) => c.checked)
			.map((c) => ({
				barcode_id: c.barcode_id,
				titel: gewaehlterTitel.titel,
				autor: gewaehlterTitel.autor || ''
			}));
		return [...leereFelder, ...etikettenListe];
	});

	async function ladeKlassen() {
		try {
			const res = await apiFetch('/api/class-books');
			if (res.ok) klassen = await res.json();
		} catch (err) {
			console.error('Fehler beim Laden der Klassengruppen:', err);
		}
	}

	function wechsleKlasse() {
		klassenTitel = klassen.find((g) => g.className === gewaehlteKlasse)?.books || [];
		gewaehlterTitel = null;
		vorhandeneExemplare = [];
		ausgesondertAnzahl = 0;
		exemplarFehler = '';
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
	 * genau dieser Nummer auf den Bogen und leert das Feld. Gelesen wird wie an der Theke: die
	 * Nummer, ein altes LMF-Etikett, das Littera-Etikett, ein Etikett mit Prüfzeichen. Ein Teil
	 * wählt nichts. Groß und klein zählen nicht: Das Feld sucht nur Nummern dieses Titels.
	 * @returns {boolean} ob ein Exemplar dieses Titels gefunden wurde
	 */
	function uebernimmNummer() {
		const scan = normalisiereScan(exemplarSuche);
		if (!scan) return false;
		const ueberLMF = vorhandeneExemplare.find((c) => c.lmf_barcode === scan)?.barcode_id;
		for (const nummer of [scan, ueberLMF, dekodiereLitteraEtikett(scan), ohnePruefzeichen(scan)]) {
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
				// Titel-Tür (view_books), nie POST /api/action (dort löst ein gescannter
				// B-Barcode ohne aktiven Schüler eine Rückgabe aus) und nicht die Theken-Suche
				// /api/search: Die liefert Schüler-Kiosk-Daten mit, die dieser Bildschirm nie
				// zeigt, und hängt am Theken-Recht perform_actions, das ein Druckbildschirm nicht
				// braucht.
				const res = await apiFetch(
					`/api/buecher/titel/suche?q=${encodeURIComponent(suche.trim())}`
				);
				if (res.ok) {
					const body = await res.json();
					if (lauf !== suchLauf) return;
					treffer = body.books || [];
				} else if (lauf === suchLauf) {
					// Sonst blieben die Treffer des vorigen Suchtextes stehen: Man klickte auf
					// eine Zeile, die zu einer anderen Eingabe gehört, und druckte deren Etikett.
					treffer = [];
					const grund = await leseApiFehler(
						res,
						'Titelsuche fehlgeschlagen — bitte erneut versuchen.'
					);
					if (lauf === suchLauf) snackbars.addToast(grund, 'error');
				}
			} catch (err) {
				if (lauf !== suchLauf) return;
				treffer = [];
				// Ohne Antwort hieße das leere Feld „kein Treffer"; der Grund steht in der Meldung.
				snackbars.addToast(fehlertext(err), 'error');
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
		exemplarFehler = '';
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
			} else {
				const grund = await leseApiFehler(res);
				if (lauf !== exemplarLauf) return;
				vorhandeneExemplare = [];
				exemplarFehler = grund;
			}
		} catch (err) {
			if (lauf !== exemplarLauf) return;
			vorhandeneExemplare = [];
			exemplarFehler = fehlertext(err);
		} finally {
			if (lauf === exemplarLauf) laedtExemplare = false;
		}
	}

	/** Druckt den Bogen. */
	function drucke() {
		const bogenDaten = { formatId, startFeld, alsQR: barcodeArt === 'qr' };
		return druckeBogen(
			bogenDaten,
			bogen.filter((l) => !l.isBlank)
		);
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

		get vorhandeneExemplare() {
			return vorhandeneExemplare;
		},
		set vorhandeneExemplare(v) {
			vorhandeneExemplare = v;
		},
		get laedtExemplare() {
			return laedtExemplare;
		},
		get exemplarFehler() {
			return exemplarFehler;
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
