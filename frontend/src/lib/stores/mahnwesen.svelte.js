// stores/mahnwesen.svelte.js
// Status- und Logikverwaltung für das Mahnwesen (Svelte 5 Runes)

import { apiFetch } from '../apiFetch.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { useMahnwesenMail } from './mahnwesenMail.svelte.js';
import { mahnungenJeKind } from '../mahnungen.js';
import { SvelteSet } from 'svelte/reactivity';

/**
 * Höchste Überfälligkeit (Tage) über alle Medien eines Schülers.
 * @param {any[]} medien
 * @returns {number}
 */
function maxTageUeberfaellig(medien) {
	let maxTage = 0;
	for (const m of medien) {
		if (m.tage_ueberfaellig > maxTage) maxTage = m.tage_ueberfaellig;
	}
	return maxTage;
}

/**
 * Reiter eines Schülers nach Überfälligkeit. (Die frühere Sonderstufe
 * 'Lehrerkollegium' für klasse='lehrer' ist mit Migration 072 gefallen —
 * Lehrkräfte sind keine Schülerzeilen mehr.)
 * @param {number} maxTage
 * @returns {string}
 */
function berechneMahnstufe(maxTage) {
	// Diese Liste enthält ausschließlich überfällige Ausleihen (rueckgabe_frist < now).
	// tage_ueberfaellig kann bei <24 h Überfälligkeit rechnerisch 0 sein — das ist dann
	// „gerade fällig", NICHT „erledigt". Daher gibt es hier bewusst kein 'Erledigt'.
	if (maxTage > 14) return 'Mahnung';
	return '1. Erinnerung';
}

/**
 * Creates the central Mahnwesen store.
 */
function erzeugeMahnwesen() {
	let daten = $state(/** @type {{ klassen: any[] } | null} */ (null));
	let laedt = $state(true);
	let fehler = $state(/** @type {string|null} */ (null));

	let heuteRetourniert = $state(0);

	// Filter und Auswahl
	let offeneKlassen = /** @type {Set<string>} */ (new SvelteSet());
	let gewaehlteKlasse = $state(''); // Klassenfilter (leer = alle)
	let suche = $state(''); // Freitextsuche über Name/Klasse

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

	// Flache Liste für MD3 Table
	let flatSchueler = $derived(() => {
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

	let gefilterteSchueler = $derived(() => {
		let liste = flatSchueler();
		if (stufenFilter !== 'Alle')
			liste = liste.filter((/** @type {any} */ s) => s.mahnstufe === stufenFilter);
		if (gewaehlteKlasse)
			liste = liste.filter((/** @type {any} */ s) => s.klasse === gewaehlteKlasse);
		const q = suche.trim().toLowerCase();
		if (q) {
			liste = liste.filter(
				(/** @type {any} */ s) =>
					(s.name || '').toLowerCase().includes(q) || (s.klasse || '').toLowerCase().includes(q)
			);
		}
		// Dringlichkeit zuerst: am längsten überfällig oben, dann alphabetisch. So arbeitet man
		// die Liste von oben nach unten ab.
		return [...liste].sort(
			(/** @type {any} */ a, /** @type {any} */ b) =>
				b.maxTage - a.maxTage || String(a.name).localeCompare(String(b.name), 'de')
		);
	});

	/**
	 * Fetches Mahnwesen data from API.
	 */
	async function lade() {
		laedt = true;
		fehler = null;
		try {
			const res = await apiFetch('/api/mahnwesen');
			if (!res.ok) throw new Error((await res.text()) || 'Fehler beim Laden');
			const json = await res.json();
			daten = json;
			heuteRetourniert = json.heute_retourniert || 0;
			gewaehlteKennungen.clear();
		} catch (e) {
			fehler = String(e);
		} finally {
			laedt = false;
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
		for (const s of gefilterteSchueler()) gewaehlteKennungen.add(s.schueler_id);
	}

	function waehleKeine() {
		gewaehlteKennungen.clear();
	}

	/** Wrapper for bulk printing that supplies needed state context. */
	async function druckeGewaehlteMahnungen() {
		await pdfStore.druckeGewaehlteMahnungen(gewaehlteKennungen, gefilterteSchueler, lade);
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
		get heuteRetourniert() {
			return heuteRetourniert;
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
			return gefilterteSchueler();
		},

		lade,
		schalteKlasse,
		schalteAuswahl,
		waehleAlle,
		waehleKeine,

		// Mail: nur noch der Massenversand (Einzelversand je Klasse entfernt am 18.09.2026,
		// siehe mahnwesenMail.svelte.js).
		sendeMahnmails: mailStore.sendeMahnmails,

		get drucktPdf() {
			return pdfStore.drucktPdf;
		},
		druckeGewaehlteMahnungen
	});
}

export const mahnwesenStore = erzeugeMahnwesen();
