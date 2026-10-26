import { apiFetch, leseApiFehler } from '../../apiFetch.js';

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
