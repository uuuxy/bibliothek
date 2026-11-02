import { apiGet } from '../../apiFetch.js';

/**
 * Lädt die geöffnete Bestellung. Es gilt die jüngste Anfrage: Ein Signal der Leitung und das Laden
 * nach einer eigenen Aktion können sich überholen, und die ältere Antwort nennte dann eine
 * überholte Frist des Links. Wer auf das Laden wartet, etwa um danach den Fokus auf die Quittung zu
 * setzen, wartet bis zum jüngsten Abruf; bis dahin stünde der alte Stand noch da.
 * @param {() => string} bestellungId
 */
export function erzeugeBestellLader(bestellungId) {
	/** @type {any} */
	let bestellung = $state(null);
	let laedt = $state(true);
	let fehler = $state('');
	let ladeNr = 0;
	/** @type {Promise<boolean>} */
	let juengsterAbruf = Promise.resolve(true);

	/** @param {number} nr */
	async function hole(nr) {
		// Beim Aktualisieren bleibt die Ansicht stehen, auch wenn der Abruf scheitert (die Meldung
		// zeigt apiGet): Baute sie sich neu auf, wäre der Link weg, der nur einmal zu sehen ist.
		const ersterAbruf = !bestellung;
		laedt = ersterAbruf;
		fehler = '';
		try {
			const antwort = await apiGet(`/api/bestellhistorie/${bestellungId()}`);
			if (nr === ladeNr) bestellung = antwort;
			return true;
		} catch (e) {
			// Ein veralteter Verweis liefert 404. Die Meldung nennt den Rückweg, statt eine
			// leere Seite stehen zu lassen, die aussieht wie eine leere Bestellung.
			if (ersterAbruf && nr === ladeNr) {
				fehler = /** @type {any} */ (e)?.message || 'Bestellung konnte nicht geladen werden.';
			}
			return false;
		} finally {
			if (nr === ladeNr) laedt = false;
		}
	}

	/** @returns {Promise<boolean>} ob die Antwort kam (für abonniereNeuladen) */
	function laden() {
		const nr = ++ladeNr;
		const abruf = hole(nr).then((kam) => (nr === ladeNr ? kam : juengsterAbruf));
		juengsterAbruf = abruf;
		return abruf;
	}

	return {
		get bestellung() {
			return bestellung;
		},
		get laedt() {
			return laedt;
		},
		get fehler() {
			return fehler;
		},
		laden
	};
}
