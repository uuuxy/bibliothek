// stores/mahnwesen.svelte.js
// Status- und Logikverwaltung für das Mahnwesen (Svelte 5 Runes)

import { apiFetch, leseApiFehler } from '../apiFetch.js';
import { fehlertext } from '../utils/fehlertext.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { useMahnwesenMail } from './mahnwesenMail.svelte.js';
import {
	mahnungenJeKind,
	maxTageUeberfaellig,
	berechneMahnstufe
} from '../components/mahnwesen/mahnungen.js';
import { passtZurSuche } from '../utils/suchnorm.js';
import { SvelteSet } from 'svelte/reactivity';

/**
 * Creates the central Mahnwesen store.
 */
function erzeugeMahnwesen() {
	let daten = $state(/** @type {{ klassen: any[] } | null} */ (null));
	let laedt = $state(true);
	let fehler = $state(/** @type {string|null} */ (null));

	// Filter und Auswahl
	let offeneKlassen = /** @type {Set<string>} */ (new SvelteSet());
	let gewaehlteKlasse = $state(''); // Klassenfilter (leer = alle)
	let suche = $state(''); // Freitextsuche über Name, Klasse und Ausweisnummer

	// MD3 Filter & Bulk Actions
	let stufenFilter = $state('Alle'); // "Alle", "1. Erinnerung", "Mahnung"
	let gewaehlteKennungen = /** @type {Set<string>} */ (new SvelteSet());

	const pdfStore = useMahnwesenPdf();
	const mailStore = useMahnwesenMail();

	// Abgeleitete Werte
	let klassen = $derived(daten?.klassen ?? []);
	// Die Gruppe der Ehemaligen ist keine Klasse; Versand und Klassendruck bieten sie nicht an.
	let versandKlassen = $derived(klassen.filter((/** @type {any} */ k) => !k.ehemalige));
	let ueberfaelligGesamt = $derived(
		klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) =>
				sum +
				k.schueler.reduce(
					(/** @type {number} */ s2, /** @type {any} */ sch) => s2 + sch.medien.length,
					0
				),
			0
		)
	);

	// $derived.by merkt die Listen selbst: Tabelle, Kopfzeile und Druck lesen sie oft je Darstellung.
	let flatSchueler = $derived.by(() => {
		let liste = [];
		for (const k of klassen) {
			for (const s of k.schueler) {
				const maxTage = maxTageUeberfaellig(s.medien);
				const mahnstufe = berechneMahnstufe(maxTage);
				liste.push({
					...s,
					maxTage,
					mahnstufe,
					...mahnungenJeKind(s.medien),
					lehrer_email: k.lehrer_email
				});
			}
		}
		return liste;
	});

	let gefilterteSchueler = $derived.by(() => {
		let liste = flatSchueler;
		if (stufenFilter !== 'Alle')
			liste = liste.filter((/** @type {any} */ s) => s.mahnstufe === stufenFilter);
		if (gewaehlteKlasse)
			liste = liste.filter((/** @type {any} */ s) => s.klasse === gewaehlteKlasse);
		// Dieselbe Regel wie an der Theke: „ozturk" findet Öztürk, „alkahil" Al-Kahil.
		if (suche.trim())
			liste = liste.filter((s) => passtZurSuche(suche, s.name, s.klasse, s.barcode_id));
		// Dringlichkeit zuerst: am längsten überfällig oben, dann alphabetisch. So arbeitet man
		// die Liste von oben nach unten ab.
		return [...liste].sort(
			(/** @type {any} */ a, /** @type {any} */ b) =>
				b.maxTage - a.maxTage || `${a.name}`.localeCompare(`${b.name}`, 'de', { numeric: true })
		);
	});

	let laufNr = 0;
	/**
	 * Lädt die Liste. Eine überholte Antwort wird verworfen, sonst überschriebe ein langsamer
	 * früherer Abruf den neueren Stand. still: das Nachladen nach einer Rückgabe an einem anderen
	 * Platz; es behält die Auswahl, soweit die Kinder noch in der Liste stehen, zeigt keinen
	 * Ladekreis und meldet ein Scheitern dem Aufrufer (false), der es später neu versucht.
	 * @param {{ still?: boolean }} [art]
	 */
	async function lade({ still = false } = {}) {
		const lauf = ++laufNr;
		if (!still) {
			laedt = true;
			fehler = null;
		}
		try {
			const res = await apiFetch('/api/mahnwesen');
			if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Laden'));
			const neu = await res.json();
			if (lauf !== laufNr) return true;
			daten = neu;
			const da = new Set(
				neu.klassen.flatMap((/** @type {any} */ k) =>
					k.schueler.map((/** @type {any} */ s) => s.schueler_id)
				)
			);
			for (const id of gewaehlteKennungen) if (!still || !da.has(id)) gewaehlteKennungen.delete(id);
			return true;
		} catch (e) {
			if (lauf === laufNr && !still) fehler = fehlertext(e);
			return false;
		} finally {
			if (lauf === laufNr) laedt = false;
		}
	}

	/** @param {string} klasse */
	function schalteKlasse(klasse) {
		const s = new SvelteSet(offeneKlassen);
		if (s.has(klasse)) s.delete(klasse);
		else s.add(klasse);
		offeneKlassen = s;
	}

	// WICHTIG: gewaehlteKennungen ist ein reaktives SvelteSet, aber KEIN $state-Binding. Deshalb muss
	// es IN-PLACE mutiert werden (add/delete/clear) — eine Neuzuweisung (gewaehlteKennungen = neuesSet)
	// wäre nicht reaktiv, die Checkboxen und die Auswahl-Toolbar würden nicht reagieren.
	/** @param {string} id */
	function schalteAuswahl(id) {
		if (gewaehlteKennungen.has(id)) gewaehlteKennungen.delete(id);
		else gewaehlteKennungen.add(id);
	}

	function waehleAlle() {
		gewaehlteKennungen.clear();
		for (const s of gefilterteSchueler) gewaehlteKennungen.add(s.schueler_id);
	}

	function waehleKeine() {
		gewaehlteKennungen.clear();
	}

	/** Wrapper for bulk printing that supplies needed state context. */
	async function druckeGewaehlteMahnungen() {
		await pdfStore.druckeGewaehlteMahnungen(gewaehlteKennungen, () => gefilterteSchueler, lade);
	}

	return /** @satisfies {object} */ ({
		get daten() {
			return daten;
		},
		get laedt() {
			return laedt;
		},
		get fehler() {
			return fehler;
		},
		get offeneKlassen() {
			return offeneKlassen;
		},
		get gewaehlteKlasse() {
			return gewaehlteKlasse;
		},
		set gewaehlteKlasse(v) {
			gewaehlteKlasse = v;
			// Sichtbereich ändert sich → Auswahl leeren, damit sie stets zur Liste passt.
			gewaehlteKennungen.clear();
		},
		get suche() {
			return suche;
		},
		set suche(v) {
			suche = v;
			gewaehlteKennungen.clear();
		},
		get klassen() {
			return klassen;
		},
		get versandKlassen() {
			return versandKlassen;
		},
		get ueberfaelligGesamt() {
			return ueberfaelligGesamt;
		},
		get stufenFilter() {
			return stufenFilter;
		},
		set stufenFilter(v) {
			stufenFilter = v;
			gewaehlteKennungen.clear();
		},
		get gewaehlteKennungen() {
			return gewaehlteKennungen;
		},
		get gefilterteSchueler() {
			return gefilterteSchueler;
		},

		lade,
		schalteKlasse,
		schalteAuswahl,
		waehleAlle,
		waehleKeine,

		// Mail: nur der Mahnlauf, mit seinem Laufzustand (mahnwesenMail.svelte.js).
		mahnlauf: mailStore,

		get drucktPdf() {
			return pdfStore.drucktPdf;
		},
		druckeGewaehlteMahnungen
	});
}

export const mahnwesenStore = erzeugeMahnwesen();
