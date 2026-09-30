import { ladeDnbSchlagwortVorschlag } from './schlagworte.js';
import { normalisiereIsbn as normiere } from './isbnFormen.js';

/**
 * Der Schlagwort-Vorschlag aus der DNB für EIN Schlagwort-Feld (entschieden am 30.09.2026): im
 * Buchformular und beim Nachbestellen eines Titels, den es schon gibt. Ein neuer Titel aus der
 * Bestellsuche bringt seinen Vorschlag schon mit (POST /api/buecher/aus-isbn).
 *
 * Der Vorschlag gilt für die ISBN, zu der er geholt wurde: Steht im Feld eine andere, liefern
 * liste und neu nichts, und der Knopf fragt neu. Nur die jüngste Antwort schreibt.
 *
 * status: '' (nicht gefragt) · 'laedt' · 'da' · 'unbekannt' (die DNB kennt die ISBN nicht) ·
 * 'fehler' (nicht erreichbar).
 */
export function erzeugeDnbSchlagwortVorschlag() {
	/** @type {'' | 'laedt' | 'da' | 'unbekannt' | 'fehler'} */
	let status = $state('');
	let fuer = $state('');
	/** @type {string[]} */
	let liste = $state([]);
	/** @type {string[]} */
	let neu = $state([]);
	let laufNr = 0;
	/** @param {string | null | undefined} isbn */
	const gilt = (isbn) => fuer !== '' && fuer === normiere(isbn);

	return {
		/** @param {string | null | undefined} isbn */
		status: (isbn) => (gilt(isbn) ? status : ''),
		/** Die Wörter der eigenen Liste. @param {string | null | undefined} isbn */
		liste: (isbn) => (gilt(isbn) ? liste : []),
		/** Die Normdatei-Wörter, die die Liste noch nicht kennt. @param {string | null | undefined} isbn */
		neu: (isbn) => (gilt(isbn) ? neu : []),
		/** @param {string | null | undefined} isbn */
		async lade(isbn) {
			const nummer = normiere(isbn);
			if (!nummer) return;
			const meine = ++laufNr;
			fuer = nummer;
			status = 'laedt';
			liste = [];
			neu = [];
			try {
				const antwort = await ladeDnbSchlagwortVorschlag(nummer);
				if (meine !== laufNr) return;
				liste = antwort.liste;
				neu = antwort.neu;
				status = antwort.dnbSatz ? 'da' : 'unbekannt';
			} catch {
				if (meine === laufNr) status = 'fehler';
			}
		}
	};
}
