import { ladeSchlagwortVorschlaege } from './schlagworte.js';

/**
 * Die Vorschläge eines Schlagwort-Felds (ui/ChipFeld): beim Öffnen die häufigsten, beim Tippen
 * die Treffer über alle Wörter (GET /api/schlagworte?suche=). Ein Baustein für beide Felder,
 * die Schlagworte entgegennehmen, Buchformular und Bestellkorb.
 *
 * Über alle Wörter, nicht nur die häufigsten: Mit den rund 13.000 Littera-Wörtern deckten
 * die 500 häufigsten nur gut die Hälfte der Zuordnungen ab, und wer „Pilz" tippt, sähe das
 * vorhandene „Pilze" nicht und legte ein zweites Wort an. Dasselbe Muster wie die Suche der
 * Pflegeseite
 * (einstellungen/schlagwortPflegeListe.svelte.js): 300 ms nach dem Tippen, nur die jüngste Antwort
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

	return /** @satisfies {object} */ ({
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
	});
}
