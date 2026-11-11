import { apiFetch, leseApiFehler } from '../../apiFetch.js';

/** @typedef {{ zeitraum: string, medientyp: string }} StatistikFilter */

/**
 * Die Anfrage an `/api/statistiken` zur Auswahl der Übersicht. Übersicht und Detailseite bauen
 * sie hier, damit „Alle anzeigen" dieselben Titel zeigt wie die Karte davor.
 * @param {StatistikFilter} filter
 */
export function statistikParameter({ zeitraum, medientyp }) {
	// eslint-disable-next-line svelte/prefer-svelte-reactivity
	const params = new URLSearchParams({ limit: '100' });
	if (zeitraum !== 'all') params.set('zeitraum', zeitraum);
	if (medientyp) params.set('type', medientyp);
	return params;
}

const ZEITRAUM_SATZ = /** @type {Record<string, string>} */ ({
	schuljahr: ' im laufenden Schuljahr',
	monat: ' in den letzten 30 Tagen'
});
const BESTAND_SATZ = /** @type {Record<string, string>} */ ({
	freihand: ', nur Freihand',
	lmf: ', nur LMF'
});

/**
 * Der Satz unter der Überschrift der Detailseite: was die Liste zeigt, mit der Auswahl, die sie
 * geladen hat. Die Ladenhüter hängen nicht am Zeitraum (zwei Jahre ohne Ausleihe).
 * @param {'renner' | 'ladenhueter'} art @param {StatistikFilter} filter
 */
export function statistikSatz(art, { zeitraum, medientyp }) {
	const bestand = BESTAND_SATZ[medientyp] ?? '';
	if (art === 'ladenhueter')
		return `Seit über 2 Jahren nicht ausgeliehen — oder noch nie${bestand}.`;
	return `Nach Anzahl der Ausleihen${ZEITRAUM_SATZ[zeitraum] ?? ', alle Zeiträume'}${bestand}.`;
}

/**
 * Der eine Abruf der Statistik (`/api/statistiken`) für Übersicht und Detailseite —
 * mit Reihenfolge-Schutz und Fehlerzustand.
 *
 * `laufNr` verwirft, was überholt wurde (wie useLeserakte): Sonst stünden nach einem
 * Wechsel des Zeitraums unter der neuen Überschrift die Zahlen des alten. `fehler` gibt der
 * Seite den Zustand, den sie mit LadeFehler zeigt, statt einer leeren Seite, die wie „keine
 * Daten" aussieht.
 */
export function statistikAbruf() {
	/** @type {any} */
	let daten = $state(null);
	let laedt = $state(true);
	let fehler = $state(false);
	let laufNr = 0;

	/** @param {URLSearchParams | string} params */
	async function laden(params) {
		const meine = ++laufNr;
		laedt = true;
		fehler = false;
		try {
			const res = await apiFetch(`/api/statistiken?${params}`);
			if (meine !== laufNr) return; // eine jüngere Anfrage ist unterwegs oder da
			if (!res.ok) throw new Error(await leseApiFehler(res));
			const json = await res.json();
			if (meine !== laufNr) return;
			daten = json;
		} catch (err) {
			if (meine !== laufNr) return;
			console.error('Statistik laden fehlgeschlagen:', err);
			daten = null;
			fehler = true;
		} finally {
			if (meine === laufNr) laedt = false;
		}
	}

	return /** @satisfies {object} */ ({
		get daten() {
			return daten;
		},
		get loading() {
			return laedt;
		},
		get fehler() {
			return fehler;
		},
		/**
		 * Ob der Server einen Abschnitt nicht lesen konnte (`nicht_gelesen`). Er steht dann leer
		 * in der Antwort und ist kein leerer Bestand.
		 * @param {'renner' | 'ladenhueter' | 'kennzahlen' | 'monats_trend'} abschnitt
		 */
		fehlt(abschnitt) {
			return daten?.nicht_gelesen?.includes(abschnitt) ?? false;
		},
		laden
	});
}
