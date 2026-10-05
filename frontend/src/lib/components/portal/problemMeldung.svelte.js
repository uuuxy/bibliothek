import { apiFetch } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';

/** Schlüssel des Formulars, das zu keinem Treffer gehört. */
export const OHNE_BUCH = 'ohne-buch';

/**
 * @typedef {{ open: boolean, worum: string, klasse: string, text: string, sending: boolean }} MeldeFormular
 */

/**
 * Formularzustand und Absenden von „Problem melden" — je Treffer ein Formular, dazu eines
 * ohne Buch (OHNE_BUCH). Der Zustand liegt hier und nicht in der Karte: Ändert sich die
 * Trefferliste, während jemand schreibt, hinge die Eingabe sonst an einer Karte, die es
 * nicht mehr gibt. Zustand in der Fabrik, kein Modul-Singleton — auf einem geteilten
 * Rechner darf die angefangene Meldung des vorigen Bedieners nicht in der nächsten Sitzung
 * stehen.
 *
 * @param {() => void | Promise<void>} nachSenden liest die eigenen Meldungen neu
 */
export function erzeugeProblemMeldung(nachSenden) {
	/** @type {Record<string, MeldeFormular>} */
	let forms = $state({});

	/** @returns {MeldeFormular} */
	const leer = () => ({ open: false, worum: '', klasse: '', text: '', sending: false });

	/**
	 * Legt das Formular an, falls es fehlt. Nur aus Ereignissen aufrufen: Eine Zuweisung an
	 * $state während des Renderns bricht es ab.
	 * @param {string} key
	 */
	function ensure(key) {
		if (!forms[key]) forms[key] = leer();
		return forms[key];
	}

	return {
		/** Lese-Sicht fürs Template, legt nichts an. @param {string} key */
		form(key) {
			return forms[key] ?? leer();
		},

		/** @param {string} key */
		oeffne(key) {
			ensure(key).open = true;
		},

		/** Schließt das Formular; die Eingabe bleibt stehen. @param {string} key */
		schliesse(key) {
			if (forms[key]) forms[key].open = false;
		},

		/**
		 * Schickt die Meldung. Am Treffer ist `titel` das Buch; ohne Buch nennt das Feld
		 * „Worum geht es?" den Gegenstand. Ohne Gegenstand oder Beschreibung geht nichts raus.
		 *
		 * @param {string} key
		 * @param {string} [titel]
		 * @returns {Promise<boolean>} ob die Meldung angenommen wurde
		 */
		async senden(key, titel) {
			const f = ensure(key);
			// Am Treffer steht das Buch fest. Trägt es im Katalog keinen Titel, geht die
			// Meldung trotzdem raus, mit dem Wort, das auch die Karte zeigt.
			const titelText = titel === undefined ? f.worum.trim() : titel.trim() || 'Unbekannter Titel';
			if (f.sending || titelText === '' || f.text.trim() === '') return false;
			f.sending = true;
			try {
				const res = await apiFetch('/api/anliegen', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						art: 'meldung',
						titel_text: titelText,
						klasse: f.klasse.trim(),
						kommentar: f.text.trim()
					})
				});
				if (!res.ok) {
					const data = await res.json().catch(() => null);
					throw new Error(data?.error || 'Die Meldung konnte nicht gesendet werden.');
				}
				toastStore.addToast('Meldung ist bei der Bibliothek.', 'success');
				forms[key] = leer();
				await nachSenden();
				return true;
			} catch (err) {
				toastStore.addToast(/** @type {any} */ (err).message || String(err), 'error');
				return false;
			} finally {
				if (forms[key]) forms[key].sending = false;
			}
		}
	};
}
