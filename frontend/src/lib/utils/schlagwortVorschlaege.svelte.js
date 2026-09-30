import { ladeSchlagwortVorschlaege } from './schlagworte.js';

/**
 * Die Vorschläge eines Schlagwort-Felds (ui/ChipFeld): beim Öffnen die häufigsten, beim Tippen
 * die Treffer über alle Wörter (GET /api/schlagworte?suche=). EIN Baustein für beide Felder,
 * die Schlagworte entgegennehmen — Buchformular und Bestellkorb.
 *
 * Bis zum 30.09.2026 luden beide Felder nur die 500 häufigsten Wörter. Mit den Littera-Wörtern
 * (Katalogisat vom Juni 2026: 13.224) deckten sie 56 % der Zuordnungen ab; 12.724 Wörter wurden
 * nie angeboten, und wer „Pilz" tippte, sah das vorhandene „Pilze" nicht und legte ein zweites
 * Wort an (Rasterdurchgang vom 30.09.2026). Dasselbe Muster wie die Suche der Pflegeseite
 * (settings/schlagwortPflegeListe.svelte.js): 300 ms nach dem Tippen, nur die jüngste Antwort
 * schreibt. Ein leeres Feld holt wieder die häufigsten.
 *
 * Antwortet der Server nicht, bleibt die bisherige Liste stehen: Das Feld nimmt weiter freien
 * Text an, es fehlen nur die neuen Vorschläge.
 */
export function erzeugeSchlagwortVorschlaege() {
	/** @type {{ wert: string, beschreibung: string }[]} */
	let liste = $state([]);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	let laufNr = 0;

	/** @param {string} [text] */
	async function lade(text = '') {
		clearTimeout(timer);
		const meine = ++laufNr;
		const antwort = await ladeSchlagwortVorschlaege(text.trim());
		if (meine !== laufNr || antwort === null) return; // eine jüngere ist unterwegs oder da
		liste = antwort;
	}

	return {
		get liste() {
			return liste;
		},
		lade,
		/** Für ChipFeld ontippen: fragt 300 ms nach dem letzten Tastendruck nach.
		 *  @param {string} text */
		getippt(text) {
			clearTimeout(timer);
			timer = setTimeout(() => lade(text), 300);
		}
	};
}
